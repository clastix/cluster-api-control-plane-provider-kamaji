// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type KamajiControlPlaneConditionType string

var (
	TenantControlPlaneCreatedConditionType      KamajiControlPlaneConditionType = "TenantControlPlaneCreated"
	TenantControlPlaneAddressReadyConditionType KamajiControlPlaneConditionType = "TenantControlPlaneAddressReady"
	InfrastructureClusterPatchedConditionType   KamajiControlPlaneConditionType = "InfrastructureClusterPatched"
	KamajiControlPlaneInitializedConditionType  KamajiControlPlaneConditionType = "KamajiControlPlaneIsInitialized"
	KamajiControlPlaneReadyConditionType        KamajiControlPlaneConditionType = "KamajiControlPlaneIsReady"
	KubeadmResourcesCreatedReadyConditionType   KamajiControlPlaneConditionType = "KubeadmResourcesCreated"
)
