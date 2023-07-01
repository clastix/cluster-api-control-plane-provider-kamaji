// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	case "KubevirtCluster":
		return r.patchKubeVirtCluster(ctx, cluster, endpoint, port)
	case "Metal3Cluster":
		return r.checkMetal3Cluster(ctx, cluster, endpoint, port)
	case "OpenStackCluster":
		return r.patchOpenStackCluster(ctx, cluster, endpoint, port)
	case "PacketCluster":
		return r.patchPacketCluster(ctx, cluster, endpoint, port)
	default:
		return errors.New("unsupported infrastructure provider")
	}
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters,verbs=patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters/status,verbs=patch

//nolint:dupl
func (r *KamajiControlPlaneReconciler) patchKubeVirtCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	kvc := unstructured.Unstructured{}

	kvc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	kvc.SetName(cluster.Spec.InfrastructureRef.Name)
	kvc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	specPatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"controlPlaneEndpoint": map[string]interface{}{
				"host": endpoint,
				"port": port,
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal KubeVirtCluster spec patch")
	}

	if err = r.client.Patch(ctx, &kvc, client.RawPatch(types.MergePatchType, specPatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the KubeVirtCluster resource")
	}

	statusPatch, err := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"ready": true,
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal KubeVirtCluster status patch")
	}

	if err = r.client.Status().Patch(ctx, &kvc, client.RawPatch(types.MergePatchType, statusPatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the KubeVirtCluster status")
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3clusters,verbs=get;watch

func (r *KamajiControlPlaneReconciler) checkMetal3Cluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	mkc := unstructured.Unstructured{}

	mkc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	mkc.SetName(cluster.Spec.InfrastructureRef.Name)
	mkc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	if err := r.client.Get(ctx, types.NamespacedName{Name: mkc.GetName(), Namespace: mkc.GetNamespace()}, &mkc); err != nil {
		return errors.Wrap(err, "cannot retrieve the Metal3Cluster resource")
	}

	controlPlaneEndpoint := mkc.Object["spec"].(map[string]interface{})["controlPlaneEndpoint"].(map[string]interface{}) //nolint:forcetypeassert
	if controlPlaneEndpoint["host"].(string) != endpoint {                                                               //nolint:forcetypeassert
		return errors.New("the Metal3 cluster has been provisioned with a mismatching host")
	}

	if controlPlaneEndpoint["port"].(int64) != port { //nolint:forcetypeassert
		return errors.New("the Metal3 cluster has been provisioned with a mismatching port")
	}

	return nil
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=packetclusters,verbs=patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=packetclusters/status,verbs=patch

//nolint:dupl
func (r *KamajiControlPlaneReconciler) patchPacketCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	packetCluster := unstructured.Unstructured{}

	packetCluster.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	packetCluster.SetName(cluster.Spec.InfrastructureRef.Name)
	packetCluster.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	specPatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"controlPlaneEndpoint": map[string]interface{}{
				"host": endpoint,
				"port": port,
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal PacketCluster spec patch")
	}

	if err = r.client.Patch(ctx, &packetCluster, client.RawPatch(types.MergePatchType, specPatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the PacketCluster resource")
	}

	statusPatch, err := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"ready": true,
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal PacketCluster status patch")
	}

	if err = r.client.Status().Patch(ctx, &packetCluster, client.RawPatch(types.MergePatchType, statusPatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the PacketCluster status")
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
