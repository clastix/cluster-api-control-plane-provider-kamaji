# Kamaji and metal-stack

The Kamaji Control Plane provider was able to create a _metal-stack_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                                                   REPLICAS AVAILABLE READY UP TO DATE STATUS REASON     SINCE  MESSAGE 
Cluster/kamaji-tenant-test                                                             3/3      3         3     3          True   Available  3m31s          
├─ClusterInfrastructure - MetalStackCluster/kamaji-tenant-test                                                                                              
├─ControlPlane - KamajiControlPlane/kamaji-tenant-test                                 2/2                2     2                                           
└─Workers                                                                                                                                                   
  └─MachineDeployment/kamaji-tenant-test-md-0                                          1/1      1         1     1          True   Available  3m31s          
    └─Machine/kamaji-tenant-test-md-0-zlkmb-m9kxw                                      1        1         1     1          True   Ready      3m31s          
      └─MachineInfrastructure - MetalStackMachine/kamaji-tenant-test-md-0-zlkmb-m9kxw 
```

## Example manifests

This example needs a Service Load Balancer (MetalLB, Kube-VIP, ...) installed in your management cluster to provide the VIP (`CONTROL_PLANE_IP`) for the Kamaji Tenant API server.

```yaml
---
kind: Cluster
apiVersion: cluster.x-k8s.io/v1beta2
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: '${CLUSTER_NAME}'
  name: '${CLUSTER_NAME}'
  namespace: '${NAMESPACE}'
spec:
  controlPlaneRef:
    apiGroup: controlplane.cluster.x-k8s.io
    kind: KamajiControlPlane
    name: '${CLUSTER_NAME}'
  clusterNetwork:
    pods:
      cidrBlocks:
        - '${PODS_CIDR}'
    services:
      cidrBlocks:
        - '${SERVICES_CIDR}'
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: MetalStackCluster
    name: ${CLUSTER_NAME}
---
kind: KamajiControlPlane
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
metadata:
  name: '${CLUSTER_NAME}'
  namespace: '${CLUSTER_NAMESPACE}'
spec:
  apiServer: {}
  controllerManager:
    extraArgs:
      - --cloud-provider=external
  dataStoreName: default
  addons:
    coreDNS: {}
    kubeProxy: {}
    konnectivity: {}
  kubelet:
    cgroupfs: systemd
    preferredAddressTypes:
      - InternalIP
  network:
    serviceType: LoadBalancer
    serviceAddress: ${CONTROL_PLANE_IP}
    serviceAnnotations:
      metallb.io/address-pool: ${CLUSTER_NAME}-vip
  version: ${KUBERNETES_VERSION}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: MetalStackCluster
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
spec:
  projectID: ${METAL_PROJECT_ID}
  partition: ${METAL_PARTITION}
  nodeNetworkID: ${METAL_NODE_NETWORK_ID:=null}
  controlPlaneIP: ${CONTROL_PLANE_IP}
  controlPlaneEndpoint:
    host: ${CONTROL_PLANE_IP}
    port: 6443
  firewallDeploymentRef:
    name: ${CLUSTER_NAME}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: MetalStackFirewallDeployment
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
spec:
  autoUpdate:
    machineImage: true
  firewallTemplateRef:
    name: ${CLUSTER_NAME}
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: MetalStackFirewallTemplate
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
spec:
  image: ${FIREWALL_MACHINE_IMAGE}
  size: ${FIREWALL_MACHINE_SIZE}
  networks: ${FIREWALL_EXTERNAL_NETWORKS}
  partition: ${METAL_PARTITION}
  project: ${METAL_PROJECT_ID}
  initialRuleSet:
    egress:
      - comment: allow outgoing HTTP and HTTPS traffic
        ports:
          - 80
          - 443
        protocol: TCP
        to:
          - 0.0.0.0/0
      - comment: allow outgoing DNS traffic via TCP
        ports:
          - 53
        protocol: TCP
        to:
          - 0.0.0.0/0
      - comment: allow outgoing traffic to Kamaji tenant API server for kubeadm join and kubelet
        ports:
          - 6443
        protocol: TCP
        to:
          - 0.0.0.0/0
      - comment: allow outgoing traffic to Konnectivity server for control-plane-to-worker tunnels
        ports:
          - 8132
        protocol: TCP
        to:
          - 0.0.0.0/0
      - comment: allow outgoing traffic to metal-api for CCM and CSI
        ports:
          - 8080
        protocol: TCP
        to:
          - 0.0.0.0/0
      - comment: allow outgoing DNS and NTP traffic via UDP
        ports:
          - 53
          - 123
        protocol: UDP
        to:
          - 0.0.0.0/0

    ingress:
      - comment: allow incoming HTTP and HTTPS traffic
        ports:
          - 80
          - 443
        protocol: TCP
        from:
          - 0.0.0.0/0
      - comment: allow incoming kubelet API traffic from Kamaji control plane
        ports:
          - 10250
        protocol: TCP
        from:
          - 0.0.0.0/0
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: MetalStackMachineTemplate
metadata:
  name: ${CLUSTER_NAME}-worker
  namespace: ${NAMESPACE}
