// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"github.com/pkg/errors"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck
	capiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	v1alpha2 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha2"
)

var (
	errKCPHubTypeMismatch     = errors.New("cannot convert KamajiControlPlane: hub is not *v1alpha2.KamajiControlPlane")
	errKCPTmplHubTypeMismatch = errors.New("cannot convert KamajiControlPlaneTemplate: hub is not *v1alpha2.KamajiControlPlaneTemplate")
)

func (src *KamajiControlPlane) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha2.KamajiControlPlane)
	if !ok {
		return errKCPHubTypeMismatch
	}

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = convertSpecToV1Alpha2(src.Spec)

	// int32 → *int32
	dst.Status.Replicas = &src.Status.Replicas
	dst.Status.ReadyReplicas = &src.Status.ReadyReplicas

	// updatedReplicas → upToDateReplicas
	dst.Status.UpToDateReplicas = &src.Status.UpdatedReplicas

	// initialized (bool) → initialization.controlPlaneInitialized (*bool)
	dst.Status.Initialization = &v1alpha2.KamajiControlPlaneInitializationStatus{
		ControlPlaneInitialized: &src.Status.Initialized,
	}

	dst.Status.Ready = src.Status.Ready
	dst.Status.Version = src.Status.Version
	dst.Status.Selector = &src.Status.Selector
	dst.Status.Conditions = src.Status.Conditions
	dst.Status.ExternalManagedControlPlane = src.Status.ExternalManagedControlPlane

	return nil
}

func (src *KamajiControlPlane) ConvertFrom(srcRaw conversion.Hub) error {
	hub, ok := srcRaw.(*v1alpha2.KamajiControlPlane)
	if !ok {
		return errKCPHubTypeMismatch
	}

	src.ObjectMeta = hub.ObjectMeta
	src.Spec = convertSpecToV1Alpha1(hub.Spec)

	src.Status.ExternalManagedControlPlane = hub.Status.ExternalManagedControlPlane
	src.Status.Ready = hub.Status.Ready
	src.Status.Version = hub.Status.Version

	// initialization.controlPlaneInitialized (*bool) → initialized (bool)
	if hub.Status.Initialization != nil &&
		hub.Status.Initialization.ControlPlaneInitialized != nil {
		src.Status.Initialized = *hub.Status.Initialization.ControlPlaneInitialized
	}

	// *int32 → int32 (nil-safe)
	if hub.Status.Replicas != nil {
		src.Status.Replicas = *hub.Status.Replicas
	}

	if hub.Status.ReadyReplicas != nil {
		src.Status.ReadyReplicas = *hub.Status.ReadyReplicas
	}

	if hub.Status.UpToDateReplicas != nil {
		src.Status.UpdatedReplicas = *hub.Status.UpToDateReplicas
	}

	if hub.Status.Selector != nil {
		src.Status.Selector = *hub.Status.Selector
	}

	// availableReplicas — new in v1alpha2, dropped on round trip
	// (controller repopulates on next reconcile)
	src.Status.Conditions = hub.Status.Conditions

	return nil
}

func (src *KamajiControlPlaneTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha2.KamajiControlPlaneTemplate)
	if !ok {
		return errKCPTmplHubTypeMismatch
	}

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.ObjectMeta = capiv1beta2.ObjectMeta(src.Spec.Template.ObjectMeta)
	dst.Spec.Template.Spec = convertFieldsToV2(src.Spec.Template.Spec)

	return nil
}

func (src *KamajiControlPlaneTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	hub, ok := srcRaw.(*v1alpha2.KamajiControlPlaneTemplate)
	if !ok {
		return errKCPTmplHubTypeMismatch
	}

	src.ObjectMeta = hub.ObjectMeta
	src.Spec.Template.ObjectMeta = capiv1beta1.ObjectMeta(hub.Spec.Template.ObjectMeta)
	src.Spec.Template.Spec = convertFieldsToV1(hub.Spec.Template.Spec)

	return nil
}

