// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package indexers

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	kcpv1alpha1 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	ecr "github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
)

const (
	ExternalClusterReferenceKamajiControlPlaneField = "externalClusterReferenceKamajiControlPlane"
)

type ExternalClusterReferenceKamajiControlPlane struct{}

func (e ExternalClusterReferenceKamajiControlPlane) Object() client.Object { //nolint:ireturn
	return &kcpv1alpha1.KamajiControlPlane{}
}

func (e ExternalClusterReferenceKamajiControlPlane) Field() string {
	return ExternalClusterReferenceKamajiControlPlaneField
}

func (e ExternalClusterReferenceKamajiControlPlane) ExtractValue() client.IndexerFunc {
	return func(object client.Object) []string {
		kcp := object.(*kcpv1alpha1.KamajiControlPlane) //nolint:forcetypeassert

		if kcp.Spec.Deployment.ExternalClusterReference != nil {
			return []string{ecr.GenerateKeyNameFromKamaji(kcp)}
		}

		return nil
	}
}
