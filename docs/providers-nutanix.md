# Kamaji and Nutanix

The Kamaji Control Plane provider was able to create a _Nutanix_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                      READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/capi-quickstart                                   True                     5m42s
├─ClusterInfrastructure - NutanixCluster/capi-quickstart
├─ControlPlane - KamajiControlPlane/kamaji-nutanix-127
└─Workers
  └─MachineDeployment/capi-quickstart-md-0                True                     68s
    └─3 Machines...                                       True                     5m13s  See capi-quickstart-md-0-nfz4l-7hkx7, capi-quickstart-md-0-nfz4l-8wj6v, ...
```

## Example manifests

This example need a Service Load Balancer (MetalLB, Kube-VIP, ...) and [CAAPH](https://github.com/kubernetes-sigs/cluster-api-addon-provider-helm) installed in your management cluster.

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: capi-quickstart
  name: capi-quickstart
spec:
  clusterNetwork:
    apiServerPort: 6443
    pods:
      cidrBlocks:
        - 192.168.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KamajiControlPlane
    name: kamaji-nutanix-127
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: NutanixCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: NutanixCluster
metadata:
  name: capi-quickstart
spec:
  controlPlaneEndpoint:
    host: 0.0.0.0 # will be automatically patch by Kamaji controller
    port: 0 # will be automatically patch by Kamaji controller
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: kamaji-nutanix-127
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
    # serviceAddress: 10.83.1.2 # can be statically assigned
    serviceType: LoadBalancer
  deployment:
  replicas: 2
  version: 1.27.8
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: NutanixMachineTemplate
metadata:
  name: capi-quickstart-worker
spec:
  template:
    spec:
      bootType: legacy
      cluster:
        name: cloud-dev
        type: name
      image:
        name: ubuntu-2204-kube-v1.27.8
        type: name
      memorySize: 4Gi
      providerID: nutanix://mycluster-m1
      subnet:
      - name: capi
        type: name
      systemDiskSize: 40Gi
      vcpuSockets: 2
      vcpusPerSocket: 1
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
          kubeletExtraArgs:
            eviction-hard: nodefs.available<10%,nodefs.inodesFree<5%,imagefs.available<15%,memory.available<100Mi,imagefs.inodesFree<10%
            tls-cipher-suites: TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
            cloud-provider: external
      postKubeadmCommands:
      - echo "after kubeadm call" > /var/log/postkubeadm.log
      preKubeadmCommands:
      - echo "before kubeadm call" > /var/log/prekubeadm.log
      - hostnamectl set-hostname "{{ ds.meta_data.hostname }}"
      users:
      - lockPassword: false
        name: capiuser
        sshAuthorizedKeys:
        - ssh-ed25519 XXXXXXXXXX # Replace you SSH public key if you want direct access to worker nodes
        sudo: ALL=(ALL) NOPASSWD:ALL
      verbosity: 10
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
        kind: NutanixMachineTemplate
        name: capi-quickstart-worker
      version: v1.27.8
---
apiVersion: addons.cluster.x-k8s.io/v1alpha1
kind: HelmChartProxy
metadata:
  name: cilium
spec:
  clusterSelector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: capi-quickstart
  releaseName: cilium
  repoURL: https://helm.cilium.io/
  chartName: cilium
  namespace: kube-system
---
apiVersion: addons.cluster.x-k8s.io/v1alpha1
kind: HelmChartProxy
metadata:
  name: nutanix-cloud-provider
spec:
  clusterSelector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: capi-quickstart
  releaseName: nutanix-cloud-provider
  repoURL: https://nutanix.github.io/helm/
  chartName: nutanix-cloud-provider
  namespace: kube-system
  valuesTemplate: |
    prismCentralEndPoint: XXX
    username: XXX
    password: 'XXX'
    nodeSelector: ''
```

## Technical considerations

The Nutanix Kubernetes cluster is requiring a VIP for the Control Plane component.
To maintain the same experience you have to know in advance the Kamaji Tenant Control Plane address, and port.

In regard to the address, the following values must be the same:

- `KamajiControlPlane.spec.network.address`
- `NutanixCluster.spec.controlPlaneEndpoint.host`

The same applies for the Kubernetes API Server binding port:

- `Cluster.spec.clusterNetwork.apiServerPort`
- `NutanixCluster.spec.controlPlaneEndpoint.port`

If you install a Service Load Balancer solution (MetalLB, Kube-VIP, ...) in your management cluster you can skip this kind of check.
VIP will be automatically assigned and the Kamaji Control Plane provider will take care of patching the `NutanixCluster` resource with the endpoint provided by Kamaji itself.

## Kubernetes Nutanix Cloud Controller Manager customisation

As there is no Control Plane node with Kamaji architecture you are not able to run pods directly on it. In this case we need to customize the Nutanix Cloud Controller Manager install and remove the nodeSelector to let it run directly on the worker nodes.

Nutanix Cloud Controller HelmChartProxy example is in the manifest above.

If you install Nutanix Cloud Controller Manually with Helm you can follow the below example:

```shell
helm install nutanix-ccm nutanix/nutanix-cloud-provider -n kube-system --set prismCentralEndPoint=xxx,username=xxx,password='xxx',nodeSelector=''
```
