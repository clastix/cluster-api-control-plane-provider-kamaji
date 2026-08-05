// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	capiv1alpha2 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha2"
)

var _ = Describe("AllocateLoadBalancerNodePorts", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("is accepted on KamajiControlPlane with the LoadBalancer service type", func() {
		kcp := &capiv1alpha2.KamajiControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allocate-lb-nodeports-lb",
				Namespace: "default",
			},
			Spec: capiv1alpha2.KamajiControlPlaneSpec{
				KamajiControlPlaneFields: capiv1alpha2.KamajiControlPlaneFields{
					Network: capiv1alpha2.NetworkComponent{
						ServiceType:                   "LoadBalancer",
						AllocateLoadBalancerNodePorts: ptr.To(false),
					},
				},
				ControlPlaneEndpoint: capiv1beta2.APIEndpoint{
					Host: "127.0.0.1",
					Port: 6443,
				},
				Version: "1.31.0",
			},
		}
		Expect(k8sClient.Create(ctx, kcp)).To(Succeed())

		defer func() {
			Expect(k8sClient.Delete(ctx, kcp)).To(Succeed())
		}()

		Expect(kcp.Spec.Network.AllocateLoadBalancerNodePorts).To(HaveValue(BeFalse()))
	})

	It("is rejected on KamajiControlPlane with a non-LoadBalancer service type", func() {
		kcp := &capiv1alpha2.KamajiControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allocate-lb-nodeports-nodeport",
				Namespace: "default",
			},
			Spec: capiv1alpha2.KamajiControlPlaneSpec{
				KamajiControlPlaneFields: capiv1alpha2.KamajiControlPlaneFields{
					Network: capiv1alpha2.NetworkComponent{
						ServiceType:                   "NodePort",
						AllocateLoadBalancerNodePorts: ptr.To(false),
					},
				},
				ControlPlaneEndpoint: capiv1beta2.APIEndpoint{
					Host: "127.0.0.1",
					Port: 6443,
				},
				Version: "1.31.0",
			},
		}
		err := k8sClient.Create(ctx, kcp)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("AllocateLoadBalancerNodePorts is supported only with LoadBalancer service type"))
	})

	It("is accepted on KamajiControlPlaneTemplate with the LoadBalancer service type", func() {
		kcpTemplate := &capiv1alpha2.KamajiControlPlaneTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allocate-lb-nodeports-template",
				Namespace: "default",
			},
			Spec: capiv1alpha2.KamajiControlPlaneTemplateSpec{
				Template: capiv1alpha2.KamajiControlPlaneTemplateResource{
					ObjectMeta: capiv1beta2.ObjectMeta{
						Labels: map[string]string{"cluster.x-k8s.io/cluster-name": "test"},
					},
					Spec: capiv1alpha2.KamajiControlPlaneFields{
						Network: capiv1alpha2.NetworkComponent{
							ServiceType:                   "LoadBalancer",
							AllocateLoadBalancerNodePorts: ptr.To(false),
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, kcpTemplate)).To(Succeed())

		defer func() {
			Expect(k8sClient.Delete(ctx, kcpTemplate)).To(Succeed())
		}()

		Expect(kcpTemplate.Spec.Template.Spec.Network.AllocateLoadBalancerNodePorts).To(HaveValue(BeFalse()))
	})
})
