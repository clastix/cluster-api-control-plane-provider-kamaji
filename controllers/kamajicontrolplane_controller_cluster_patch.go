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
		return r.patchGenericCluster(ctx, cluster, endpoint, port)
	case "Metal3Cluster":
		return r.checkMetal3Cluster(ctx, cluster, endpoint, port)
	case "OpenStackCluster":
		return r.patchOpenStackCluster(ctx, cluster, endpoint, port)
	case "PacketCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port)
	default:
		return errors.New("unsupported infrastructure provider")
	}
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters;packetclusters,verbs=patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters/status;packetclusters/status,verbs=patch

func (r *KamajiControlPlaneReconciler) patchGenericCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
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
