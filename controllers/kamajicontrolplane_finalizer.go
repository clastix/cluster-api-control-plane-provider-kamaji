// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
)

func (r *KamajiControlPlaneReconciler) handleFinalizer(ctx context.Context, kcp *v1alpha1.KamajiControlPlane) error {
	finalizers := sets.New[string](kcp.Finalizers...)
	if !finalizers.Has(ExternalClusterReferenceFinalizer) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() (scopedErr error) { //nolint:nonamedreturns
			if scopedErr = r.client.Get(ctx, types.NamespacedName{Namespace: kcp.Namespace, Name: kcp.Name}, kcp); scopedErr != nil {
				return scopedErr //nolint:wrapcheck
			}

			finalizers.Insert(ExternalClusterReferenceFinalizer)

			kcp.SetFinalizers(finalizers.UnsortedList())

			return r.client.Update(ctx, kcp)
		})
		if err != nil {
			return err //nolint:wrapcheck
		}
	}

	return nil
}

func (r *KamajiControlPlaneReconciler) handleDeletion(ctx context.Context, kcp v1alpha1.KamajiControlPlane) error {
	finalizers, log := sets.New[string](kcp.Finalizers...), ctrllog.FromContext(ctx)

	if !finalizers.Has(ExternalClusterReferenceFinalizer) || kcp.Spec.Deployment.ExternalClusterReference == nil {
		log.Info("waiting for KamajiControlPlane finalizers")

		return nil
	}

	remoteClient, cErr := r.extractRemoteClient(ctx, kcp)
	if cErr != nil {
		log.Error(cErr, "cannot generate remote client for deletion")

		return cErr
	}

	var tcp kamajiv1alpha1.TenantControlPlane
	tcp.Name, tcp.Namespace = externalclusterreference.GenerateRemoteTenantControlPlaneNames(kcp)

	// Check KamajiControlPlaneUIDLabel on TCP, to avoid deleting it if it doesn't belong to our KCP
	if kcp.Spec.Deployment.ExternalClusterReference.DeploymentName != "" {

		if err := remoteClient.Get(ctx, types.NamespacedName{Namespace: tcp.Namespace, Name: tcp.Name}, &tcp); err != nil {
			if errors.IsNotFound(err) {
				log.Info("resource may have been deleted")
			}
			log.Error(err, "unable to get remote TenantControlPlane")
		}

		if val := tcp.GetLabels()[v1alpha1.KamajiControlPlaneUIDLabel]; val != "" && val != string(kcp.UID) {
			log.Info("Did not delete remote TenantControlPlane as it belongs to another KamajiControlPlane")

			return nil
		}
	}

	if tcpErr := remoteClient.Delete(ctx, &tcp); tcpErr != nil {
		if errors.IsNotFound(tcpErr) {
			log.Info("remote TenantControlPlane is already deleted")

			return nil
		}

		log.Error(tcpErr, "cannot delete remote TenantControlPlane")

		return tcpErr //nolint:wrapcheck
	}

	log.Info("remote TenantControlPlane has been deleted")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.client.Get(ctx, types.NamespacedName{Name: kcp.Name, Namespace: kcp.Namespace}, &kcp); err != nil {
			return err //nolint:wrapcheck
		}

		finalizers = sets.New[string](kcp.Finalizers...)
		finalizers.Delete(ExternalClusterReferenceFinalizer)

		kcp.Finalizers = finalizers.UnsortedList()

		return r.client.Update(ctx, &kcp)
	})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("object may have been deleted")

			return nil
		}

		log.Error(err, "unable to remove finalizer")

		return err //nolint:wrapcheck
	}

	log.Info("finalizer has been removed")

	return nil
}
