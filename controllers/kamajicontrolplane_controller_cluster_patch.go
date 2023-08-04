// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:cyclop
func (r *KamajiControlPlaneReconciler) patchCluster(ctx context.Context, cluster capiv1beta1.Cluster, hostPort string) error {
	if cluster.Spec.InfrastructureRef == nil {
		return errors.New("capiv1beta1.Cluster has no InfrastructureRef")
	}

	endpoint, strPort, err := net.SplitHostPort(hostPort)
	if err != nil {
		return errors.Wrap(err, "cannot split the Kamaji endpoint host port pair")
	}

	port, err := strconv.ParseInt(strPort, 10, 64)
	if err != nil {
		return errors.Wrap(err, "cannot convert Kamaji endpoint port pair")
	}

	switch cluster.Spec.InfrastructureRef.Kind {
	case "AWSCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
	case "KubevirtCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, true)
	case "Metal3Cluster":
		return r.checkGenericCluster(ctx, cluster, endpoint, port)
	case "NutanixCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, true)
	case "OpenStackCluster":
		return r.patchOpenStackCluster(ctx, cluster, endpoint, port)
	case "PacketCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, true)
	case "VSphereCluster":
		return r.checkOrPatchVSphereCluster(ctx, cluster, endpoint, port)
	default:
		return errors.New("unsupported infrastructure provider")
	}
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vsphereclusters,verbs=get
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vsphereclusters,verbs=patch

func (r *KamajiControlPlaneReconciler) checkOrPatchVSphereCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	if err := r.checkGenericCluster(ctx, cluster, endpoint, port); err != nil {
		if errors.Is(err, UnmanagedControlPlaneAddressError{}) {
			return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
		}

		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters;nutanixclusters;packetclusters;awsclusters,verbs=patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters/status;nutanixclusters/status;packetclusters/status,verbs=patch

func (r *KamajiControlPlaneReconciler) patchGenericCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64, patchStatus bool) error {
	infraCluster := unstructured.Unstructured{}

	infraCluster.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	infraCluster.SetName(cluster.Spec.InfrastructureRef.Name)
	infraCluster.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	specPatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"controlPlaneEndpoint": map[string]interface{}{
				"host": endpoint,
				"port": port,
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to marshal %s spec patch", infraCluster.GetKind()))
	}

	if err = r.client.Patch(ctx, &infraCluster, client.RawPatch(types.MergePatchType, specPatch)); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot perform PATCH update for the %s resource", infraCluster.GetKind()))
	}

	if !patchStatus {
		return nil
	}

	statusPatch, err := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"ready": true,
		},
	})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to marshal %s status patch", infraCluster.GetKind()))
	}

	if err = r.client.Status().Patch(ctx, &infraCluster, client.RawPatch(types.MergePatchType, statusPatch)); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot perform PATCH update for the %s status", infraCluster.GetKind()))
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3clusters,verbs=get

func (r *KamajiControlPlaneReconciler) checkGenericCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	gkc := unstructured.Unstructured{}

	gkc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	gkc.SetName(cluster.Spec.InfrastructureRef.Name)
	gkc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	if err := r.client.Get(ctx, types.NamespacedName{Name: gkc.GetName(), Namespace: gkc.GetNamespace()}, &gkc); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot retrieve the %s resource", gkc.GetKind()))
	}

	controlPlaneEndpoint := gkc.Object["spec"].(map[string]interface{})["controlPlaneEndpoint"].(map[string]interface{}) //nolint:forcetypeassert

	cpHost, cpPort := controlPlaneEndpoint["host"].(string), controlPlaneEndpoint["port"].(int64) //nolint:forcetypeassert

	if len(cpHost) == 0 && cpPort == 0 {
		return NewUnmanagedControlPlaneAddressError(gkc.GetKind())
	}

	if cpHost != endpoint {
		return fmt.Errorf("the %s cluster has been provisioned with a mismatching host", gkc.GetKind()) //nolint:goerr113
	}

	if cpPort != port {
		return fmt.Errorf("the %s cluster has been provisioned with a mismatching port", gkc.GetKind()) //nolint:goerr113
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,verbs=patch

func (r *KamajiControlPlaneReconciler) patchOpenStackCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	osc := unstructured.Unstructured{}

	osc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	osc.SetName(cluster.Spec.InfrastructureRef.Name)
	osc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	mergePatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"apiServerFixedIP": endpoint,
			"apiServerPort":    port,
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal OpenStackCluster patch")
	}

	if err = r.client.Patch(ctx, &osc, client.RawPatch(types.MergePatchType, mergePatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the OpenStackCluster resource")
	}

	return nil
}
