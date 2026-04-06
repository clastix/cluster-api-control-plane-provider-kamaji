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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
	capiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	capicontract "sigs.k8s.io/cluster-api/util/contract"
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

	// Ingress or Gateway API can be used to redefine the control plane endpoint
	var hostname string
	switch {
	case controlPlane.Spec.Network.Ingress != nil:
		hostname = controlPlane.Spec.Network.Ingress.Hostname
	case controlPlane.Spec.Network.Gateway != nil:
		hostname = controlPlane.Spec.Network.Gateway.Hostname
	}
	if hostname != "" {
		if len(strings.Split(hostname, ":")) == 1 {
			hostname += ":443"
		}
		if endpoint, strPort, err = net.SplitHostPort(hostname); err != nil {
			return "", 0, errors.Wrap(err, "cannot split the control plane hostname into endpoint and port")
		}
		if port, pErr = strconv.ParseInt(strPort, 10, 64); pErr != nil {
			return "", 0, errors.Wrap(pErr, "cannot parse the control plane port into an integer")
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

		controlPlane.Spec.ControlPlaneEndpoint = capiv1beta2.APIEndpoint{
			Host: endpoint,
			Port: int32(port), //nolint:gosec
		}

		return r.client.Update(ctx, controlPlane)
	}); err != nil {
		return errors.Wrap(err, "cannot update KamajiControlPlane with ControlPlaneEndpoint")
	}

	return nil
}

//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// getInfraClusterFromRef resolves a ContractVersionedObjectReference to an unstructured object
// by looking up the CRD metadata to discover the correct API version.
func (r *KamajiControlPlaneReconciler) getInfraClusterFromRef(ctx context.Context, ref capiv1beta2.ContractVersionedObjectReference, namespace string) (*unstructured.Unstructured, error) {
	if !ref.IsDefined() {
		return nil, errors.New("object reference is not defined")
	}

	// Look up the CRD metadata to find the served API version
	crdName := capicontract.CalculateCRDName(ref.APIGroup, ref.Kind)
	crdMeta := &metav1.PartialObjectMetadata{}
	crdMeta.SetName(crdName)
	crdMeta.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})

	if err := r.client.Get(ctx, client.ObjectKey{Name: crdName}, crdMeta); err != nil {
		return nil, errors.Wrapf(err, "cannot get CRD metadata for %s", ref.Kind)
	}

	// Find the latest compatible API version from contract labels.
	// CRDs are labeled like: cluster.x-k8s.io/v1beta2: v1beta2
	// or: cluster.x-k8s.io/v1beta1: v1beta1_v1beta2
	apiVersion := ""
	for _, contractVersion := range []string{"v1beta2", "v1beta1"} {
		labelKey := fmt.Sprintf("%s/%s", capiv1beta2.GroupVersion.Group, contractVersion)
		if versions, ok := crdMeta.GetLabels()[labelKey]; ok && versions != "" {
			// Pick the latest version from the underscore-separated list
			parts := strings.Split(versions, "_")
			apiVersion = parts[len(parts)-1]

			break
		}
	}

	if apiVersion == "" {
		return nil, fmt.Errorf("no compatible API version found in CRD %s labels", crdName)
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(schema.GroupVersion{Group: ref.APIGroup, Version: apiVersion}.String())
	obj.SetKind(ref.Kind)
	obj.SetName(ref.Name)
	obj.SetNamespace(namespace)

	if err := r.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		return nil, errors.Wrapf(err, "cannot retrieve %s %s/%s", ref.Kind, namespace, ref.Name)
	}

	return obj, nil
}

//nolint:cyclop
func (r *KamajiControlPlaneReconciler) patchCluster(ctx context.Context, cluster capiv1beta2.Cluster, controlPlane *v1alpha1.KamajiControlPlane, hostPort string) error {
	if !cluster.Spec.InfrastructureRef.IsDefined() {
		return errors.New("capiv1beta2.Cluster has no InfrastructureRef")
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
	case "Metal3Cluster":
		return r.checkGenericCluster(ctx, cluster, endpoint, port)
	case "MetalStackCluster":
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

func (r *KamajiControlPlaneReconciler) checkOrPatchGenericCluster(ctx context.Context, cluster capiv1beta2.Cluster, endpoint string, port int64) error {
	if err := r.checkGenericCluster(ctx, cluster, endpoint, port); err != nil {
		if errors.As(err, &UnmanagedControlPlaneAddressError{}) {
			return r.patchGenericCluster(ctx, cluster, endpoint, port, false)
		}

		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=awsclusters;azureclusters;hetznerclusters;kubevirtclusters;nutanixclusters;packetclusters;ionoscloudclusters,verbs=patch;get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters/status;nutanixclusters/status;packetclusters/status,verbs=patch

func (r *KamajiControlPlaneReconciler) patchGenericCluster(ctx context.Context, cluster capiv1beta2.Cluster, endpoint string, port int64, patchStatus bool) error {
	infraCluster, err := r.getInfraClusterFromRef(ctx, cluster.Spec.InfrastructureRef, cluster.GetNamespace())
	if err != nil {
		return errors.Wrap(err, "cannot retrieve infrastructure cluster "+cluster.Spec.InfrastructureRef.Kind)
	}

	patchHelper, err := patch.NewHelper(infraCluster, r.client)
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

	if err = patchHelper.Patch(ctx, infraCluster); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot perform PATCH update for the %s resource", infraCluster.GetKind()))
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3clusters;metalstackclusters,verbs=get;list;watch

func (r *KamajiControlPlaneReconciler) checkGenericCluster(ctx context.Context, cluster capiv1beta2.Cluster, endpoint string, port int64) error {
	gkc, err := r.getInfraClusterFromRef(ctx, cluster.Spec.InfrastructureRef, cluster.GetNamespace())
	if err != nil {
		return errors.Wrap(err, "cannot retrieve infrastructure cluster "+cluster.Spec.InfrastructureRef.Kind)
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

func (r *KamajiControlPlaneReconciler) patchOpenStackCluster(ctx context.Context, cluster capiv1beta2.Cluster, endpoint string, port int64) error {
	osc, err := r.getInfraClusterFromRef(ctx, cluster.Spec.InfrastructureRef, cluster.GetNamespace())
	if err != nil {
		return errors.Wrap(err, "cannot retrieve infrastructure cluster "+cluster.Spec.InfrastructureRef.Kind)
	}

	patchHelper, err := patch.NewHelper(osc, r.client)
	if err != nil {
		return errors.Wrap(err, "unable to create patch helper")
	}

	if err = unstructured.SetNestedField(osc.Object, endpoint, "spec", "apiServerFixedIP"); err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to set unstructured %s spec apiServerFixedIP", osc.GetKind()))
	}

	if err = unstructured.SetNestedField(osc.Object, port, "spec", "apiServerPort"); err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to set unstructured %s spec apiServerPort", osc.GetKind()))
	}

	if err = patchHelper.Patch(ctx, osc); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the OpenStackCluster resource")
	}

	return nil
}
