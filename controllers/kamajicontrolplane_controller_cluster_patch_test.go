// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("setOpenStackClusterAPIServer", func() {
	newOpenStackCluster := func(apiVersion string, spec map[string]interface{}) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       "OpenStackCluster",
			"spec":       spec,
		}}
	}

	It("sets the top-level fields for v1beta1", func() {
		osc := newOpenStackCluster("infrastructure.cluster.x-k8s.io/v1beta1", map[string]interface{}{
			"disableAPIServerFloatingIP": true,
		})

		Expect(setOpenStackClusterAPIServer(osc, "10.0.0.1", 6443)).To(Succeed())
		Expect(osc.Object["spec"]).To(Equal(map[string]interface{}{
			"disableAPIServerFloatingIP": true,
			"apiServerFixedIP":           "10.0.0.1",
			"apiServerPort":              int64(6443),
		}))
	})

	It("sets the spec.apiServer fields for v1beta2, preserving the existing ones", func() {
		osc := newOpenStackCluster("infrastructure.cluster.x-k8s.io/v1beta2", map[string]interface{}{
			"apiServer": map[string]interface{}{
				"enableFloatingIP": false,
			},
		})

		Expect(setOpenStackClusterAPIServer(osc, "10.0.0.1", 6443)).To(Succeed())
		Expect(osc.Object["spec"]).To(Equal(map[string]interface{}{
			"apiServer": map[string]interface{}{
				"enableFloatingIP": false,
				"fixedIP":          "10.0.0.1",
				"port":             int64(6443),
			},
		}))
	})

	It("creates spec.apiServer for v1beta2 when missing", func() {
		osc := newOpenStackCluster("infrastructure.cluster.x-k8s.io/v1beta2", map[string]interface{}{})

		Expect(setOpenStackClusterAPIServer(osc, "10.0.0.1", 6443)).To(Succeed())
		Expect(osc.Object["spec"]).To(Equal(map[string]interface{}{
			"apiServer": map[string]interface{}{
				"fixedIP": "10.0.0.1",
				"port":    int64(6443),
			},
		}))
	})
})
