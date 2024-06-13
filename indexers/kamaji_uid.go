// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package indexers

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	kcpv1alpha1 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
)

const (
	KamajiControlPlaneUIDField = "kamajiControlPlaneUID"
)

type KamajiControlPlaneUID struct{}

func (k KamajiControlPlaneUID) Object() client.Object { //nolint:ireturn
	return &kcpv1alpha1.KamajiControlPlane{}
}

func (k KamajiControlPlaneUID) Field() string {
	return KamajiControlPlaneUIDField
}

func (k KamajiControlPlaneUID) ExtractValue() client.IndexerFunc {
	return func(object client.Object) []string {
		kcp := object.(*kcpv1alpha1.KamajiControlPlane) //nolint:forcetypeassert

		return []string{string(kcp.ObjectMeta.UID)}
	}
}
