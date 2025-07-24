// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
)

func (r *KamajiControlPlaneReconciler) controlPlaneEndpoint(controlPlane *v1alpha1.KamajiControlPlane, statusEndpoint string) (string, int64, error) {
	endpoint, strPort, err := net.SplitHostPort(statusEndpoint)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot split the Kamaji endpoint host port pair")
	}

	port, pErr := strconv.ParseInt(strPort, 10, 16)
	if pErr != nil {
		return "", 0, errors.Wrap(pErr, "cannot convert port to integer")
	}

	if ingress := controlPlane.Spec.Network.Ingress; ingress != nil {
		if len(strings.Split(ingress.Hostname, ":")) == 1 {
			ingress.Hostname += ":443"
		}

		if endpoint, strPort, err = net.SplitHostPort(ingress.Hostname); err != nil {
			return "", 0, errors.Wrap(err, "cannot split the Kamaji Ingress hostname host port pair")
		}

		if port, pErr = strconv.ParseInt(strPort, 10, 64); pErr != nil {
			return "", 0, errors.Wrap(pErr, "cannot convert Kamaji Ingress hostname port pair")
		}
	}

	return endpoint, port, nil
}

func (r *KamajiControlPlaneReconciler) patchControlPlaneEndpoint(ctx context.Context, controlPlane *v1alpha1.KamajiControlPlane, hostPort string) error {
	endpoint, port, err := r.controlPlaneEndpoint(controlPlane, hostPort)
	if err != nil {
		return errors.Wrap(err, "cannot retrieve ControlPlaneEndpoint")
	}

	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if scopedErr := r.client.Get(ctx, client.ObjectKeyFromObject(controlPlane), controlPlane); scopedErr != nil {
			return errors.Wrap(scopedErr, "cannot retrieve *v1alpha1.KamajiControlPlane")
		}

		controlPlane.Spec.ControlPlaneEndpoint = capiv1beta1.APIEndpoint{
			Host: endpoint,
			Port: int32(port), //nolint:gosec
		}

		return r.client.Update(ctx, controlPlane)
	}); err != nil {
		return errors.Wrap(err, "cannot update KamajiControlPlane with ControlPlaneEndpoint")
	}

	return nil
}

//nolint:cyclop
func (r *KamajiControlPlaneReconciler) patchCluster(ctx context.Context, cluster capiv1beta1.Cluster, controlPlane *v1alpha1.KamajiControlPlane, hostPort string) error {
	if cluster.Spec.InfrastructureRef == nil {
		return errors.New("capiv1beta1.Cluster has no InfrastructureRef")
	}

	endpoint, port, err := r.controlPlaneEndpoint(controlPlane, hostPort)
	if err != nil {
		return errors.Wrap(err, "cannot retrieve ControlPlaneEndpoint")
	}

	switch cluster.Spec.InfrastructureRef.Kind {
	case "AWSCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
	case "AzureCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
	case "HetznerCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
	case "IonosCloudCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
	case "KubevirtCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, true)
	case "DockerCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
	case "Metal3Cluster":
		return r.checkGenericCluster(ctx, cluster, endpoint, port)
	case "NutanixCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, true)
	case "OpenStackCluster":
		return r.patchOpenStackCluster(ctx, cluster, endpoint, port)
	case "PacketCluster":
		return r.patchGenericCluster(ctx, cluster, endpoint, port, true)
	case "ProxmoxCluster":
		return r.checkOrPatchGenericCluster(ctx, cluster, endpoint, port)
	case "TinkerbellCluster":
		return r.checkOrPatchGenericCluster(ctx, cluster, endpoint, port)
	case "VSphereCluster":
		return r.checkOrPatchGenericCluster(ctx, cluster, endpoint, port)
	default:
		if r.DynamicInfrastructureClusters.Has(cluster.Spec.InfrastructureRef.Kind) {
			return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
		}

		return errors.New("unsupported infrastructure provider")
	}
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=proxmoxclusters;vsphereclusters;tinkerbellclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=proxmoxclusters;vsphereclusters;tinkerbellclusters,verbs=patch

func (r *KamajiControlPlaneReconciler) checkOrPatchGenericCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	if err := r.checkGenericCluster(ctx, cluster, endpoint, port); err != nil {
		if errors.As(err, &UnmanagedControlPlaneAddressError{}) {
			return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
		}

		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=awsclusters;azureclusters;dockerclusters;hetznerclusters;kubevirtclusters;nutanixclusters;packetclusters;ionoscloudclusters,verbs=patch;get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters/status;nutanixclusters/status;packetclusters/status,verbs=patch

func (r *KamajiControlPlaneReconciler) patchGenericCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64, patchStatus bool) error {
	infraCluster := unstructured.Unstructured{}

	infraCluster.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	infraCluster.SetName(cluster.Spec.InfrastructureRef.Name)
	infraCluster.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	if err := r.client.Get(ctx, types.NamespacedName{Name: infraCluster.GetName(), Namespace: infraCluster.GetNamespace()}, &infraCluster); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot retrieve the %s resource", infraCluster.GetKind()))
	}

	patchHelper, err := patch.NewHelper(&infraCluster, r.client)
	if err != nil {
		return errors.Wrap(err, "unable to create patch helper")
	}

	if err = unstructured.SetNestedMap(infraCluster.Object, map[string]interface{}{
		"host": endpoint,
		"port": port,
	}, "spec", "controlPlaneEndpoint"); err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to set unstructured %s spec patch", infraCluster.GetKind()))
	}

	if patchStatus {
		if err = unstructured.SetNestedField(infraCluster.Object, true, "status", "ready"); err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to set unstructured %s status patch", infraCluster.GetKind()))
		}
	}

	if err = patchHelper.Patch(ctx, &infraCluster); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot perform PATCH update for the %s resource", infraCluster.GetKind()))
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3clusters,verbs=get;list;watch

