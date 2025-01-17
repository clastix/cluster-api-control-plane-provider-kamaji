// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	ecr "github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/features"
)

const (
	ExternalClusterReferenceFinalizer = "ecr.kamaji.clastix.io/finalizer"
)

var (
	ErrExternalClusterReferenceNotEnabled                 = errors.New("external cluster feature gates are not enabled")
	ErrExternalClusterReferenceCrossNamespaceReference    = errors.New("the ExternalClusterReference is enforcing kubeconfig in the same Namespace, ExternalClusterReferenceCrossNamespace must be enabled")
	ErrExternalCLusterReferenceSecretEmptyError           = errors.New("could not extract kubeconfig for external cluster reference, secret is empty")
	ErrExternalClusterReferenceSecretKeyEmpty             = errors.New("could not extract kubeconfig for external cluster reference, key is empty")
	ErrExternalClusterReferenceNonInitializedStore        = errors.New("remote manager is not yet initialized")
	ErrExternalClusterReferenceTenantControlPlaneNotFound = errors.New("TenantControlPlane custom resource not available in external cluster")
)

//nolint:cyclop
func (r *KamajiControlPlaneReconciler) extractRemoteClient(ctx context.Context, kcp v1alpha1.KamajiControlPlane) (client.Client, error) { //nolint:ireturn
	if !r.FeatureGates.Enabled(features.ExternalClusterReference)  {
		return nil, ErrExternalClusterReferenceNotEnabled
	}

	if r.FeatureGates.Enabled(features.ExternalClusterReference) && 
		!r.FeatureGates.Enabled(features.ExternalClusterReferenceCrossNamespace) &&
		kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretNamespace != "" &&
		kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretNamespace != kcp.Namespace {
		return nil, ErrExternalClusterReferenceCrossNamespaceReference
	}

	namespace := kcp.Namespace

	if kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretNamespace != "" {
		namespace = kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretNamespace
	}

	var secret corev1.Secret

	if err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretName}, &secret); err != nil {
		return nil, errors.Wrapf(err, "could not get external cluster reference secret")
	}

	if secret.Data == nil {
		return nil, ErrExternalCLusterReferenceSecretEmptyError
	}

	if secret.Data[kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretKey] == nil {
		return nil, ErrExternalClusterReferenceSecretKeyEmpty
	}

	mgr, found := r.ExternalClusterReferenceStore.Get(ecr.GenerateKeyNameFromKamaji(&kcp), secret.ResourceVersion)
	if !found {
		return nil, ErrExternalClusterReferenceNonInitializedStore
	}

	// Use the RESTMapper to check if the CRD is installed
	gvr := kamajiv1alpha1.GroupVersion.WithResource("tenantcontrolplanes")
	if _, err := mgr.GetRESTMapper().KindFor(gvr); err != nil {
		return nil, ErrExternalClusterReferenceTenantControlPlaneNotFound
	}

	return mgr.GetClient(), nil
}
