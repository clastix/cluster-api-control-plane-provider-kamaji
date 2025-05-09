// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ControlPlaneComponent allows the customization for the given component of the control plane.
type ControlPlaneComponent struct {
	ExtraVolumeMounts []corev1.VolumeMount        `json:"extraVolumeMounts,omitempty"`
	ExtraArgs         []string                    `json:"extraArgs,omitempty"`
	Resources         corev1.ResourceRequirements `json:"resources,omitempty"`
	// In combination with the container registry, it can override the component container image.
	// With no value, the default images will be used.
	// +kubebuilder:validation:MinLength=1
	ContainerImageName string `json:"containerImageName,omitempty"`
}

// KineComponent allows the customization for the kine component of the control plane.
// Available only if Kamaji is running using Kine as backing storage.
type KineComponent struct {
	ExtraArgs []string                    `json:"extraArgs,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type IngressComponent struct {
	// Defines the Ingress Class for the Ingress object.
	ClassName string `json:"className,omitempty"`
	// Defines the hostname for the Ingress object.
	// When using an Ingress object the FQDN is automatically added to the Certificate SANs.
	// +kubebuilder:required
	// +kubebuilder:validation:MinLength=1
	Hostname string `json:"hostname"`
	// Defines the extra labels for the Ingress object.
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
	// Defines the extra annotations for the Ingress object.
	// Useful if you need to define TLS/SSL passthrough, or other Ingress Controller-specific options.
	ExtraAnnotations map[string]string `json:"extraAnnotations,omitempty"`
}

// LoadBalancerConfig is used when the KamajiControlPlane is exposed using a LoadBalancer service type.
type LoadBalancerConfig struct {
	// LoadBalancerSourceRanges restricts the IP ranges that can access
	// the LoadBalancer type Service. This field defines a list of IP
	// address ranges (in CIDR format) that are allowed to access the service.
	// If left empty, the service will allow traffic from all IP ranges (0.0.0.0/0).
	// This feature is useful for restricting access to API servers or services
	// to specific networks for security purposes.
	// Example: {"192.168.1.0/24", "10.0.0.0/8"}
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`
	// Specify the LoadBalancer class in case of multiple load balancer implementations.
	// Field supported only for Tenant Control Plane instances exposed using a LoadBalancer Service.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="LoadBalancerClass is immutable"
	LoadBalancerClass *string `json:"loadBalancerClass,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!has(self.loadBalancerConfig) || !has(self.loadBalancerConfig.loadBalancerSourceRanges) || (size(self.loadBalancerConfig.loadBalancerSourceRanges) == 0 || self.serviceType == 'LoadBalancer')", message="LoadBalancerSourceRanges are supported only with LoadBalancer service type"
// +kubebuilder:validation:XValidation:rule="!has(self.loadBalancerConfig) || !has(self.loadBalancerConfig.loadBalancerClass) || self.serviceType == 'LoadBalancer'", message="LoadBalancerClass is supported only with LoadBalancer service type"
// +kubebuilder:validation:XValidation:rule="self.serviceType != 'LoadBalancer' || (oldSelf.serviceType != 'LoadBalancer' && self.serviceType == 'LoadBalancer') || !has(self.loadBalancerConfig) || has(self.loadBalancerConfig) && has(self.loadBalancerConfig.loadBalancerClass) == has(oldSelf.loadBalancerConfig.loadBalancerClass)",message="LoadBalancerClass cannot be set or unset at runtime"

type NetworkComponent struct {
	// Optional configuration for the LoadBalancer service that exposes the Kamaji control plane.
	LoadBalancerConfig *LoadBalancerConfig `json:"loadBalancerConfig,omitempty"`
	// When specified, the KamajiControlPlane will be reachable using an Ingress object
	// deployed in the management cluster.
	Ingress *IngressComponent `json:"ingress,omitempty"`
	// +kubebuilder:default="LoadBalancer"
	ServiceType kamajiv1alpha1.ServiceType `json:"serviceType,omitempty"`
	// This field can be used in case of pre-assigned address, such as a VIP,
	// helping when serviceType is NodePort.
	ServiceAddress     string            `json:"serviceAddress,omitempty"`
	ServiceLabels      map[string]string `json:"serviceLabels,omitempty"`
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`
	// Configure additional Subject Address Names for the kube-apiserver certificate,
	// useful if the TenantControlPlane is going to be exposed behind a FQDN with NAT.
	CertSANs []string `json:"certSANs,omitempty"` //nolint:tagliatelle
	// DNSServiceIPs contains the DNS Service IPs.
	// If the CoreDNS addon is specified, its DNSServiceIPs will be used instead.
	// When set to an empty slice, Kamaji will automatically inflect it from the Service CIDR.
	DNSServiceIPs []string `json:"dnsServiceIPs,omitempty"`
}

