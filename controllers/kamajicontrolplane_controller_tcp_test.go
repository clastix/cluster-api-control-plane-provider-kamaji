// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	"k8s.io/utils/ptr"

	capiv1alpha2 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha2"
)

var _ = Describe("controlPlaneProbes", func() {
	var kcp capiv1alpha2.KamajiControlPlane

	BeforeEach(func() {
		kcp = capiv1alpha2.KamajiControlPlane{}
	})

	It("is nil when no probes are set", func() {
		Expect(controlPlaneProbes(kcp)).To(BeNil())
	})

	It("forwards the Deployment probes without component overrides", func() {
		kcp.Spec.Deployment.Probes = &kamajiv1alpha1.ProbeSet{
			Liveness:  &kamajiv1alpha1.ProbeSpec{TimeoutSeconds: ptr.To[int32](5)},
			Readiness: &kamajiv1alpha1.ProbeSpec{TimeoutSeconds: ptr.To[int32](6)},
			Startup:   &kamajiv1alpha1.ProbeSpec{TimeoutSeconds: ptr.To[int32](7)},
		}

		Expect(controlPlaneProbes(kcp)).To(Equal(&kamajiv1alpha1.ControlPlaneProbes{
			Liveness:  kcp.Spec.Deployment.Probes.Liveness,
			Readiness: kcp.Spec.Deployment.Probes.Readiness,
			Startup:   kcp.Spec.Deployment.Probes.Startup,
		}))
	})

	It("forwards the Deployment probes along with the component overrides", func() {
		kcp.Spec.Deployment.Probes = &kamajiv1alpha1.ProbeSet{
			Liveness: &kamajiv1alpha1.ProbeSpec{TimeoutSeconds: ptr.To[int32](5)},
		}
		kcp.Spec.ApiServer.Probes = &kamajiv1alpha1.ProbeSet{
			Liveness: &kamajiv1alpha1.ProbeSpec{TimeoutSeconds: ptr.To[int32](10)},
		}
		kcp.Spec.ControllerManager.Probes = &kamajiv1alpha1.ProbeSet{
			Readiness: &kamajiv1alpha1.ProbeSpec{PeriodSeconds: ptr.To[int32](20)},
		}
		kcp.Spec.Scheduler.Probes = &kamajiv1alpha1.ProbeSet{
			Startup: &kamajiv1alpha1.ProbeSpec{FailureThreshold: ptr.To[int32](30)},
		}

		Expect(controlPlaneProbes(kcp)).To(Equal(&kamajiv1alpha1.ControlPlaneProbes{
			Liveness:          kcp.Spec.Deployment.Probes.Liveness,
			APIServer:         kcp.Spec.ApiServer.Probes,
			ControllerManager: kcp.Spec.ControllerManager.Probes,
			Scheduler:         kcp.Spec.Scheduler.Probes,
		}))
	})
})