spec:
  template:
    spec:
      size: ${WORKER_MACHINE_SIZE}
      image: ${WORKER_MACHINE_IMAGE}
---
apiVersion: cluster.x-k8s.io/v1beta2
kind: MachineDeployment
metadata:
  name: ${CLUSTER_NAME}-md-0
  labels:
    cluster.x-k8s.io/cluster-name: ${CLUSTER_NAME}
    nodepool: nodepool-0
spec:
  clusterName: ${CLUSTER_NAME}
  replicas: ${WORKER_MACHINE_COUNT}
  selector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: ${CLUSTER_NAME}
      nodepool: nodepool-0
  template:
    metadata:
      labels:
        cluster.x-k8s.io/cluster-name: ${CLUSTER_NAME}
        nodepool: nodepool-0
    spec:
      clusterName: ${CLUSTER_NAME}
      version: "${KUBERNETES_VERSION}"
      bootstrap:
        configRef:
          name: ${CLUSTER_NAME}-md-0
          apiGroup: bootstrap.cluster.x-k8s.io
          kind: KubeadmConfigTemplate
      infrastructureRef:
        name: ${CLUSTER_NAME}-worker
        apiGroup: infrastructure.cluster.x-k8s.io
        kind: MetalStackMachineTemplate
      deletion:
        nodeDrainTimeoutSeconds: 120
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta2
kind: KubeadmConfigTemplate
metadata:
  name: ${CLUSTER_NAME}-md-0
spec:
  template:
    spec:
      format: ignition
      clusterConfiguration:
        controllerManager:
          extraArgs:
            - name: cloud-provider
              value: external
        controlPlaneEndpoint: ${CONTROL_PLANE_IP}
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            - name: cloud-provider
              value: external
            - name: feature-gates
              value: "KubeletCrashLoopBackOffMax=true,KubeletEnsureSecretPulledImages=true"
```

## Technical considerations

The `KamajiControlPlane` requires a valid service address (`spec.network.serviceAddress`) and an external load balancer configuration in the management cluster to be reachable by the worker nodes. Usually, a solution like MetalLB or Kube-VIP is deployed in the management cluster to expose the Kamaji Tenant API Servers.
To maintain a correct flow for Cluster API, you have to know the Kamaji Tenant Control Plane address/port in advance, in order to provide it to the `MetalStackCluster`.

In regard to the address, the following values must be the same:

- `KamajiControlPlane.spec.network.serviceAddress`
- `MetalStackCluster.spec.controlPlaneEndpoint.host`

The same applies for the Kubernetes API Server binding port:

- `Cluster.spec.clusterNetwork.apiServerPort`
- `MetalStackCluster.spec.controlPlaneEndpoint.port`

Additionally, egress firewall rules on the `MetalStackFirewallTemplate` must be configured to allow outgoing traffic to:
1. The Kamaji Tenant API Server (`6443`) to allow `kubeadm join` and `kubelet` communication.
2. The Konnectivity Server (`8132`) for control-plane to worker node tunnels.

> [!NOTE]
> For a _metal-stack_ deployment to become fully operational and to allow pods to be scheduled, a CNI-Plugin (like Calico) and [`metal-ccm`](https://github.com/metal-stack/metal-ccm) need to be deployed into the tenant cluster.

## Development and Testing Environment

The _metal-stack_ project provides a fully virtual containerlab-based test environment called `capi-lab`, that makes it easy to explore the Kamaji integration locally. 

The `capi-lab` environment provides a `kamaji` flavor that runs:
- **Kamaji** as the Control Plane provider on a Kind management cluster
- **MetalLB** to provide Virtual IPs for the Tenant Control Planes
- Localized mini-lab nodes acting as bare-metal worker machines and firewalls via the _metal-stack_ provider

You can find the setup instructions in the [`cluster-api-provider-metal-stack` DEVELOPMENT docs](https://github.com/metal-stack/cluster-api-provider-metal-stack/blob/main/DEVELOPMENT.md).