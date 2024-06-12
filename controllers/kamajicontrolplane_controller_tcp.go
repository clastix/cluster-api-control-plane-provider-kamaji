// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"strings"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/retry"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kcpv1alpha1 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
)

//+kubebuilder:rbac:groups=kamaji.clastix.io,resources=tenantcontrolplanes,verbs=get;list;watch;create;update

//nolint:funlen,gocognit,cyclop
func (r *KamajiControlPlaneReconciler) createOrUpdateTenantControlPlane(ctx context.Context, cluster capiv1beta1.Cluster, kcp kcpv1alpha1.KamajiControlPlane) (*kamajiv1alpha1.TenantControlPlane, error) {
	tcp := &kamajiv1alpha1.TenantControlPlane{}
	tcp.Name = kcp.GetName()
	tcp.Namespace = kcp.GetNamespace()

	if tcp.Annotations == nil {
		tcp.Annotations = make(map[string]string)
	}

	if kubeconfigSecretKey := kcp.Annotations[kamajiv1alpha1.KubeconfigSecretKeyAnnotation]; kubeconfigSecretKey != "" {
		tcp.Annotations[kamajiv1alpha1.KubeconfigSecretKeyAnnotation] = kubeconfigSecretKey
	} else {
		delete(tcp.Annotations, kamajiv1alpha1.KubeconfigSecretKeyAnnotation)
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, scopeErr := controllerutil.CreateOrUpdate(ctx, r.client, tcp, func() error {
			// TenantControlPlane port
			if apiPort := cluster.Spec.ClusterNetwork.APIServerPort; apiPort != nil {
				tcp.Spec.NetworkProfile.Port = *apiPort
			} else {
				tcp.Spec.NetworkProfile.Port = 6443
			}
			// TenantControlPlane Services CIDR
			if serviceCIDR := cluster.Spec.ClusterNetwork.Services; serviceCIDR != nil && len(serviceCIDR.CIDRBlocks) > 0 {
				tcp.Spec.NetworkProfile.ServiceCIDR = serviceCIDR.CIDRBlocks[0]
			}
			// TenantControlPlane Pods CIDR
			if podsCIDR := cluster.Spec.ClusterNetwork.Pods; podsCIDR != nil && len(podsCIDR.CIDRBlocks) > 0 {
				tcp.Spec.NetworkProfile.PodCIDR = podsCIDR.CIDRBlocks[0]
			}
			// Replicas
			tcp.Spec.ControlPlane.Deployment.Replicas = kcp.Spec.Replicas
			// Version
			// Tolerate version strings without a "v" prefix: prepend it if it's not there
			if !strings.HasPrefix(kcp.Spec.Version, "v") {
				tcp.Spec.Kubernetes.Version = fmt.Sprintf("v%s", kcp.Spec.Version)
			} else {
				tcp.Spec.Kubernetes.Version = kcp.Spec.Version
			}
			// Kamaji addons and CoreDNS overrides
			tcp.Spec.Addons = kcp.Spec.Addons.AddonsSpec
			if kcp.Spec.Addons.CoreDNS != nil {
				tcp.Spec.NetworkProfile.DNSServiceIPs = kcp.Spec.Addons.CoreDNS.DNSServiceIPs

				if kcp.Spec.Addons.CoreDNS.AddonSpec == nil {
					kcp.Spec.Addons.CoreDNS.AddonSpec = &kamajiv1alpha1.AddonSpec{}
				}

				tcp.Spec.Addons.CoreDNS = kcp.Spec.Addons.CoreDNS.AddonSpec
			} else {
				tcp.Spec.Addons.CoreDNS = nil
				tcp.Spec.NetworkProfile.DNSServiceIPs = nil
			}
			// Kamaji specific options
			tcp.Spec.DataStore = kcp.Spec.DataStoreName
			tcp.Spec.Kubernetes.AdmissionControllers = kcp.Spec.AdmissionControllers
			tcp.Spec.ControlPlane.Deployment.RegistrySettings.Registry = kcp.Spec.ContainerRegistry
			// Volume mounts
			if tcp.Spec.ControlPlane.Deployment.AdditionalVolumeMounts == nil {
				tcp.Spec.ControlPlane.Deployment.AdditionalVolumeMounts = &kamajiv1alpha1.AdditionalVolumeMounts{}
			}

			tcp.Spec.ControlPlane.Deployment.AdditionalVolumeMounts.ControllerManager = kcp.Spec.ControllerManager.ExtraVolumeMounts
			tcp.Spec.ControlPlane.Deployment.AdditionalVolumeMounts.Scheduler = kcp.Spec.Scheduler.ExtraVolumeMounts
			tcp.Spec.ControlPlane.Deployment.AdditionalVolumeMounts.APIServer = kcp.Spec.ApiServer.ExtraVolumeMounts
			// Extra args
			if tcp.Spec.ControlPlane.Deployment.ExtraArgs == nil {
				tcp.Spec.ControlPlane.Deployment.ExtraArgs = &kamajiv1alpha1.ControlPlaneExtraArgs{}
			}

			tcp.Spec.ControlPlane.Deployment.ExtraArgs.ControllerManager = kcp.Spec.ControllerManager.ExtraArgs
			tcp.Spec.ControlPlane.Deployment.ExtraArgs.Scheduler = kcp.Spec.Scheduler.ExtraArgs
			tcp.Spec.ControlPlane.Deployment.ExtraArgs.APIServer = kcp.Spec.ApiServer.ExtraArgs
			tcp.Spec.ControlPlane.Deployment.ExtraArgs.Kine = kcp.Spec.Kine.ExtraArgs
			// Resources
			if tcp.Spec.ControlPlane.Deployment.Resources == nil {
				tcp.Spec.ControlPlane.Deployment.Resources = &kamajiv1alpha1.ControlPlaneComponentsResources{}
			}

			tcp.Spec.ControlPlane.Deployment.Resources.ControllerManager = &kcp.Spec.ControllerManager.Resources
			tcp.Spec.ControlPlane.Deployment.Resources.Scheduler = &kcp.Spec.Scheduler.Resources
			tcp.Spec.ControlPlane.Deployment.Resources.APIServer = &kcp.Spec.ApiServer.Resources
			tcp.Spec.ControlPlane.Deployment.Resources.Kine = &kcp.Spec.Kine.Resources
			// Container image overrides
			tcp.Spec.ControlPlane.Deployment.RegistrySettings.ControllerManagerImage = kcp.Spec.ControllerManager.ContainerImageName
			tcp.Spec.ControlPlane.Deployment.RegistrySettings.SchedulerImage = kcp.Spec.Scheduler.ContainerImageName
			tcp.Spec.ControlPlane.Deployment.RegistrySettings.APIServerImage = kcp.Spec.ApiServer.ContainerImageName
			// Kubelet
			tcp.Spec.Kubernetes.Kubelet = kcp.Spec.Kubelet
			// Network
			tcp.Spec.NetworkProfile.Address = kcp.Spec.Network.ServiceAddress
			tcp.Spec.ControlPlane.Service.ServiceType = kcp.Spec.Network.ServiceType
			tcp.Spec.ControlPlane.Service.AdditionalMetadata.Labels = kcp.Spec.Network.ServiceLabels
			tcp.Spec.ControlPlane.Service.AdditionalMetadata.Annotations = kcp.Spec.Network.ServiceAnnotations
			tcp.Spec.NetworkProfile.CertSANs = kcp.Spec.Network.CertSANs
			// Ingress
			if kcp.Spec.Network.Ingress != nil {
				tcp.Spec.ControlPlane.Ingress = &kamajiv1alpha1.IngressSpec{
					AdditionalMetadata: kamajiv1alpha1.AdditionalMetadata{
						Labels:      kcp.Spec.Network.Ingress.ExtraLabels,
						Annotations: kcp.Spec.Network.Ingress.ExtraAnnotations,
					},
					IngressClassName: kcp.Spec.Network.Ingress.ClassName,
					Hostname:         kcp.Spec.Network.Ingress.Hostname,
				}
				// In the case of enabled ingress, adding the FQDN to the CertSANs
				if tcp.Spec.NetworkProfile.CertSANs == nil {
					tcp.Spec.NetworkProfile.CertSANs = []string{}
				}

				tcp.Spec.NetworkProfile.CertSANs = append(tcp.Spec.NetworkProfile.CertSANs, kcp.Spec.Network.Ingress.Hostname)
			} else {
				tcp.Spec.ControlPlane.Ingress = nil
			}
			// Deployment
			tcp.Spec.ControlPlane.Deployment.NodeSelector = kcp.Spec.Deployment.NodeSelector
			tcp.Spec.ControlPlane.Deployment.RuntimeClassName = kcp.Spec.Deployment.RuntimeClassName
			tcp.Spec.ControlPlane.Deployment.ServiceAccountName = kcp.Spec.Deployment.ServiceAccountName
			tcp.Spec.ControlPlane.Deployment.AdditionalMetadata = kcp.Spec.Deployment.AdditionalMetadata
			tcp.Spec.ControlPlane.Deployment.PodAdditionalMetadata = kcp.Spec.Deployment.PodAdditionalMetadata
			tcp.Spec.ControlPlane.Deployment.Strategy = kcp.Spec.Deployment.Strategy
			tcp.Spec.ControlPlane.Deployment.Affinity = kcp.Spec.Deployment.Affinity
			tcp.Spec.ControlPlane.Deployment.Tolerations = kcp.Spec.Deployment.Tolerations
			tcp.Spec.ControlPlane.Deployment.TopologySpreadConstraints = kcp.Spec.Deployment.TopologySpreadConstraints
			tcp.Spec.ControlPlane.Deployment.AdditionalInitContainers = kcp.Spec.Deployment.ExtraInitContainers
			tcp.Spec.ControlPlane.Deployment.AdditionalContainers = kcp.Spec.Deployment.ExtraContainers
			tcp.Spec.ControlPlane.Deployment.AdditionalVolumes = kcp.Spec.Deployment.ExtraVolumes

			return controllerutil.SetControllerReference(&kcp, tcp, r.client.Scheme()) //nolint:wrapcheck
		})

		return scopeErr //nolint:wrapcheck
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create or update TenantControlPlane")
	}

	return tcp, nil
}