// AddonsSpec defines the enabled addons and their features.
type AddonsSpec struct {
	kamajiv1alpha1.AddonsSpec `json:",inline"`

	CoreDNS *CoreDNSAddonSpec `json:"coreDNS,omitempty"` //nolint:tagliatelle
}

type CoreDNSAddonSpec struct {
	*kamajiv1alpha1.AddonSpec `json:",inline"`
	// DNSServiceIPs contains the CoreDNS Service IPs.
	// When set to an empty slice, Kamaji will automatically inflect it from the Service CIDR.
	DNSServiceIPs []string `json:"dnsServiceIPs,omitempty"`
}

// KamajiControlPlaneSpec defines the desired state of KamajiControlPlane.
type KamajiControlPlaneSpec struct {
	KamajiControlPlaneFields `json:",inline"`
	// ControlPlaneEndpoint propagates the endpoint the Kubernetes API Server managed by Kamaji is located.
	ControlPlaneEndpoint capiv1beta1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
	// Number of desired replicas for the given TenantControlPlane.
	// Defaults to 2.
	// +kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`
	// Version defines the desired Kubernetes version.
	Version string `json:"version"`
}

type DeploymentComponent struct {
	NodeSelector     map[string]string `json:"nodeSelector,omitempty"`
	RuntimeClassName string            `json:"runtimeClassName,omitempty"`
	// AdditionalMetadata refers to the additional labels and annotations attached
	// to the resulting Deployment managed by Kamaji.
	AdditionalMetadata kamajiv1alpha1.AdditionalMetadata `json:"additionalMetadata,omitempty"`
	// PodAdditionalMetadata defines the additional labels and annotations that must be attached
	// to the resulting Pods managed by the Deployment.
	PodAdditionalMetadata     kamajiv1alpha1.AdditionalMetadata `json:"podAdditionalMetadata,omitempty"`
	ServiceAccountName        string                            `json:"serviceAccountName,omitempty"`
	Strategy                  appsv1.DeploymentStrategy         `json:"strategy,omitempty"`
	Affinity                  *corev1.Affinity                  `json:"affinity,omitempty"`
	Tolerations               []corev1.Toleration               `json:"tolerations,omitempty"`
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	ExtraInitContainers       []corev1.Container                `json:"extraInitContainers,omitempty"`
	ExtraContainers           []corev1.Container                `json:"extraContainers,omitempty"`
	ExtraVolumes              []corev1.Volume                   `json:"extraVolumes,omitempty"`
	// ExternalClusterReference allows defining the target Cluster where the Tenant Control Plane components must be deployed.
	// When this value is nil, the Cluster API management cluster will be used as a target.
	// The ExternalClusterReference feature gate must be enabled with one of the available flags.
	ExternalClusterReference *ExternalClusterReference `json:"externalClusterReference,omitempty"`
}

