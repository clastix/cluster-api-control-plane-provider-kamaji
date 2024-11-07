// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package features

const (
	// ExternalClusterReference allows deploying Tenant Control Plane pods to a different cluster from the Management one.
	// This will require a valid kubeconfig referenced in the KamajiControlPlane object, in the same Namespace of the said object.
	ExternalClusterReference = "ExternalClusterReference"

	// ExternalClusterReferenceCrossNamespace allows deploying Tenant Control Plane pods to a different cluster from the Management one.
	// It supports referencing a kubeconfig available in a different Namespace than the KamajiControlPlane.
	ExternalClusterReferenceCrossNamespace = "ExternalClusterReferenceCrossNamespace"

	// SkipInfraClusterPatch bypasses patching the InfraCluster with the control-plane endpoint.
	SkipInfraClusterPatch = "SkipInfraClusterPatch"
)
