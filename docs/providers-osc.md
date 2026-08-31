# Kamaji and OUTSCALE

The Kamaji Control Plane provider was able to create an _OUTSCALE_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                      READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/capi-quickstart                                   True                     12m
├─ClusterInfrastructure - OscCluster/capi-quickstart
├─ControlPlane - KamajiControlPlane/capi-quickstart
└─Workers
  └─MachineDeployment/capi-quickstart-md-0                True                     68s
    └─3 Machines...                                       True                     5m13s  See capi-quickstart-md-0-6blxw-7hkx7, capi-quickstart-md-0-6blxw-9wj6v, ...
```

## Example manifests

The [Cluster API Provider OUTSCALE](https://github.com/outscale/cluster-api-provider-outscale) (CAPOSC) version must be `>= v1.2.0`, since the `network.disable` field is used to disable the OUTSCALE Load Balancer.

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
        - 172.16.0.0/16
    services:
      cidrBlocks:
        - 10.96.0.0/12
  controlPlaneRef:
    apiGroup: controlplane.cluster.x-k8s.io
    kind: KamajiControlPlane
    name: capi-quickstart
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: OscCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  credentials:
    fromSecret: cluster-api-provider-outscale
  network:
    disable:
      - loadbalancer
    bastion:
      enable: false
    net:
      ipRange: 10.0.0.0/16
    subregionName: eu-west-2a
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha2
kind: KamajiControlPlane
metadata:
  name: capi-quickstart
  namespace: default
spec:
  replicas: 3
  version: v1.35.7
  dataStoreName: default
  addons:
    coreDNS: {}
    kubeProxy: {}
  kubelet:
    preferredAddressTypes:
      - InternalIP
      - ExternalIP
      - Hostname
  network:
    serviceAnnotations:
      # service.beta.kubernetes.io/osc-load-balancer-internal: "true" # uncomment for an internal LBU
      service.beta.kubernetes.io/osc-load-balancer-ingress-address: both
    serviceType: LoadBalancer
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      node:
        image:
          name: ubuntu-2404-kube-v1.35.7
        vm:
          keypairName: my-keypair
          rootDisk:
            rootDiskIops: 2000
            rootDiskSize: 150
            rootDiskType: io1
          subregionName: eu-west-2a
          vmType: tinav7.c4r8p1
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta2
kind: KubeadmConfigTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          name: "{{ ds.meta_data.local_hostname }}"
          kubeletExtraArgs:
            - name: cloud-provider
              value: external
            - name: provider-id
              value: aws:///'{{ ds.meta_data.placement.availability_zone }}'/'{{ ds.meta_data.instance_id }}'
---
apiVersion: cluster.x-k8s.io/v1beta2
kind: MachineDeployment
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  clusterName: capi-quickstart
  replicas: 3
  rollout:
    strategy:
      rollingUpdate:
        maxSurge: 1
        maxUnavailable: 0
      type: RollingUpdate
  selector:
    matchLabels: {}
  template:
    spec:
      bootstrap:
        configRef:
          apiGroup: bootstrap.cluster.x-k8s.io
          kind: KubeadmConfigTemplate
          name: capi-quickstart-md-0
      clusterName: capi-quickstart
      infrastructureRef:
        apiGroup: infrastructure.cluster.x-k8s.io
        kind: OscMachineTemplate
        name: capi-quickstart-md-0
      version: v1.35.7
```

## Technical considerations

Once applying the `OscCluster` manifest, pay attention to performing these actions:

- The OUTSCALE Load Balancer (LBU) of the Control Plane provided by the `OscCluster` resource must be disabled, because Kamaji creates the endpoint of the Control Plane and updates `spec.controlPlaneEndpoint` based on it.
  Without the `network.disable` option, CAPOSC would reconcile the endpoint to the LBU DNS name and fight with the Kamaji Control Plane provider over the `spec.controlPlaneEndpoint` field.
- The `KamajiControlPlane.spec.network.serviceType: LoadBalancer` creates the Control Plane endpoint exposed by the OUTSCALE Cloud Controller Manager running in the management cluster.
