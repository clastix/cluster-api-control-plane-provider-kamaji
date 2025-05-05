# Kamaji and Hetzner

The Kamaji Control Plane provider was able to create a _Hetzner_ backed Kubernetes cluster by providing Kamaji Control Planes.

> The Cluster API provider used and tested by Kamaji is the [Syself](https://github.com/syself/cluster-api-provider-hetzner) one.

```
NAME                                                        READY  SEVERITY  REASON  SINCE  MESSAGE                                                                        
Cluster/workload                                            True                     11m                                                                                    
├─ClusterInfrastructure - HetznerCluster/workload                                                                                                                           
├─ControlPlane - KamajiControlPlane/workload-control-plane                                                                                                                  
└─Workers                                                                                                                                                                   
  └─MachineDeployment/workload-md-0                         True                     3m8s                                                                                   
    └─2 Machines...                                         True                     10m    See workload-md-0-5849b98d48xhd4bc-lrlhc, workload-md-0-5849b98d48xhd4bc-txq4j
```

## Example manifests

The said cluster has been created with the following manifests.

```yaml
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: workload-md-0
  namespace: default
spec:
  template:
    spec:
      files:
        - content: |
            net.ipv4.conf.lxc*.rp_filter = 0
          owner: root:root
          path: /etc/sysctl.d/99-cilium.conf
          permissions: "0744"
        - content: |
            overlay
            br_netfilter
          owner: root:root
          path: /etc/modules-load.d/crio.conf
          permissions: "0744"
        - content: |
            version = 2
            [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
              runtime_type = "io.containerd.runc.v2"
            [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
              SystemdCgroup = true
            [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.crun]
              runtime_type = "io.containerd.runc.v2"
            [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.crun.options]
              BinaryName = "crun"
              Root = "/usr/local/sbin"
              SystemdCgroup = true
            [plugins."io.containerd.grpc.v1.cri".containerd]
              default_runtime_name = "crun"
            [plugins."io.containerd.runtime.v1.linux"]
              runtime = "crun"
              runtime_root = "/usr/local/sbin"
          owner: root:root
          path: /etc/containerd/config.toml
          permissions: "0744"
        - content: |
            net.bridge.bridge-nf-call-iptables  = 1
            net.bridge.bridge-nf-call-ip6tables = 1
            net.ipv4.ip_forward                 = 1
          owner: root:root
          path: /etc/sysctl.d/99-kubernetes-cri.conf
          permissions: "0744"
        - content: |
            vm.overcommit_memory=1
            kernel.panic=10
            kernel.panic_on_oops=1
          owner: root:root
          path: /etc/sysctl.d/99-kubelet.conf
          permissions: "0744"
        - content: |
            nameserver 1.1.1.1
            nameserver 1.0.0.1
            nameserver 2606:4700:4700::1111
          owner: root:root
          path: /etc/kubernetes/resolv.conf
          permissions: "0744"
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            anonymous-auth: "false"
            authentication-token-webhook: "true"
            authorization-mode: Webhook
            cloud-provider: external
            event-qps: "5"
            kubeconfig: /etc/kubernetes/kubelet.conf
            max-pods: "220"
            read-only-port: "0"
            resolv-conf: /etc/kubernetes/resolv.conf
            rotate-server-certificates: "true"
            tls-cipher-suites: TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_128_GCM_SHA256
      preKubeadmCommands:
        - set -x
        - export CRUN=1.8.4
        - export CONTAINERD=1.7.0
        - export KUBERNETES_VERSION=$(echo v1.25.2 | sed 's/^v//')
        - ARCH=amd64
        - if [ "$(uname -m)" = "aarch64" ]; then ARCH=arm64; fi
        - localectl set-locale LANG=en_US.UTF-8
        - localectl set-locale LANGUAGE=en_US.UTF-8
        - apt-get update -y
        - apt-get -y install at jq unzip wget socat mtr logrotate apt-transport-https
        - sed -i '/swap/d' /etc/fstab
        - swapoff -a
        - modprobe overlay && modprobe br_netfilter && sysctl --system
        - wget https://github.com/containerd/containerd/releases/download/v$CONTAINERD/cri-containerd-cni-$CONTAINERD-linux-$ARCH.tar.gz
        - wget https://github.com/containerd/containerd/releases/download/v$CONTAINERD/cri-containerd-cni-$CONTAINERD-linux-$ARCH.tar.gz.sha256sum
        - sha256sum --check cri-containerd-cni-$CONTAINERD-linux-$ARCH.tar.gz.sha256sum
        - tar --no-overwrite-dir -C / -xzf cri-containerd-cni-$CONTAINERD-linux-$ARCH.tar.gz
        - rm -f cri-containerd-cni-$CONTAINERD-linux-$ARCH.tar.gz cri-containerd-cni-$CONTAINERD-linux-$ARCH.tar.gz.sha256sum
        - wget https://github.com/containers/crun/releases/download/$CRUN/crun-$CRUN-linux-$ARCH
          -O /usr/local/sbin/crun && chmod +x /usr/local/sbin/crun
        - rm -f /etc/cni/net.d/10-containerd-net.conflist
        - chmod -R 644 /etc/cni && chown -R root:root /etc/cni
        - systemctl daemon-reload && systemctl enable containerd && systemctl start
          containerd
        - curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key
          add -
        - echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee -a
          /etc/apt/sources.list.d/kubernetes.list
        - apt-get update
        - apt-get install -y kubelet=$KUBERNETES_VERSION-00 kubeadm=$KUBERNETES_VERSION-00
          kubectl=$KUBERNETES_VERSION-00  bash-completion && apt-mark hold kubelet kubectl
          kubeadm && systemctl enable kubelet
        - kubeadm config images pull --kubernetes-version $KUBERNETES_VERSION
        - echo 'source <(kubectl completion bash)' >>~/.bashrc
        - echo 'export KUBECONFIG=/etc/kubernetes/admin.conf' >>~/.bashrc
        - apt-get -y autoremove && apt-get -y clean all
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: workload
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
        - 10.244.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KamajiControlPlane
    name: workload-control-plane
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: HetznerCluster
    name: workload
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  labels:
    nodepool: workload-md-0
  name: workload-md-0
  namespace: default
spec:
  clusterName: workload
  replicas: 2
  selector:
    matchLabels: null
  template:
    metadata:
      labels:
        nodepool: workload-md-0
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: workload-md-0
      clusterName: workload
      failureDomain: fsn1
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: HCloudMachineTemplate
        name: workload-md-0
      version: v1.25.2
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineHealthCheck
metadata:
  name: workload-md-0-unhealthy-5m
  namespace: default
spec:
  clusterName: workload
  maxUnhealthy: 100%
  nodeStartupTimeout: 10m
  remediationTemplate:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: HCloudRemediationTemplate
    name: worker-remediation-request
  selector:
    matchLabels:
      nodepool: workload-md-0
  unhealthyConditions:
    - status: Unknown
      timeout: 180s
      type: Ready
    - status: "False"
      timeout: 180s
      type: Ready
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: HCloudMachineTemplate
metadata:
  name: workload-md-0
  namespace: default
spec:
  template:
    spec:
      imageName: ubuntu-22.04
      placementGroupName: md-0
      type: cpx31
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: HCloudRemediationTemplate
metadata:
  name: worker-remediation-request
  namespace: default
spec:
  template:
    spec:
      strategy:
        retryLimit: 1
        timeout: 180s
        type: Reboot
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: HetznerCluster
metadata:
  annotations:
    capi.syself.com/allow-empty-control-plane-address: "true"
  name: workload
  namespace: default
spec:
  controlPlaneRegions: []
  controlPlaneLoadBalancer:
    enabled: false
  hcloudNetwork:
    enabled: false
  hcloudPlacementGroups:
    - name: md-0
      type: spread
  hetznerSecretRef:
    key:
      hcloudToken: hcloud
    name: hetzner
  sshKeys:
    hcloud:
      - name: prometherion@akephalos
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: workload-control-plane
  namespace: default
spec:
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
    serviceType: LoadBalancer
    serviceAnnotations:
      load-balancer.hetzner.cloud/location: fsn1
  deployment:
  replicas: 2
  version: 1.25.2
```

## Technical considerations

If the __management__ cluster is deployed on HetznerCloud the resulting Kamaji Control Plane Service object must have the following annotation: `load-balancer.hetzner.cloud/location=fsn1`.
The region (`fsn1`) may vary according to your deployed region.

If you're offloading Kamaji in creating the Load Balancer on your behalf, pay attention in disabling the default Load Balancer using the field `HetznerCloud.controlPlaneLoadBalancer.enabled=false`.
The field `HetznerCluster.spec.controlPlaneEndpoint` will be populated once the endpoint is ready and available.

Once provisioned, you have to deploy the [Hetzner Cloud Controller manager](https://github.com/syself/hetzner-cloud-controller-manager) which requires a _Secret_ to interact with the Hetzner API.
Please refer to the official documentation of the project.
