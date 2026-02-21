// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"net"
	"strings"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kcpv1alpha1 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
)

var ErrUnsupportedCertificateSAN = errors.New("a certificate SAN must be made of host only with no port")

//+kubebuilder:rbac:groups=kamaji.clastix.io,resources=tenantcontrolplanes,verbs=get;list;watch;create;update

//nolint:funlen,gocognit,cyclop,maintidx,gocyclo
func (r *KamajiControlPlaneReconciler) createOrUpdateTenantControlPlane(ctx context.Context, remoteClient client.Client, cluster capiv1beta1.Cluster, kcp kcpv1alpha1.KamajiControlPlane) (*kamajiv1alpha1.TenantControlPlane, error) {
	tcp := &kamajiv1alpha1.TenantControlPlane{}
	tcp.Name = kcp.GetName()
	tcp.Namespace = kcp.GetNamespace()

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		k8sClient := r.client

		var isDelegatedExternally bool

		if isDelegatedExternally = remoteClient != nil; isDelegatedExternally {
			k8sClient = remoteClient
			tcp.Name, tcp.Namespace = externalclusterreference.GenerateRemoteTenantControlPlaneNames(kcp)
		}

		_, scopeErr := controllerutil.CreateOrUpdate(ctx, k8sClient, tcp, func() error {
			if tcp.Annotations == nil {
				tcp.Annotations = make(map[string]string)
			}

			for k, v := range kcp.Annotations {
				if k == corev1.LastAppliedConfigAnnotation {
					continue
				}

				tcp.Annotations[k] = v
			}

			tcp.Labels = kcp.Labels

			if kubeconfigSecretKey := kcp.Annotations[kamajiv1alpha1.KubeconfigSecretKeyAnnotation]; kubeconfigSecretKey != "" {
				tcp.Annotations[kamajiv1alpha1.KubeconfigSecretKeyAnnotation] = kubeconfigSecretKey
			} else {
				delete(tcp.Annotations, kamajiv1alpha1.KubeconfigSecretKeyAnnotation)
			}
			if cluster.Spec.ClusterNetwork != nil {
				// TenantControlPlane port
				if apiPort := cluster.Spec.ClusterNetwork.APIServerPort; apiPort != nil {
					tcp.Spec.NetworkProfile.Port = *apiPort
				}
				// TenantControlPlane Services CIDR
				if serviceCIDR := cluster.Spec.ClusterNetwork.Services; serviceCIDR != nil && len(serviceCIDR.CIDRBlocks) > 0 {
					tcp.Spec.NetworkProfile.ServiceCIDR = serviceCIDR.CIDRBlocks[0]
				}
				// TenantControlPlane Pods CIDR
				if podsCIDR := cluster.Spec.ClusterNetwork.Pods; podsCIDR != nil && len(podsCIDR.CIDRBlocks) > 0 {
					tcp.Spec.NetworkProfile.PodCIDR = podsCIDR.CIDRBlocks[0]
				}
				// TenantControlPlane cluster domain
				tcp.Spec.NetworkProfile.ClusterDomain = cluster.Spec.ClusterNetwork.ServiceDomain
			}
			// Replicas
			tcp.Spec.ControlPlane.Deployment.Replicas = kcp.Spec.Replicas
			// Version
			// Tolerate version strings without a "v" prefix: prepend it if it's not there
			if !strings.HasPrefix(kcp.Spec.Version, "v") {
				tcp.Spec.Kubernetes.Version = "v" + kcp.Spec.Version
			} else {
				tcp.Spec.Kubernetes.Version = kcp.Spec.Version
			}
			// Set before CoreDNS addon to allow override.
			tcp.Spec.NetworkProfile.DNSServiceIPs = kcp.Spec.Network.DNSServiceIPs
			// Kamaji addons and CoreDNS overrides
			tcp.Spec.Addons = kcp.Spec.Addons.AddonsSpec
			if kcp.Spec.Addons.CoreDNS != nil {
				tcp.Spec.NetworkProfile.DNSServiceIPs = kcp.Spec.Addons.CoreDNS.DNSServiceIPs

				if kcp.Spec.Addons.CoreDNS.AddonSpec == nil {
					kcp.Spec.Addons.CoreDNS.AddonSpec = &kamajiv1alpha1.AddonSpec{}
				}

				tcp.Spec.Addons.CoreDNS = kcp.Spec.Addons.CoreDNS.AddonSpec
			}
			// Kamaji specific options
			if kcp.Spec.DataStoreName != "" {
				tcp.Spec.DataStore = kcp.Spec.DataStoreName
			}
			if kcp.Spec.DataStoreSchema != "" {
				tcp.Spec.DataStoreSchema = kcp.Spec.DataStoreSchema
			}
			if kcp.Spec.DataStoreUsername != "" {
				tcp.Spec.DataStoreUsername = kcp.Spec.DataStoreUsername
			}
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

			for _, i := range kcp.Spec.Network.CertSANs {
				// validating CertSANs as soon as possible to avoid github.com/clastix/kamaji/issues/679:
				// nil err means the entry is in the form of <HOST>:<PORT> which is not accepted
				if _, _, err := net.SplitHostPort(i); err == nil {
					return errors.Wrap(ErrUnsupportedCertificateSAN, fmt.Sprintf("entry %s is invalid", i))
				}
			}

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

				if host, _, err := net.SplitHostPort(kcp.Spec.Network.Ingress.Hostname); err == nil {
					// no error means <FQDN>:<PORT>, we need the host variable
					tcp.Spec.NetworkProfile.CertSANs = append(tcp.Spec.NetworkProfile.CertSANs, host)
				} else {
					// No port specification, adding bare entry
					tcp.Spec.NetworkProfile.CertSANs = append(tcp.Spec.NetworkProfile.CertSANs, kcp.Spec.Network.Ingress.Hostname)
				}
			} else {
				tcp.Spec.ControlPlane.Ingress = nil
			}
			// LoadBalancer
			if kcp.Spec.Network.LoadBalancerConfig != nil {
				if lbClass := kcp.Spec.Network.LoadBalancerConfig.LoadBalancerClass; lbClass != nil {
					tcp.Spec.NetworkProfile.LoadBalancerClass = ptr.To(*lbClass)
				}

				if srcRange := kcp.Spec.Network.LoadBalancerConfig.LoadBalancerSourceRanges; srcRange != nil {
					tcp.Spec.NetworkProfile.LoadBalancerSourceRanges = srcRange
				}
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

			if !isDelegatedExternally {
				return controllerutil.SetControllerReference(&kcp, tcp, k8sClient.Scheme())
			}

			return nil
		})

		return scopeErr //nolint:wrapcheck
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create or update TenantControlPlane")
	}

	return tcp, nil
}