func (r *KamajiControlPlaneReconciler) checkGenericCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	gkc := unstructured.Unstructured{}

	gkc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	gkc.SetName(cluster.Spec.InfrastructureRef.Name)
	gkc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	if err := r.client.Get(ctx, types.NamespacedName{Name: gkc.GetName(), Namespace: gkc.GetNamespace()}, &gkc); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot retrieve the %s resource", gkc.GetKind()))
	}

	cpHost, _, err := unstructured.NestedString(gkc.Object, "spec", "controlPlaneEndpoint", "host")
	if err != nil {
		return errors.Wrap(err, "cannot extract control plane endpoint host")
	}

	if cpHost == "" {
		return *NewUnmanagedControlPlaneAddressError(gkc.GetKind())
	}

	cpPort, _, err := unstructured.NestedInt64(gkc.Object, "spec", "controlPlaneEndpoint", "port")
	if err != nil {
		return errors.Wrap(err, "cannot extract control plane endpoint host")
	}

	if len(cpHost) == 0 && cpPort == 0 {
		return *NewUnmanagedControlPlaneAddressError(gkc.GetKind())
	}

	if cpHost != endpoint {
		return fmt.Errorf("the %s cluster has been provisioned with a mismatching host", gkc.GetKind())
	}

	if cpPort != port {
		return fmt.Errorf("the %s cluster has been provisioned with a mismatching port", gkc.GetKind())
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,verbs=patch;get;list;watch

func (r *KamajiControlPlaneReconciler) patchOpenStackCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	osc := unstructured.Unstructured{}

	osc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	osc.SetName(cluster.Spec.InfrastructureRef.Name)
	osc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	if err := r.client.Get(ctx, types.NamespacedName{Name: osc.GetName(), Namespace: osc.GetNamespace()}, &osc); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot retrieve the %s resource", osc.GetKind()))
	}

	patchHelper, err := patch.NewHelper(&osc, r.client)
	if err != nil {
		return errors.Wrap(err, "unable to create patch helper")
	}

	if err = unstructured.SetNestedField(osc.Object, endpoint, "spec", "apiServerFixedIP"); err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to set unstructured %s spec apiServerFixedIP", osc.GetKind()))
	}

	if err = unstructured.SetNestedField(osc.Object, port, "spec", "apiServerPort"); err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to set unstructured %s spec apiServerPort", osc.GetKind()))
	}

	if err = patchHelper.Patch(ctx, &osc); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the OpenStackCluster resource")
	}

	return nil
}

func (r *KamajiControlPlaneReconciler) patchMainClusterEndpoint(ctx context.Context, cluster *capiv1beta1.Cluster, endpoint string, port int64) error {
	patchHelper, err := patch.NewHelper(cluster, r.client)
	if err != nil {
		return errors.Wrap(err, "unable to create patch helper for cluster")
	}

	cluster.Spec.ControlPlaneEndpoint = capiv1beta1.APIEndpoint{
		Host: endpoint,
		Port: int32(port),
	}

	if err = patchHelper.Patch(ctx, cluster); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the Cluster resource")
	}

	return nil
}
