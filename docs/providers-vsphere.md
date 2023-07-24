# Kamaji and vSphere

The Kamaji Control Plane provider was able to create a _vSphere_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                       READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/capi-quickstart                                    True                     33m
├─ClusterInfrastructure - VSphereCluster/capi-quickstart   True                     34m
├─ControlPlane - KamajiControlPlane/kamaji-vsphere-125
└─Workers
  └─MachineDeployment/capi-quickstart-md-0                 True                     80s
    └─3 Machines...                                        True                     32m    See capi-quickstart-md-0-694f486d5xk9jj5-8b5xd, capi-quickstart-md-0-694f486d5xk9jj5-mr966, ...
```

## Example manifests

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: capi-quickstart
  name: capi-quickstart
spec:
  clusterNetwork:
    apiServerPort: 31429  # YMMV
    pods:
      cidrBlocks:
        - 192.168.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KamajiControlPlane
    name: kamaji-vsphere-125
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: VSphereCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: VSphereCluster
metadata:
  name: capi-quickstart
spec:
  controlPlaneEndpoint:
    host: 10.46.0.50 # YMMV
    port: 31429 # YMMV
  identityRef:
    kind: Secret
    name: capi-quickstart
  server: REDACTED # VSPHERE_SERVER
  thumbprint: REDACTED # VSPHERE_TLS_THUMBPRINT
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: kamaji-vsphere-125
  namespace: default
spec:
  apiServer:
    extraArgs:
      - --cloud-provider=external
  controllerManager:
    extraArgs:
      - --cloud-provider=external
  dataStoreName: default
  addons:
    coreDNS: { }
    kubeProxy: { }
  kubelet:
    cgroupfs: systemd
    preferredAddressTypes:
      - ExternalIP
      - InternalIP
      - Hostname
  network:
    serviceAddress: 10.46.0.50 # YMMV
    serviceType: NodePort
  deployment:
  replicas: 2
  version: 1.25.2
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: VSphereMachineTemplate
metadata:
  name: capi-quickstart-worker
spec:
  template:
    spec:
      cloneMode: linkedClone
      datacenter: REDACTED # VSPHERE_DATACENTER
      datastore: REDACTED # VSPHERE_DATASTORE
      diskGiB: 25
      folder: REDACTED # VSPHERE_FOLDER
      memoryMiB: 8192
      network:
        devices:
          - dhcp4: true
            networkName: REDACTED # VSPHERE_NETWORK
      numCPUs: 2
      os: Linux
      resourcePool: "REDACTED/REDACTED" # VSPHERE_RESOURCE_POOL
      server: REDACTED # VSPHERE_SERVER
      storagePolicyName: "" # VSPHERE_STORAGE_POLICY
      template: ubuntu-2004-kube-v1.26.2 # VSPHERE_TEMPLATE
      thumbprint: REDACTED # VSPHERE_TLS_THUMBPRINT
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: capi-quickstart-md-0
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          criSocket: /var/run/containerd/containerd.sock
          kubeletExtraArgs:
            cloud-provider: external
          name: '{{ local_hostname }}'
      preKubeadmCommands:
        - hostnamectl set-hostname "{{ ds.meta_data.hostname }}"
        - echo "::1         ipv6-localhost ipv6-loopback localhost6 localhost6.localdomain6"
          >/etc/hosts
        - echo "127.0.0.1   {{ ds.meta_data.hostname }} {{ local_hostname }} localhost
          localhost.localdomain localhost4 localhost4.localdomain4" >>/etc/hosts
      users:
        - name: capv
          sshAuthorizedKeys:
            - REDACTED
          sudo: ALL=(ALL) NOPASSWD:ALL
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: capi-quickstart
  name: capi-quickstart-md-0
spec:
  clusterName: capi-quickstart
  replicas: 3
  selector:
    matchLabels: {}
  template:
    metadata:
      labels:
        cluster.x-k8s.io/cluster-name: capi-quickstart
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: capi-quickstart-md-0
      clusterName: capi-quickstart
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: VSphereMachineTemplate
        name: capi-quickstart-worker
      version: v1.25.2
---
apiVersion: v1
kind: Secret
metadata:
  name: capi-quickstart
stringData:
  password: "REDACTED" # VSPHERE_PASSWORD
  username: "REDACTED" # VSPHERE_USERNAME
```

## Technical considerations

The vSphere Kubernetes cluster is requiring a VIP for the Control Plane component.
To maintain the same experience you have to know in advance the Kamaji Tenant Control Plane address, and port.

In regard to the address, the following values must be the same:

- `KamajiControlPlane.spec.network.address`
- `VSphereCluster.spec.controlPlaneEndpoint.host`

The same applies for the Kubernetes API Server binding port:

- `Cluster.spec.clusterNetwork.apiServerPort`
- `VSphereCluster.spec.controlPlaneEndpoint.port`

If your management cluster is offering a native Load Balancer solution you can skip this kind of check.
The Kamaji Control Plane provider will take care of patching the `VSphereCluster` resource with the endpoint provided by Kamaji itself.

## Kubernetes vSphere Cloud Provider customisation

`CAPV` is generating all the required manifests to install the CPI (Cloud Provider Interface) implementation to the downstream cluster.

Since the common setup is to have dedicated machines for the Control Plane with the tolerations, the `vsphere-cloud-controller-manager` object must be modified as follows:

- the `Kind` can be translated from `DaemonSet` to `Deployment`
- the following taints must be added to let it deploy to the worker nodes
```yaml
  - effect: NoSchedule
    key: node.cluster.x-k8s.io/uninitialized
  - effect: NoSchedule
    key: node.cloudprovider.kubernetes.io/uninitialized
    value: "true"
```
- the `affinity` must be removed, since giving for granted it must run on Control Plane nodes 
