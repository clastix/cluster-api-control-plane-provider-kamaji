// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package indexers

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ecr "github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
)

const (
	ExternalClusterReferenceSecretField = "externalClusterReferenceSecret"
)

type ExternalClusterReferenceSecret struct{}

func (e ExternalClusterReferenceSecret) Object() client.Object { //nolint:ireturn
	return &corev1.Secret{}
}

func (e ExternalClusterReferenceSecret) Field() string {
	return ExternalClusterReferenceSecretField
}

func (e ExternalClusterReferenceSecret) ExtractValue() client.IndexerFunc {
	return func(object client.Object) []string {
		secret := object.(*corev1.Secret) //nolint:forcetypeassert

		return ecr.GenerateKeyNameFromSecret(secret)
	}
}