type KamajiControlPlaneFields struct {
	// The Kamaji DataStore to use for the given TenantControlPlane.
	// Retrieve the list of the allowed ones by issuing "kubectl get datastores.kamaji.clastix.io".
	DataStoreName string `json:"dataStoreName,omitempty"`
	// DataStoreSchema allows to specify the name of the database (for relational DataStores) or the key prefix (for etcd)
	DataStoreSchema string `json:"dataStoreSchema,omitempty"`
	// The addons that must be managed by Kamaji, such as CoreDNS, kube-proxy, and konnectivity.
	Addons AddonsSpec `json:"addons,omitempty"`
	// List of the admission controllers to configure for the TenantControlPlane kube-apiserver.
	// By default, no admission controllers are enabled, refer to the desired Kubernetes version.
	//
	// More info: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/
	AdmissionControllers kamajiv1alpha1.AdmissionControllers `json:"admissionControllers,omitempty"`
	// Override the container registry used to pull the components image.
	// Helpful if running in an air-gapped environment.
	// +kubebuilder:default="registry.k8s.io"
	ContainerRegistry string `json:"registry,omitempty"`

	ControllerManager ControlPlaneComponent `json:"controllerManager,omitempty"`
	ApiServer         ControlPlaneComponent `json:"apiServer,omitempty"` //nolint:revive
	Scheduler         ControlPlaneComponent `json:"scheduler,omitempty"`
	Kine              KineComponent         `json:"kine,omitempty"`
	// Configure the Kubelet options, such as the preferred address types, or the expected cgroupfs.
	// +kubebuilder:default={preferredAddressTypes:{"Hostname","InternalIP","ExternalIP"},cgroupfs:"systemd"}
	Kubelet kamajiv1alpha1.KubeletSpec `json:"kubelet,omitempty"`
	// Configure how the TenantControlPlane should be exposed.
	// +kubebuilder:default={serviceType:"LoadBalancer"}
	Network NetworkComponent `json:"network,omitempty"`
	// Configure how the TenantControlPlane Deployment object should be configured.
	Deployment DeploymentComponent `json:"deployment,omitempty"`
}

type ExternalClusterReference struct {
	// The Secret object containing the kubeconfig used to interact with the remote cluster that will host
	// the Tenant Control Plane resources generated by the Control Plane Provider.
	// +kubebuilder:required
	// +kubebuilder:validation:MinLength=1
	KubeconfigSecretName string `json:"kubeconfigSecretName"`
	// The key used to extract the kubeconfig from the specified Secret.
	// +kubebuilder:required
	// +kubebuilder:validation:MinLength=1
	KubeconfigSecretKey string `json:"kubeconfigSecretKey"`
	// When ExternalClusterReferenceCrossNamespace is enabled allows specifying a different Namespace where the kubeconfig can be retrieved.
	// With ExternalClusterReference this value can be left empty since the KamajiControlPlane object Namespace will be used.
	KubeconfigSecretNamespace string `json:"kubeconfigSecretNamespace,omitempty"`
	// The Namespace where the resulting TenantControlPlane must be deployed to.
	DeploymentNamespace string `json:"deploymentNamespace"`
}

// KamajiControlPlaneStatus defines the observed state of KamajiControlPlane.
type KamajiControlPlaneStatus struct {
	// The TenantControlPlane has completed initialization.
	Initialized bool `json:"initialized"`
	// The Kamaji Control Plane is ready to link Cluster API with the Tenant Control Plane.
	Ready bool `json:"ready"`

	// Total number of fully running and ready control plane instances.
	ReadyReplicas int32 `json:"readyReplicas"`
	// Total number of non-terminated control plane instances.
	Replicas int32  `json:"replicas"`
	Selector string `json:"selector"`
	// Total number of unavailable TenantControlPlane instances targeted by this control plane,
	// equal to the desired number of control plane instances - ready instances.
	UnavailableReplicas int32 `json:"unavailableReplicas"`
	// Total number of non-terminated Pods targeted by this control plane that have the desired template spec.
	UpdatedReplicas int32 `json:"updatedReplicas"`
	// ExternalManagedControlPlane indicates to Cluster API that the Control Plane
	// is externally managed by Kamaji.
	// +kubebuilder:default=true
	ExternalManagedControlPlane *bool `json:"externalManagedControlPlane"`
	// Share the failed process of the KamajiControlPlane provider which wasn't able to complete the reconciliation for the given resource.
	FailureReason string `json:"failureReason,omitempty"`
	// The error message, if available, for the failing reconciliation.
	FailureMessage string `json:"failureMessage,omitempty"`
	// String representing the minimum Kubernetes version for the control plane machines in the cluster.
	Version    string             `json:"version"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:categories=cluster-api;kamaji,shortName=ktcp
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="The desired Kubernetes version"
//+kubebuilder:printcolumn:name="Initialized",type="boolean",JSONPath=".status.initialized",description="Check if the Kamaji Control Plane has been initialized"
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Check if the Kamaji Control Plane is up and running"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age"

// KamajiControlPlane is the Schema for the kamajicontrolplanes API.
type KamajiControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KamajiControlPlaneSpec   `json:"spec,omitempty"`
	Status KamajiControlPlaneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KamajiControlPlaneList contains a list of KamajiControlPlane.
type KamajiControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KamajiControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KamajiControlPlane{}, &KamajiControlPlaneList{})
}
