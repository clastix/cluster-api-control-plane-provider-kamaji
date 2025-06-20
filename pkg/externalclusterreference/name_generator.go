// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package externalclusterreference

import (
	"strings"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
)

const (
	RemoteTCPPrefix = "kcp-"
)

func ParseKamajiControlPlaneUIDFromTenantControlPlane(tcp kamajiv1alpha1.TenantControlPlane) string {
	if !strings.HasPrefix(tcp.Name, RemoteTCPPrefix) {
		return ""
	}

	return strings.TrimPrefix(tcp.Name, RemoteTCPPrefix)
}

func GenerateRemoteTenantControlPlaneNames(kcp v1alpha1.KamajiControlPlane) (name string, namespace string) { //nolint:nonamedreturns
	if kcp.Spec.Deployment.ExternalClusterReference.KeepDefaultName {
		return kcp.GetName(), kcp.Spec.Deployment.ExternalClusterReference.DeploymentNamespace
	}

	return RemoteTCPPrefix + string(kcp.UID), kcp.Spec.Deployment.ExternalClusterReference.DeploymentNamespace
}

func GenerateKeyNameFromSecret(secret *corev1.Secret) []string {
	names := make([]string, 0, len(secret.Data))

	for k := range secret.Data {
		names = append(names, secret.Namespace+"/"+secret.Name+"/"+k)
	}

	return names
}

func GenerateKeyNameFromKamaji(kcp *v1alpha1.KamajiControlPlane) string {
	namespace := kcp.Namespace

	if kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretNamespace != "" {
		namespace = kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretNamespace
	}

	return namespace + "/" + kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretName + "/" + kcp.Spec.Deployment.ExternalClusterReference.KubeconfigSecretKey
}
