// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// KamajiControlPlaneTemplateSpec defines the desired state of KamajiControlPlaneTemplate.
type KamajiControlPlaneTemplateSpec struct {
	Template KamajiControlPlaneTemplateResource `json:"template"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:categories=cluster-api;kamaji,shortName=ktcpt

// KamajiControlPlaneTemplate is the Schema for the kamajicontrolplanetemplates API.
type KamajiControlPlaneTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KamajiControlPlaneTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// KamajiControlPlaneTemplateList contains a list of KamajiControlPlaneTemplate.
type KamajiControlPlaneTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KamajiControlPlaneTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KamajiControlPlaneTemplate{}, &KamajiControlPlaneTemplateList{})
}

// KamajiControlPlaneTemplateResource describes the data needed to create a KamajiControlPlane from a template.
type KamajiControlPlaneTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta     `json:"metadata,omitempty"`
	Spec       KamajiControlPlaneFields `json:"spec"`
}
