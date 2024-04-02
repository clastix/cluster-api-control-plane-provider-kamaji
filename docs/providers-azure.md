# Kamaji and Azure

The Kamaji Control Plane provider was able to create an _Azure_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                                  READY  SEVERITY  REASON  SINCE  MESSAGE                                                                                           
Cluster/capi-quickstart                                               True                     31m                                                                                                       
├─ClusterInfrastructure - AzureCluster/kamaji-quickstart-control-plane  True                     31m                                                                                                       
├─ControlPlane - KamajiControlPlane/kamaji-azure-127                                                                                                                                       
└─Workers                                                                                                                                                                                
  └─MachineDeployment/capi-quickstart-md-0                            True                     28s                                                                                                       
    └─3 Machines...                                                   True                     12m    See capi-quickstart-md-0-6848dccdffxn5j9b-cjgp5, capi-quickstart-md-0-6848dccdffxn5j9b-gk95g, ...
```

## Example manifests

The said cluster has been created with the following manifests.

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
        - 192.168.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KamajiControlPlane
    name: kamaji-quickstart-control-plane
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: AzureCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: AzureCluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  identityRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: AzureClusterIdentity
    name: azure-identity
  location: germanywestcentral
  networkSpec:
    subnets:
      - name: control-plane-subnet
        role: control-plane
      - name: node-subnet
        role: node
    vnet:
      name: workload
  resourceGroup: test-resource-group
  subscriptionID: 00000000-0000-0000-0000-000000000000
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: AzureClusterIdentity
metadata:
  labels:
    clusterctl.cluster.x-k8s.io/move-hierarchy: "true"
  name: azure-identity
spec:
  allowedNamespaces: {}
  clientID: 00000000-0000-0000-0000-000000000000
  clientSecret:
    name: azure-client-secret
    namespace: azure
  tenantID: 00000000-0000-0000-0000-000000000000
  type: ServicePrincipal
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: kamaji-quickstart-control-plane
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
    konnectivity: { }
    kubeProxy: { }
  kubelet:
    cgroupfs: systemd
    preferredAddressTypes:
      - ExternalIP
      - InternalIP
      - Hostname
  network:
    serviceAddress: 78.134.89.204
    serviceType: ClusterIP
  deployment:
  replicas: 2
  version: 1.26.0
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  clusterName: capi-quickstart
  replicas: 2
  selector:
    matchLabels: null
  template:
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: capi-quickstart-md-0
      clusterName: workload
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: AzureMachineTemplate
        name: capi-quickstart-md-0
      version: v1.26.0
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: AzureMachineTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      osDisk:
        diskSizeGB: 128
        osType: Linux
      sshPublicKey: ""
      vmSize: Standard_DS3_v2
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      files:
        - contentFrom:
            secret:
              key: worker-node-azure.json
              name: workload-md-0-azure-json
          owner: root:root
          path: /etc/kubernetes/azure.json
          permissions: "0644"
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            cloud-provider: external
          name: '{{ ds.meta_data["local_hostname"] }}'
      preKubeadmCommands: []
```

## Technical considerations

The Cluster API Azure infrastructure provider supports starting from [v1.13.0](https://github.com/kubernetes-sigs/cluster-api-provider-azure).
The Azure Cluster API operator will create some additional network components for the control plane which are not needed but do not hinder the cluster from working.

Once the cluster has been provisioned, you need to install the [Azure Cloud Controller Manager](https://github.com/kubernetes-sigs/cloud-provider-azure).
