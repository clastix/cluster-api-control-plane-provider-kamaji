# Kamaji and Beskar7

The Kamaji Control Plane provider allows creating a _Beskar7_ backed bare-metal Kubernetes cluster by providing Kamaji Control Planes.

## Control Plane endpoint

The [Cluster API Provider Beskar7](https://github.com/projectbeskar/beskar7) version must be `>= v0.9.0`.

Beskar7 never discovers a Control Plane endpoint, it reads it from `Cluster.spec.controlPlaneEndpoint` or `Beskar7Cluster.spec.controlPlaneEndpoint`.

The Kamaji Control Plane provider fills `Beskar7Cluster.spec.controlPlaneEndpoint` once the Tenant Control Plane is reachable:

- when the endpoint is empty, it is patched with the Kamaji Control Plane endpoint;
- when the endpoint is already set, it must match the Kamaji Control Plane endpoint, otherwise the reconciliation fails.

Leave the endpoint empty in the `Beskar7Cluster` (or `Beskar7ClusterTemplate`) to let Kamaji manage it.

## Example manifests

```yaml
apiVersion: cluster.x-k8s.io/v1beta2
kind: Cluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
        - 10.244.0.0/16
  controlPlaneRef:
    apiGroup: controlplane.cluster.x-k8s.io
    kind: KamajiControlPlane
    name: capi-quickstart
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: Beskar7Cluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: Beskar7Cluster
metadata:
  name: capi-quickstart
  namespace: default
spec: {}
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha2
kind: KamajiControlPlane
metadata:
  name: capi-quickstart
  namespace: default
spec:
  replicas: 2
  version: v1.35.7
  dataStoreName: default
  addons:
    coreDNS: {}
    kubeProxy: {}
  network:
    serviceType: LoadBalancer
```

Worker nodes are provisioned with `Beskar7MachineTemplate` resources: refer to the Beskar7 documentation for the machine and bootstrap configuration.
