// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	goerrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/component-base/featuregate"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kcpv1alpha1 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/features"
)

// KamajiControlPlaneReconciler reconciles a KamajiControlPlane object.
type KamajiControlPlaneReconciler struct {
	ExternalClusterReferenceStore externalclusterreference.Store
	FeatureGates                  featuregate.FeatureGate
	MaxConcurrentReconciles       int

	client client.Client
}

//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kamajicontrolplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kamajicontrolplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kamajicontrolplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch

func (r *KamajiControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) { //nolint:funlen,cyclop,maintidx,gocognit,gocyclo
	var err error

	now, log := time.Now(), ctrllog.FromContext(ctx)

	log.Info("reconciliation started")

	// Retrieving the KamajiControlPlane instance from the request
	kcp := kcpv1alpha1.KamajiControlPlane{}
	if err = r.client.Get(ctx, req.NamespacedName, &kcp); err != nil {
		if errors.IsNotFound(err) {
			log.Info("resource may have been deleted")

			return ctrl.Result{}, nil
		}

		log.Error(err, "unable to get kcpv1alpha1.KamajiControlPlane")

		return ctrl.Result{}, err //nolint:wrapcheck
	}
	// The ControlPlane must have an OwnerReference set from the Cluster controller, waiting for this condition:
	// https://cluster-api.sigs.k8s.io/developer/architecture/controllers/control-plane.html#relationship-to-other-cluster-api-types
	if len(kcp.GetOwnerReferences()) == 0 {
		log.Info("missing OwnerReference from the Cluster controller, waiting for it")

		return ctrl.Result{}, nil
	}

	// Retrieving the Cluster information
	cluster := capiv1beta1.Cluster{}
	cluster.SetName(kcp.GetOwnerReferences()[0].Name)
	cluster.SetNamespace(kcp.GetNamespace())

	if err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &cluster); err != nil {
		if errors.IsNotFound(err) {
			log.Info("capiv1beta1.Cluster resource may have been deleted, withdrawing reconciliation")

			return ctrl.Result{}, nil
		}

		log.Error(err, "unable to get capiv1beta1.Cluster")

		return ctrl.Result{}, err //nolint:wrapcheck
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(&cluster, &kcp) {
		log.Info("Reconciliation is paused for this object")

		return ctrl.Result{}, nil
	}

	// Handling finalizer for external deployment:
	// in case of ExternalClusterReference the remote TCP must be deleted.
	if kcp.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, kcp)
	}

	// Extracting conditions, used to update the KamajiControlPlane ones upon the end of the reconciliation.
	conditions := kcp.Status.Conditions

	defer func() {
		deferErr := r.updateKamajiControlPlaneStatus(ctx, &kcp, func() {
			kcp.Status.Conditions = conditions
		})

		if deferErr != nil {
			log.Error(err, "unable to update kcpv1alpha1.KamajiControlPlane conditions")
		}
	}()
	// When ExternalClusterReference feature is enabled, we need to interact with a different API endpoint
	// to deploy and read the resulting Tenant Control Plane: in the case of nil value, it means we're targeting
	// the same management cluster, so no extra quirks are required.
	var remoteClient client.Client

	if kcp.Spec.Deployment.ExternalClusterReference != nil {
		TrackConditionType(&conditions, kcpv1alpha1.FoundExternalClusterReferenceConditionType, kcp.Generation, func() error {
			remoteClient, err = r.extractRemoteClient(ctx, kcp)

			return err
		})

		if err != nil {
			log.Error(err, "unable to get remote Client")

			return ctrl.Result{}, err
		}

		if err = r.handleFinalizer(ctx, &kcp); err != nil {
			log.Error(err, "unable to update finalizers")

			return ctrl.Result{}, err
		}
	}
	// Reconciling the Kamaji TenantControlPlane resource
	var tcp *kamajiv1alpha1.TenantControlPlane

	TrackConditionType(&conditions, kcpv1alpha1.TenantControlPlaneCreatedConditionType, kcp.Generation, func() error {
		tcp, err = r.createOrUpdateTenantControlPlane(ctx, remoteClient, cluster, kcp)

		return err
	})

	if err != nil {
		log.Error(err, "unable to create or update the TenantControlPlane instance")

		return ctrl.Result{}, err
	}
	// Waiting for the TenantControlPlane address: pay attention!
	//
	// This is still a work-in-progress and changing the Control Plane Controller contract.
	// Due to the given for granted concept that Control Plane and Worker nodes are on the same infrastructure,
	// we have to change the approach and wait for the advertised Control Plane endpoint, since Kamaji is offering a
	// Managed Kubernetes Service, although running as a regular pod.
	TrackConditionType(&conditions, kcpv1alpha1.TenantControlPlaneAddressReadyConditionType, kcp.Generation, func() error {
		if len(tcp.Status.ControlPlaneEndpoint) == 0 {
			err = fmt.Errorf("Control Plane Endpoint is not yet available since unprocessed by Kamaji") //nolint:goerr113,stylecheck
		}

		return err
	})
	// Treating the missing Control Plane Endpoint error as a sentinel:
	// there's no need to start the requeue with error logging, the Infrastructure Provider will react once the address
	// is available and assigned to the managed TenantControlPlane resource.
	if err != nil {
		log.Info(fmt.Sprintf("%s, enqueuing back", err.Error()))

		return ctrl.Result{}, nil //nolint:nilerr
	}
	// Starting from CAPI v1.8, the ControlPlane provider can set the Control Plane endpoint:
	// this will make useless the patchCluster function in the future.
	// More info: https://release-1-8.cluster-api.sigs.k8s.io/developer/providers/control-plane#optional-spec-fields-for-implementations-providing-endpoints
	TrackConditionType(&conditions, kcpv1alpha1.ControlPlaneEndpointPatchedConditionType, kcp.Generation, func() error {
		err = r.patchControlPlaneEndpoint(ctx, &kcp, tcp.Status.ControlPlaneEndpoint)

		return err
	})

	if err != nil {
		log.Error(err, "cannot patch kcpv1alpha1.KamajiControlPlane")

		return ctrl.Result{}, err
	}

	// We need to fetch the updated cluster resource here because otherwise the cluster.spec.controlPlaneEndpoint.Host
	// check that happens latter will never succeed.
	if err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &cluster); err != nil {
		if errors.IsNotFound(err) {
			log.Info("capiv1beta1.Cluster resource may have been deleted, withdrawing reconciliation")

			return ctrl.Result{}, nil
		}

		log.Error(err, "unable to get capiv1beta1.Cluster")

		return ctrl.Result{}, err //nolint:wrapcheck
	}

	// The following code path will be skipped when the InfraClusterOptional=true. This enables
	// the use of a KamajiControlPlane without an infrastructure cluster.
	if !r.FeatureGates.Enabled(features.SkipInfraClusterPatch) {
		// Patching the Infrastructure Cluster:
		// this will be removed on the upcoming Kamaji Control Plane versions.
		TrackConditionType(&conditions, kcpv1alpha1.InfrastructureClusterPatchedConditionType, kcp.Generation, func() error {
			err = r.patchCluster(ctx, cluster, &kcp, tcp.Status.ControlPlaneEndpoint)

			return err
		})

		if err != nil {
			log.Error(err, "cannot patch capiv1beta1.Cluster")

			return ctrl.Result{}, err
		}
	}

	// Before continuing, the Cluster object needs some validation, such as:
	// 1. an assigned Control Plane endpoint
	// 2. a ready infrastructure
	if len(cluster.Spec.ControlPlaneEndpoint.Host) == 0 {
		log.Info("capiv1beta1.Cluster Control Plane endpoint still unprocessed, enqueuing back")

		return ctrl.Result{Requeue: true}, nil
	}

	if !cluster.Status.InfrastructureReady {
		log.Info("capiv1beta1.Cluster infrastructure is not yet ready, enqueuing back")

		return ctrl.Result{Requeue: true}, nil
	}

	if tcp.Status.Kubernetes.Version.Status == nil {
		log.Info("kamajiv1alpha1.TenantControlPlane is not yet initialized, enqueuing back")

		return ctrl.Result{Requeue: true}, nil
	}

	if *tcp.Status.Kubernetes.Version.Status == kamajiv1alpha1.VersionReady && !kcp.Status.Initialized {
		// TenantControlPlane has been initialized
		TrackConditionType(&conditions, kcpv1alpha1.KamajiControlPlaneInitializedConditionType, kcp.Generation, func() error {
			err = r.updateKamajiControlPlaneStatus(ctx, &kcp, func() {
				kcp.Status.Initialized = true
			})

			return err
		})

		if err != nil {
			log.Error(err, "unable to set kcpv1alpha1.KamajiControlPlane as initialized")

			return ctrl.Result{}, err
		}
	}

	if !kcp.Status.Initialized {
		log.Info("kcpv1alpha1.KamajiControlPlane is not yet initialized, enqueuing back")

		return ctrl.Result{Requeue: true}, nil
	}

	// Updating KamajiControlPlane ready status, along with scaling values
	TrackConditionType(&conditions, kcpv1alpha1.KamajiControlPlaneInitializedConditionType, kcp.Generation, func() error {
		err = r.updateKamajiControlPlaneStatus(ctx, &kcp, func() {
			kcp.Status.ReadyReplicas = tcp.Status.Kubernetes.Deployment.ReadyReplicas
			kcp.Status.Replicas = tcp.Status.Kubernetes.Deployment.Replicas
			kcp.Status.Selector = metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: kcp.GetLabels()})
			kcp.Status.UnavailableReplicas = tcp.Status.Kubernetes.Deployment.UnavailableReplicas
			kcp.Status.UpdatedReplicas = tcp.Status.Kubernetes.Deployment.UpdatedReplicas
			kcp.Status.Version = tcp.Status.Kubernetes.Version.Version
		})

		return err
	})

	if err != nil {
		log.Error(err, "unable to report kcpv1alpha1.KamajiControlPlane status")

		return ctrl.Result{}, err
	}
	// KamajiControlPlane must be considered ready before replicating required resources
	TrackConditionType(&conditions, kcpv1alpha1.KamajiControlPlaneInitializedConditionType, kcp.Generation, func() error {
		err = r.updateKamajiControlPlaneStatus(ctx, &kcp, func() {
			kcp.Status.Initialized = true
		})

		return err
	})

	var result ctrl.Result

	TrackConditionType(&conditions, kcpv1alpha1.KubeadmResourcesCreatedReadyConditionType, kcp.Generation, func() error {
		result, err = r.createRequiredResources(ctx, remoteClient, cluster, kcp, tcp)

		return err
	})

	if err != nil {
		if goerrors.Is(err, ErrEnqueueBack) {
			log.Info(err.Error())

			return ctrl.Result{Requeue: true}, nil
		}

		log.Error(err, "unable to satisfy Secrets contract")

		return ctrl.Result{}, err
	}

	TrackConditionType(&conditions, kcpv1alpha1.KamajiControlPlaneReadyConditionType, kcp.Generation, func() error {
		err = r.updateKamajiControlPlaneStatus(ctx, &kcp, func() {
			kcp.Status.Ready = *tcp.Status.Kubernetes.Version.Status == kamajiv1alpha1.VersionReady || *tcp.Status.Kubernetes.Version.Status == kamajiv1alpha1.VersionUpgrading
		})
		if err != nil {
			return err
		}

		if !kcp.Status.Ready {
			return fmt.Errorf("TenantControlPlane in %s status, %w", *tcp.Status.Kubernetes.Version.Status, ErrEnqueueBack)
		}

		return nil
	})

	if err != nil {
		if goerrors.Is(err, ErrEnqueueBack) {
			log.Info(err.Error())

			return ctrl.Result{Requeue: true}, nil
		}

		log.Error(err, "unable to report kcpv1alpha1.KamajiControlPlane readiness")

		return ctrl.Result{}, err
	}

	log.Info("reconciliation completed", "duration", time.Since(now).String())

	return result, nil
}

