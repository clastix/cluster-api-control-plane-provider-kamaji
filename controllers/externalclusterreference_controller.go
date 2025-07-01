// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"strings"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/indexers"
)

type ExternalClusterReferenceReconciler struct {
	Client         client.Client
	Store          externalclusterreference.Store
	TriggerChannel chan event.GenericEvent
}

//nolint:funlen,cyclop
func (r *ExternalClusterReferenceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	var secret corev1.Secret
	if err := r.Client.Get(ctx, req.NamespacedName, &secret); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Error(err, "unable to fetch Secret")

		return ctrl.Result{}, err //nolint:wrapcheck
	}
	//nolint:prealloc
	var keys []string

	for _, key := range externalclusterreference.GenerateKeyNameFromSecret(&secret) {
		var kcpList v1alpha1.KamajiControlPlaneList

		if err := r.Client.List(ctx, &kcpList, client.MatchingFields{indexers.ExternalClusterReferenceKamajiControlPlaneField: key}); err != nil {
			log.Error(err, "unable to use indexer", "key", key)

			return ctrl.Result{}, err //nolint:wrapcheck
		}

		if len(kcpList.Items) == 0 {
			if r.Store.Stop(key) {
				log.Info("stopping manager, unused")
			}

			continue
		}

		log.Info("secret entry is referenced", "key", key, "count", len(kcpList.Items))

		keys = append(keys, key)
	}

	for _, key := range keys {
		if _, found := r.Store.Get(key, secret.ResourceVersion); found {
			continue
		}

		if !r.Store.Stop(key) {
			log.Info("new configuration, loading manager")
		} else {
			log.Info("configuration seems changed, restarting manager")
		}

		cfg, cfgErr := clientcmd.RESTConfigFromKubeConfig(secret.Data[strings.Split(key, "/")[2]])
		if cfgErr != nil {
			log.Error(cfgErr, "cannot generate REST config from Secret content", "key", key)

			return ctrl.Result{}, cfgErr //nolint:wrapcheck
		}

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:  r.Client.Scheme(),
			Metrics: server.Options{BindAddress: "0"},
			Cache: cache.Options{
				ByObject: map[client.Object]cache.ByObject{
					// Reduce memory overhead by only caching watched resources.
					&kamajiv1alpha1.TenantControlPlane{}: {},
				},
			},
		})
		if err != nil {
			log.Error(err, "cannot generate manager")

			return ctrl.Result{}, err //nolint:wrapcheck
		}

		if err = (&PushKamajiChange{ParentClient: r.Client, Client: mgr.GetClient(), TriggerChannel: r.TriggerChannel}).SetupWithManager(mgr); err != nil {
			log.Error(err, "unable to create controller", "controller", "PushKamajiChange")

			return ctrl.Result{}, err
		}

		mgrCtx, cancelFn := context.WithCancel(ctx)
		go r.startManager(mgrCtx, mgr, key)

		r.Store.Add(key, secret.ResourceVersion, mgr, cancelFn)
	}

	return ctrl.Result{}, nil
}

func (r *ExternalClusterReferenceReconciler) startManager(ctx context.Context, mgr ctrl.Manager, name string) {
	if mgrErr := mgr.Start(ctx); mgrErr != nil {
		ctrllog.FromContext(ctx).Error(mgrErr, "manager cannot be started, external cluster reference could not work")

		r.Store.Stop(name)
	}
}

func (r *ExternalClusterReferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//nolint:wrapcheck
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Watches(&v1alpha1.KamajiControlPlane{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
			kcp := object.(*v1alpha1.KamajiControlPlane) //nolint:forcetypeassert
			if kcp.Spec.Deployment.ExternalClusterReference == nil {
				return nil
			}

			var requests []reconcile.Request

			for _, secret := range r.getSecretFromKamajiControlPlaneReferences(ctx, kcp) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: secret.Namespace,
						Name:      secret.Name,
					},
				})
			}

			return requests
		})).
		Complete(r)
}

func (r *ExternalClusterReferenceReconciler) getSecretFromKamajiControlPlaneReferences(ctx context.Context, kcp *v1alpha1.KamajiControlPlane) []corev1.Secret {
	var secretList corev1.SecretList

	val := externalclusterreference.GenerateKeyNameFromKamaji(kcp)

	if err := r.Client.List(ctx, &secretList, client.MatchingFields{indexers.ExternalClusterReferenceSecretField: val}); err != nil {
		return nil
	}

	return secretList.Items
}

type PushKamajiChange struct {
	ParentClient   client.Client
	Client         client.Client
	TriggerChannel chan event.GenericEvent
}

func (p *PushKamajiChange) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var tcp kamajiv1alpha1.TenantControlPlane

	if err := p.Client.Get(ctx, request.NamespacedName, &tcp); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err //nolint:wrapcheck
	}

	value := externalclusterreference.ParseKamajiControlPlaneUIDFromTenantControlPlane(tcp)
	if value == "" {
		return reconcile.Result{}, nil
	}

	var kcpList v1alpha1.KamajiControlPlaneList

	if err := p.ParentClient.List(ctx, &kcpList, client.MatchingFields{indexers.KamajiControlPlaneUIDField: value}); err != nil {
		return reconcile.Result{}, err //nolint:wrapcheck
	}

	for _, kcp := range kcpList.Items {
		p.TriggerChannel <- event.GenericEvent{
			Object: &v1alpha1.KamajiControlPlane{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kcp.Name,
					Namespace: kcp.Namespace,
				},
			},
		}
	}

	return reconcile.Result{}, nil
}

//nolint:wrapcheck
func (p *PushKamajiChange) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			SkipNameValidation: ptr.To(true),
		}).
		For(&kamajiv1alpha1.TenantControlPlane{}).
		Complete(p)
}
