# Kamaji and Proxmox

The Kamaji Control Plane provider allows creating a _Proxmox by IONOS Cloud_ backed Kubernetes cluster by providing Kamaji Control Planes.

## Example manifests

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: proxmox-quickstart
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
      - REDACTED/REDACTED
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
    kind: KamajiControlPlane
    name: proxmox-quickstart
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: ProxmoxCluster
    name: proxmox-quickstart
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: proxmox-quickstart
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
  network:
    serviceType: LoadBalancer
  deployment:
  replicas: 2
  version: 1.29.7
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: ProxmoxCluster
metadata:
  name: proxmox-quickstart
  namespace: default
spec:
  allowedNodes:
  - pve
  dnsServers:
  - REDACTED
  - REDACTED
  externalManagedControlPlane: true
  ipv4Config:
    addresses:
    - REDACTED-REDACTED
    gateway: REDACTED
    prefix: REDACTED
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: proxmox-quickstart-workers
  namespace: default
spec:
  clusterName: proxmox-quickstart
  replicas: 2
  selector:
    matchLabels: null
  template:
    metadata:
      labels:
        node-role.kubernetes.io/node: ""
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: proxmox-quickstart-worker
      clusterName: proxmox-quickstart
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
        kind: ProxmoxMachineTemplate
        name: proxmox-quickstart-worker
      version: v1.29.7
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: ProxmoxMachineTemplate
metadata:
  name: proxmox-quickstart-worker
  namespace: default
spec:
  template:
    spec:
      disks:
        bootVolume:
          disk: scsi0
          sizeGb: REDACTED
      format: qcow2
      full: true
      memoryMiB: REDACTED
      network:
        default:
          bridge: REDACTED
          model: virtio
      numCores: REDACTED
      numSockets: REDACTED
      sourceNode: pve
      templateID: REDACTED
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: proxmox-quickstart-worker
  namespace: default
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            provider-id: proxmox://'{{ ds.meta_data.instance_id }}'
      users:
      - name: root
        sshAuthorizedKeys:
        - REDACTED
```

## Technical considerations

The `ProxmoxCluster` `spec.externalManagedControlPlane` value must be toggled to true:
such a field is available starting from the v0.6.0 release.