func convertFieldsToV2(src KamajiControlPlaneFields) v1alpha2.KamajiControlPlaneFields {
	return v1alpha2.KamajiControlPlaneFields{
		DataStoreName:        src.DataStoreName,
		DataStoreSchema:      src.DataStoreSchema,
		DataStoreUsername:    src.DataStoreUsername,
		DataStoreOverrides:   src.DataStoreOverrides,
		Addons:               convertAddonsToV2(src.Addons),
		AdmissionControllers: src.AdmissionControllers,
		ContainerRegistry:    src.ContainerRegistry,
		ControllerManager:    v1alpha2.ControlPlaneComponent(src.ControllerManager),
		ApiServer:            v1alpha2.ControlPlaneComponent(src.ApiServer),
		Scheduler:            v1alpha2.ControlPlaneComponent(src.Scheduler),
		Kine:                 v1alpha2.KineComponent(src.Kine),
		Kubelet:              src.Kubelet,
		Network:              convertNetworkToV2(src.Network),
		Deployment:           convertDeploymentToV2(src.Deployment),
	}
}

func convertFieldsToV1(src v1alpha2.KamajiControlPlaneFields) KamajiControlPlaneFields {
	return KamajiControlPlaneFields{
		DataStoreName:        src.DataStoreName,
		DataStoreSchema:      src.DataStoreSchema,
		DataStoreUsername:    src.DataStoreUsername,
		DataStoreOverrides:   src.DataStoreOverrides,
		Addons:               convertAddonsToV1(src.Addons),
		AdmissionControllers: src.AdmissionControllers,
		ContainerRegistry:    src.ContainerRegistry,
		ControllerManager:    ControlPlaneComponent(src.ControllerManager),
		ApiServer:            ControlPlaneComponent(src.ApiServer),
		Scheduler:            ControlPlaneComponent(src.Scheduler),
		Kine:                 KineComponent(src.Kine),
		Kubelet:              src.Kubelet,
		Network:              convertNetworkToV1(src.Network),
		Deployment:           convertDeploymentToV1(src.Deployment),
	}
}

func convertSpecToV1Alpha2(src KamajiControlPlaneSpec) v1alpha2.KamajiControlPlaneSpec {
	return v1alpha2.KamajiControlPlaneSpec{
		KamajiControlPlaneFields: convertFieldsToV2(src.KamajiControlPlaneFields),
		ControlPlaneEndpoint: capiv1beta2.APIEndpoint{
			Host: src.ControlPlaneEndpoint.Host,
			Port: src.ControlPlaneEndpoint.Port,
		},
		Replicas: src.Replicas,
		Version:  src.Version,
	}
}

func convertSpecToV1Alpha1(src v1alpha2.KamajiControlPlaneSpec) KamajiControlPlaneSpec {
	return KamajiControlPlaneSpec{
		KamajiControlPlaneFields: convertFieldsToV1(src.KamajiControlPlaneFields),
		ControlPlaneEndpoint: capiv1beta1.APIEndpoint{
			Host: src.ControlPlaneEndpoint.Host,
			Port: src.ControlPlaneEndpoint.Port,
		},
		Replicas: src.Replicas,
		Version:  src.Version,
	}
}

// Helpers for types with nested pointer-to-local-type fields that can't be directly cast.
func convertAddonsToV2(src AddonsSpec) v1alpha2.AddonsSpec {
	dst := v1alpha2.AddonsSpec{AddonsSpec: src.AddonsSpec}
	if src.CoreDNS != nil {
		v := v1alpha2.CoreDNSAddonSpec(*src.CoreDNS)
		dst.CoreDNS = &v
	}

	return dst
}

func convertAddonsToV1(src v1alpha2.AddonsSpec) AddonsSpec {
	dst := AddonsSpec{AddonsSpec: src.AddonsSpec}
	if src.CoreDNS != nil {
		v := CoreDNSAddonSpec(*src.CoreDNS)
		dst.CoreDNS = &v
	}

	return dst
}

