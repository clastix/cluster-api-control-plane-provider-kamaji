# Kamaji and AWS

The Kamaji Control Plane provider was able to create an _AWS_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                                  READY  SEVERITY  REASON  SINCE  MESSAGE                                                                                           
Cluster/capi-quickstart                                               True                     31m                                                                                                       
├─ClusterInfrastructure - AWSCluster/kamaji-quickstart-control-plane  True                     31m                                                                                                       
├─ControlPlane - KamajiControlPlane/kamaji-aws-127                                                                                                                                       
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
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
    kind: AWSCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: AWSCluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  region: us-east-1
  sshKeyName: default
  controlPlaneLoadBalancer:
    loadBalancerType: disabled
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
      clusterName: capi-quickstart
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
        kind: AWSMachineTemplate
        name: capi-quickstart-md-0
      version: v1.26.0
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: AWSMachineTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      iamInstanceProfile: nodes.cluster-api-provider-aws.sigs.k8s.io
      instanceType: t3.large
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
          name: '{{ ds.meta_data.local_hostname }}'
```

## Technical considerations

The Cluster API AWS infrastructure provider supports starting from [v2.4.0](https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/tag/v2.4.0) the ability to disable the Control Plane load balancer.
This is required since Kamaji Control Plane provider is taking care of this task.

Once the cluster has been provisioned, you need to install the [AWS Cloud Controller Manager](https://github.com/kubernetes/cloud-provider-aws).
You can follow the documentation available in the [CAPA documentation website](https://cluster-api-aws.sigs.k8s.io/topics/external-cloud-provider-with-ebs-csi-driver).

Once applying the manifests, pay attention to performing these actions:

- the `DaemonSet` manifest must be translated into a `Deployment`
- the `nodeSelector` must be removed, or updated according to your topology
- tolerations must be aligned to the workload nodes ones

These steps are mandatory since with Kamaji there will not be Control Plane nodes, thus the Cloud Controller Manager will run into worker ones.

## Running the Kamaji management cluster on AWS

If you're aiming to run the management cluster on AWS the `TenantControlPlane` requires some additional hack due to AWS ELBs CNAME-based ingress.

You can learn more on how to set up the management cluster in this blog post by [CLASTIX](https://clastix.io/post/overcoming-eks-limitations-with-kamaji-on-aws/). 
