# Kamaji and KubeVirt

The Kamaji Control Plane provider was able to create a _KubeVirt_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                          READY  SEVERITY  REASON  SINCE  MESSAGE                                                                                           
Cluster/capi-quickstart                                       True                     7m3s                                                                                                      
├─ClusterInfrastructure - KubevirtCluster/capi-quickstart                                                                                                                                        
├─ControlPlane - KamajiControlPlane/capi-quickstart-kubevirt                                                                                                                                     
└─Workers                                                                                                                                                                                        
  └─MachineDeployment/capi-quickstart-md-0                    True                     9s                                                                                                        
    └─3 Machines...                                           True                     50s    See capi-quickstart-md-0-795b94c44fxspk46-btcbb, capi-quickstart-md-0-795b94c44fxspk46-ld6sj, ...
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
      - 10.243.0.0/16
    services:
      cidrBlocks:
      - 10.95.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
    kind: KamajiControlPlane
    name: capi-quickstart-kubevirt
    namespace: default
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: KubevirtCluster
    name: capi-quickstart
    namespace: default
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: KubevirtCluster
metadata:
  annotations:
    cluster.x-k8s.io/managed-by: kamaji
  name: capi-quickstart
  namespace: default
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: capi-quickstart-kubevirt
  namespace: default
spec:
  dataStoreName: default
  addons:
    coreDNS: { }
    kubeProxy: { }
  kubelet:
    cgroupfs: systemd
    preferredAddressTypes:
    - InternalIP
    - ExternalIP
  network:
    serviceType: LoadBalancer
  deployment:
  replicas: 2
  version: 1.23.10
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: KubevirtMachineTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      virtualMachineBootstrapCheck:
        checkStrategy: ssh
      virtualMachineTemplate:
        metadata:
          namespace: default
        spec:
          runStrategy: Always
          template:
            spec:
              domain:
                cpu:
                  cores: 2
                devices:
                  disks:
                  - disk:
                      bus: virtio
                    name: containervolume
                  networkInterfaceMultiqueue: true
                memory:
                  guest: 4Gi
              evictionStrategy: External
              volumes:
              - containerDisk:
                  image: quay.io/capk/ubuntu-2004-container-disk:v1.23.10
                name: containervolume
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
          kubeletExtraArgs: {}
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  clusterName: capi-quickstart
  replicas: 3
  selector:
    matchLabels: null
  template:
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: capi-quickstart-md-0
          namespace: default
      clusterName: capi-quickstart
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
        kind: KubevirtMachineTemplate
        name: capi-quickstart-md-0
        namespace: default
      version: v1.23.10
```

## Technical considerations

According to the said manifests, the `KubevirtCluster` resource must be marked as externally managed using the annotation `cluster.x-k8s.io/managed-by`.

The reason behind that is that the CAPI KubeVirt provider is automatically creating a "Load Balancer" service for the Control Plane VMs, although these will be missing since Kamaji will take care of the Control Plane component.

This is made possible by the said annotation and will let the Kamaji Control Plane provider to patch the `KubeVirtCluster` with the address and port provisioned by Kamaji itself, with no extra `Service` resources.

```
$: kubectl get svc
NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
capi-quickstart-kubevirt   LoadBalancer   10.96.215.239   172.18.255.200   6443:32525/TCP   12m
kubernetes                 ClusterIP      10.96.0.1       <none>           443/TCP          41m
```

> Please, notice the missing `capi-quickstart-lb` Service that would be expected with a regular provisioned cluster.