func convertNetworkToV2(src NetworkComponent) v1alpha2.NetworkComponent {
	dst := v1alpha2.NetworkComponent{
		ServiceType:        src.ServiceType,
		ServiceAddress:     src.ServiceAddress,
		AdvertiseAddress:   src.AdvertiseAddress,
		ServiceLabels:      src.ServiceLabels,
		ServiceAnnotations: src.ServiceAnnotations,
		CertSANs:           src.CertSANs,
		DNSServiceIPs:      src.DNSServiceIPs,
	}

	if src.LoadBalancerConfig != nil {
		v := v1alpha2.LoadBalancerConfig(*src.LoadBalancerConfig)
		dst.LoadBalancerConfig = &v
	}

	if src.Gateway != nil {
		v := v1alpha2.GatewayComponent(*src.Gateway)
		dst.Gateway = &v
	}

	if src.Ingress != nil {
		v := v1alpha2.IngressComponent(*src.Ingress)
		dst.Ingress = &v
	}

	return dst
}

func convertNetworkToV1(src v1alpha2.NetworkComponent) NetworkComponent {
	dst := NetworkComponent{
		ServiceType:        src.ServiceType,
		ServiceAddress:     src.ServiceAddress,
		AdvertiseAddress:   src.AdvertiseAddress,
		ServiceLabels:      src.ServiceLabels,
		ServiceAnnotations: src.ServiceAnnotations,
		CertSANs:           src.CertSANs,
		DNSServiceIPs:      src.DNSServiceIPs,
	}
	if src.LoadBalancerConfig != nil {
		v := LoadBalancerConfig(*src.LoadBalancerConfig)
		dst.LoadBalancerConfig = &v
	}

	if src.Gateway != nil {
		v := GatewayComponent(*src.Gateway)
		dst.Gateway = &v
	}

	if src.Ingress != nil {
		v := IngressComponent(*src.Ingress)
		dst.Ingress = &v
	}

	return dst
}

func convertDeploymentToV2(src DeploymentComponent) v1alpha2.DeploymentComponent {
	dst := v1alpha2.DeploymentComponent{
		NodeSelector:              src.NodeSelector,
		RuntimeClassName:          src.RuntimeClassName,
		AdditionalMetadata:        src.AdditionalMetadata,
		PodAdditionalMetadata:     src.PodAdditionalMetadata,
		ServiceAccountName:        src.ServiceAccountName,
		Strategy:                  src.Strategy,
		Affinity:                  src.Affinity,
		Tolerations:               src.Tolerations,
		TopologySpreadConstraints: src.TopologySpreadConstraints,
		ExtraInitContainers:       src.ExtraInitContainers,
		ExtraContainers:           src.ExtraContainers,
		ExtraVolumes:              src.ExtraVolumes,
		Probes:                    src.Probes,
	}

	if src.ExternalClusterReference != nil {
		v := v1alpha2.ExternalClusterReference(*src.ExternalClusterReference)
		dst.ExternalClusterReference = &v
	}

	return dst
}

func convertDeploymentToV1(src v1alpha2.DeploymentComponent) DeploymentComponent {
	dst := DeploymentComponent{
		NodeSelector:              src.NodeSelector,
		RuntimeClassName:          src.RuntimeClassName,
		AdditionalMetadata:        src.AdditionalMetadata,
		PodAdditionalMetadata:     src.PodAdditionalMetadata,
		ServiceAccountName:        src.ServiceAccountName,
		Strategy:                  src.Strategy,
		Affinity:                  src.Affinity,
		Tolerations:               src.Tolerations,
		TopologySpreadConstraints: src.TopologySpreadConstraints,
		ExtraInitContainers:       src.ExtraInitContainers,
		ExtraContainers:           src.ExtraContainers,
		ExtraVolumes:              src.ExtraVolumes,
		Probes:                    src.Probes,
	}

	if src.ExternalClusterReference != nil {
		v := ExternalClusterReference(*src.ExternalClusterReference)
		dst.ExternalClusterReference = &v
	}

	return dst
}