func (r *KamajiControlPlaneReconciler) updateKamajiControlPlaneStatus(ctx context.Context, kcp *kcpv1alpha1.KamajiControlPlane, modifierFn func()) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.client.Get(ctx, types.NamespacedName{Name: kcp.Name, Namespace: kcp.Namespace}, kcp); err != nil {
			return err //nolint:wrapcheck
		}

		modifierFn()

		return r.client.Status().Update(ctx, kcp) //nolint:wrapcheck
	})
	if err != nil {
		return goerrors.Wrap(err, "cannot update KamajiControlPlane resource")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KamajiControlPlaneReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, channel chan event.GenericEvent) error {
	r.client = mgr.GetClient()
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&kcpv1alpha1.KamajiControlPlane{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return len(object.GetOwnerReferences()) > 0
		}))).
		Owns(&corev1.Secret{}).
		WatchesRawSource(source.Channel(channel, &handler.EnqueueRequestForObject{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx)))

	cs, csErr := kubernetes.NewForConfig(mgr.GetConfig())
	if csErr != nil {
		return goerrors.Wrap(csErr, "cannot create Kubernetes Client-set")
	}

	if _, rsErr := cs.Discovery().ServerResourcesForGroupVersion(kamajiv1alpha1.GroupVersion.String()); rsErr == nil {
		ctrlBuilder = ctrlBuilder.Owns(&kamajiv1alpha1.TenantControlPlane{})
	}

	//nolint:wrapcheck
	return ctrlBuilder.Complete(r)
}
