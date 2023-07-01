# Kamaji and Equinix

The Kamaji Control Plane provider was able to create a _Packet_ (now Equinix) backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                            READY  SEVERITY  REASON  SINCE  MESSAGE                                                                                                   
Cluster/capi-quickstart                                         True                     7m20s                                                                                                             
├─ClusterInfrastructure - PacketCluster/capi-quickstart                                                                                                                                                    
├─ControlPlane - KamajiControlPlane/capi-equinix-control-plane                                                                                                                                             
└─Workers                                                                                                                                                                                                  
  └─MachineDeployment/capi-quickstart-worker-a                  True                     2m1s                                                                                                              
    └─3 Machines...                                             True                     4m6s   See capi-quickstart-worker-a-777b67559bxq6dw9-2rxc4, capi-quickstart-worker-a-777b67559bxq6dw9-fw29m, ..
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
    services:
      cidrBlocks:
        - 172.26.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KamajiControlPlane
    name: capi-equinix-control-plane
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: PacketCluster
    name: capi-quickstart
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: PacketCluster
metadata:
  annotations:
    cluster.x-k8s.io/managed-by: kamaji
  name: capi-quickstart
  namespace: default
spec:
  metro: REDACTED
  projectID: REDACTED
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: capi-equinix-control-plane
  namespace: default
spec:
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
    serviceType: LoadBalancer
  deployment:
  replicas: 2
  version: 1.27.0
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: capi-quickstart
    pool: worker-a
  name: capi-quickstart-worker-a
  namespace: default
spec:
  clusterName: capi-quickstart
  replicas: 3
  selector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: capi-quickstart
      pool: worker-a
  template:
    metadata:
      labels:
        cluster.x-k8s.io/cluster-name: capi-quickstart
        pool: worker-a
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: capi-quickstart-worker-a
      clusterName: capi-quickstart
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: PacketMachineTemplate
        name: capi-quickstart-worker-a
      version: v1.27.0
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: PacketMachineTemplate
metadata:
  name: capi-quickstart-worker-a
  namespace: default
spec:
  template:
    spec:
      billingCycle: hourly
      machineType: c3.medium.x86
      os: ubuntu_20_04
      sshKeys:
        - ssh-rsa REDACTED REDACTED@REDACTED
      tags: []
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: capi-quickstart-worker-a
  namespace: default
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            cloud-provider: external
            provider-id: equinixmetal://{{ `{{ v1.instance_id }}` }}
      preKubeadmCommands:
        - |
          sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab
          swapoff -a
          mount -a
          cat <<EOF > /etc/modules-load.d/containerd.conf
          overlay
          br_netfilter
          EOF
          modprobe overlay
          modprobe br_netfilter
          cat <<EOF > /etc/sysctl.d/99-kubernetes-cri.conf
          net.bridge.bridge-nf-call-iptables  = 1
          net.ipv4.ip_forward                 = 1
          net.bridge.bridge-nf-call-ip6tables = 1
          EOF
          sysctl --system
          export DEBIAN_FRONTEND=noninteractive
          apt-get update -y
          apt-get remove -y docker docker-engine containerd runc
          apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release linux-generic jq
          install -m 0755 -d /etc/apt/keyrings
          curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
          curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /etc/apt/keyrings/kubernetes-archive-keyring.gpg
          chmod a+r /etc/apt/keyrings/docker.gpg
          chmod a+r /etc/apt/keyrings/kubernetes-archive-keyring.gpg
          echo "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" > /etc/apt/sources.list.d/docker.list
          echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
          apt-get update -y
          TRIMMED_KUBERNETES_VERSION=$(echo {{ .kubernetesVersion }} | sed 's/\./\\\\./g' | sed 's/^v//')
          RESOLVED_KUBERNETES_VERSION=$(apt-cache madison kubelet | awk -v VERSION=${TRIMMED_KUBERNETES_VERSION} '$3~ VERSION { print $3 }' | head -n1)
          apt-get install -y containerd.io kubelet=${RESOLVED_KUBERNETES_VERSION} kubeadm=${RESOLVED_KUBERNETES_VERSION} kubectl=${RESOLVED_KUBERNETES_VERSION}
          cat  <<EOF > /etc/crictl.yaml
          runtime-endpoint: unix:///run/containerd/containerd.sock
          image-endpoint: unix:///run/containerd/containerd.sock
          EOF
          containerd config default > /etc/containerd/config.toml
          sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
          sed -i "s,sandbox_image.*$,sandbox_image = \"$(kubeadm config images list | grep pause | sort -r | head -n1)\"," /etc/containerd/config.toml
          systemctl restart containerd
```

## Technical considerations

According to the said manifests, the `PacketCluster` resource must be marked as externally managed using the annotation `cluster.x-k8s.io/managed-by`.

The reason behind that is that the CAPI Packet provider is automatically creating a "VIP Manager" service for the Control Plane VMs.
However, Kamaji is providing Kubernetes Control Plane as a Service and taking care of the creation of the required Load Balancer from the management cluster.

Thanks to the said annotation, the Kamaji Control Plane provider will wait for the Control Plane address reachability, and patch the `PacketCluster` with the right address, besides marking the Infrastructure as ready to continue the provisioning process.
