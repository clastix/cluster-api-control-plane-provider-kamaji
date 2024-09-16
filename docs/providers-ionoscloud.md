# Kamaji and IONOS Cloud

The Kamaji Control Plane provider was able to create an _IONOS Cloud_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                           READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/kamaji-quickstart                                      True                     10m
├─ClusterInfrastructure - IonosCloudCluster/kamaji-quickstart  True                     11m
├─ControlPlane - KamajiControlPlane/kamaji-quickstart
└─Workers
  └─MachineDeployment/kamaji-quickstart                        True                     19s
    └─Machine/kamaji-quickstart-xqwjx-5xhln                    True                     105s
```

## Example manifests

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: kamaji-quickstart
  labels:
    cluster.x-k8s.io/cluster-name: kamaji-quickstart
spec:
  clusterNetwork:
    apiServerPort: 6443
    pods:
      cidrBlocks:
        - 192.168.0.0/16
  controlPlaneRef:
    kind: KamajiControlPlane
    apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
    name: kamaji-quickstart
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: IonosCloudCluster
    name: kamaji-quickstart
---
apiVersion: v1
kind: Secret
metadata:
  name: kamaji-quickstart-credentials
  labels:
    cluster.x-k8s.io/cluster-name: kamaji-quickstart
type: Opaque
stringData:
  token: REDACTED
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: IonosCloudCluster
metadata:
  name: kamaji-quickstart
  labels:
    cluster.x-k8s.io/cluster-name: kamaji-quickstart
spec:
  credentialsRef:
    name: kamaji-quickstart-credentials
---
kind: KamajiControlPlane
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
metadata:
  name: kamaji-quickstart
  labels:
    cluster.x-k8s.io/cluster-name: kamaji-quickstart
spec:
  replicas: 1
  version: 1.29.2
  dataStoreName: default
  addons:
    coreDNS: {}
    kubeProxy: {}
  network:
    serviceType: LoadBalancer
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: kamaji-quickstart
  labels: cluster.x-k8s.io/cluster-name: kamaji-quickstart
spec:
  clusterName: kamaji-quickstart
  replicas: 1
  selector:
    matchLabels:
  template:
    metadata:
      labels:
        node-role.kubernetes.io/node: ""
        cluster.x-k8s.io/cluster-name: kamaji-quickstart
    spec:
      clusterName: kamaji-quickstart
      version: 1.29.2
      bootstrap:
        configRef:
          name: kamaji-quickstart
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
      infrastructureRef:
        name: kamaji-quickstart
        apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
        kind: IonosCloudMachineTemplate
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: IonosCloudMachineTemplate
metadata:
  name: kamaji-quickstart
  labels:
    cluster.x-k8s.io/cluster-name: kamaji-quickstart
spec:
  template:
    spec:
      datacenterID: REDACTED
      numCores: 2
      memoryMB: 4096
      disk:
        image:
          id: REDACTED
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: kamaji-quickstart
  labels:
    cluster.x-k8s.io/cluster-name: kamaji-quickstart
spec:
  template:
    spec:
      users:
        - name: root
          sshAuthorizedKeys: [REDACTED]
      files:
        - content: |
            {"datacenter-id":"REDACTED"}
          owner: root:root
          path: /etc/ie-csi/cfg.json
          permissions: '0644'
      postKubeadmCommands:
        - |
          export system_uuid=$(kubectl --kubeconfig /etc/kubernetes/kubelet.conf get node $(hostname) -ojsonpath='{..systemUUID }')
          kubectl --kubeconfig /etc/kubernetes/kubelet.conf patch node $(hostname) --type strategic -p '{"spec": {"providerID": "ionos://'${system_uuid}'"}}'
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            cloud-provider: ""
          criSocket: unix:///run/containerd/containerd.sock
```

## Technical considerations

The Cluster API IONOS Cloud infrastructure provider supports Kamaji managed Control Planes starting from [v0.4.0](https://github.com/ionos-cloud/cluster-api-provider-ionoscloud/releases/tag/v0.4.0).

To make use of service type `LoadBalancer` for the `KamajiControlPlane`, you need to install the [IONOS Cloud Controller Manager](https://github.com/ionos-cloud/cloud-provider-ionoscloud/tree/main/charts/ionoscloud-cloud-controller-manager).
Alternatively, you can install the CAPI stack in a [managed cluster](https://cloud.ionos.com/managed/kubernetes).
