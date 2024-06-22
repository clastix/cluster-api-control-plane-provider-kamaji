# Kamaji and OpenStack

The Kamaji Control Plane provider was able to create an OpenStack backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                                 READY  SEVERITY  REASON  SINCE  MESSAGE 
Cluster/capi-quickstart                                              True                     12m             
├─ClusterInfrastructure - OpenStackCluster/capi-quickstart                                                    
├─ControlPlane - KamajiControlPlane/kamaji-quickstart-control-plane                                           
└─Workers                                                                                                     
  └─MachineDeployment/capi-quickstart-md-0                           True                     2m43s           
    └─Machine/capi-quickstart-md-0-f5xz7-w54x9                       True                     3m20s 
```

## Example manifests

The cluster has been created with the following manifests.

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  clusterNetwork:
    serviceDomain: cluster.local
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
    kind: KamajiControlPlane
    name: kamaji-quickstart-control-plane
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: OpenStackCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackCluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  apiServerLoadBalancer:
    enabled: false
  controlPlaneAvailabilityZones:
  - REDACTED
  disableAPIServerFloatingIP: true
  disableExternalNetwork: true
  apiServerFixedIP: ""
  network:
    id: REDACTED
  subnets:
  - id: REDACTED
  identityRef:
    name: capi-quickstart
    cloudName: openstack
  managedSecurityGroups:
    allowAllInClusterTraffic: false
    allNodesSecurityGroupRules:
    - remoteManagedGroups:
      - worker
      direction: ingress
      etherType: IPv4
      name: BGP
      portRangeMin: 179
      portRangeMax: 179
      protocol: "tcp"
      description: "Allow BGP among workers"
    - remoteManagedGroups:
      - worker
      direction: ingress
      etherType: IPv4
      name: IP-in-IP
      protocol: "4"
      description: "Allow IP-in-IP among workers"
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: kamaji-quickstart-control-plane
  namespace: default
spec:
  replicas: 1
  version: 1.28.10
  dataStoreName: default
  addons:
    coreDNS: {}
    kubeProxy: {}
    konnectivity: {}
  kubelet:
    preferredAddressTypes:
      - InternalIP
      - ExternalIP
      - Hostname
  network:
    serviceType: LoadBalancer
    serviceAnnotations:
      service.beta.kubernetes.io/openstack-internal-load-balancer: "false"
      loadbalancer.openstack.org/floating-network-id: REDACTED
  controllerManager:
    extraArgs:
    - --cloud-provider=external
  apiServer:
    extraArgs:
    - --cloud-provider=external
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  clusterName: capi-quickstart
  replicas: 1
  template:
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: capi-quickstart-md-0
      clusterName: capi-quickstart
      failureDomain: REDACTED
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: OpenStackMachineTemplate
        name: capi-quickstart-md-0
      version: v1.28.10
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OpenStackMachineTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      flavor: REDACTED
      image:
        id: REDACTED
      sshKeyName: REDACTED
      identityRef:
        name: capi-quickstart
        cloudName: openstack
      ports:
      - network:
          id: REDACTED
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            cloud-provider: external
          name: '{{ local_hostname }}'
---
apiVersion: v1
data:
  clouds.yaml: REDACTED # BASE64 ENCODED
kind: Secret
metadata:
  name: capi-quickstart
  namespace: default
```

## Technical considerations

The Cluster API OpenStack infrastructure provider supports starting from [v0.8.0](https://github.com/kubernetes-sigs/cluster-api-provider-openstack/releases/tag/v0.8.0) the ability to allow update of OpenStackCluster API server fixed IP.
This is required since Kamaji Control Plane provider is taking care of this task. 

Once applying the `KamajiControlPlane` manifest, pay attention to performing these actions:

- KubeletPreferredAddress must be set as below because dns resolution does not work with Hostname
```yaml
  kubelet:
    preferredAddressTypes:
      - InternalIP
      - ExternalIP
      - Hostname
```
- Kamaji can create a OpenStack LoadBalancer for the endpoint of Kamaji Control Plane based on information below.
```yaml
  network:
    serviceType: LoadBalancer
    serviceAnnotations:
      service.beta.kubernetes.io/openstack-internal-load-balancer: "true"
      loadbalancer.openstack.org/floating-network-id: REDACTED
```

Once applying the `OpenStackCluster` manifest, pay attention to performing these actions:

- LoadBalancer or FloatingIP functionality for the endpoint of Control Plane provided by the `OpenStackCluster` resource must be disabled because Kamaji creates the endpoint of Control Plane and updates apiServerFixedIP based on it.
```yaml
  apiServerLoadBalancer:
    enabled: false
  disableAPIServerFloatingIP: true
  disableExternalNetwork: true
  apiServerFixedIP: ""
```
- security group rules need to be added for CNI such as Calico if you need.
```yaml
managedSecurityGroups:
    allowAllInClusterTraffic: false
    allNodesSecurityGroupRules:
    - remoteManagedGroups:
      - worker
      direction: ingress
      etherType: IPv4
      name: BGP
      portRangeMin: 179
      portRangeMax: 179
      protocol: "tcp"
      description: "Allow BGP among workers"
    - remoteManagedGroups:
      - worker
      direction: ingress
      etherType: IPv4
      name: IP-in-IP
      protocol: "4"
      description: "Allow IP-in-IP among workers"
```

Once the cluster has been provisioned, you need to install the [OpenStack Cloud Controller Manager](https://github.com/kubernetes/cloud-provider-openstack/blob/master/docs/openstack-cloud-controller-manager/using-openstack-cloud-controller-manager.md).
You can follow the documentation available in the [CAPO documentation website](https://cluster-api-openstack.sigs.k8s.io/topics/external-cloud-provider).

Once applying the manifests, pay attention to performing these actions:

- the `nodeSelector` must be removed, or updated according to your topology
```yaml
  nodeSelector:
    node-role.kubernetes.io/control-plane: ""
```
- the following toleration must be added to let it deployed to the worker nodes
```yaml
  - effect: NoSchedule
    key: node.cluster.x-k8s.io/uninitialized
```

These steps are mandatory since with Kamaji there will not be Control Plane nodes, thus the Cloud Controller Manager will run into worker nodes.