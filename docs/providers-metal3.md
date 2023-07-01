# Kamaji and Metal³

The Kamaji Control Plane provider was able to create a _Metal³_ backed Kubernetes cluster by providing Kamaji Control Planes.

```
NAME                                                        READY  SEVERITY  REASON  SINCE  MESSAGE 
Cluster/test1                                               True                     13m             
├─ClusterInfrastructure - Metal3Cluster/test1               True                     14m             
├─ControlPlane - KamajiControlPlane/capi-quickstart-metal3                                           
└─Workers                                                                                            
  └─MachineDeployment/test1                                 True                     2m40s           
    └─Machine/test1-55b88586c9xxqd46-sg844                  True                     6m48s
```

## Example manifests

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: test1
  namespace: metal3
spec:
  clusterNetwork:
    apiServerPort: 6443
    pods:
      cidrBlocks:
        - 192.168.0.0/18
    services:
      cidrBlocks:
        - 10.96.0.0/12
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KamajiControlPlane
    name: capi-quickstart-metal3
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: Metal3Cluster
    name: test1
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: Metal3Cluster
metadata:
  name: test1
  namespace: metal3
spec:
  controlPlaneEndpoint:
    host: 172.18.255.200
    port: 6443
  noCloudProvider: true
---
apiVersion: ipam.metal3.io/v1alpha1
kind: IPPool
metadata:
  name: provisioning-pool
  namespace: metal3
spec:
  clusterName: test1
  namePrefix: test1-prov
  pools:
    - end: 172.22.0.200
      start: 172.22.0.100
  prefix: 24
---
apiVersion: ipam.metal3.io/v1alpha1
kind: IPPool
metadata:
  name: externalv4-pool
  namespace: metal3
spec:
  clusterName: test1
  gateway: 192.168.111.1
  namePrefix: test1-bmv4
  pools:
    - end: 192.168.111.200
      start: 192.168.111.100
  prefix: 24
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha1
kind: KamajiControlPlane
metadata:
  name: capi-quickstart-metal3
  namespace: metal3
spec:
  dataStoreName: default
  kubelet:
    cgroupfs: systemd
    preferredAddressTypes:
      - ExternalIP
      - InternalIP
      - Hostname
  addons:
    coreDNS: { }
    kubeProxy: { }
  network:
    serviceType: LoadBalancer
    serviceAddress: 172.18.255.200
  deployment:
  replicas: 2
  version: 1.27.1
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: test1
    nodepool: nodepool-0
  name: test1
  namespace: metal3
spec:
  clusterName: test1
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: test1
      nodepool: nodepool-0
  template:
    metadata:
      labels:
        cluster.x-k8s.io/cluster-name: test1
        nodepool: nodepool-0
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: KubeadmConfigTemplate
          name: test1-workers
      clusterName: test1
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: Metal3MachineTemplate
        name: test1-workers
      nodeDrainTimeout: 0s
      version: v1.27.1
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: Metal3MachineTemplate
metadata:
  name: test1-workers
  namespace: metal3
spec:
  template:
    spec:
      dataTemplate:
        name: test1-workers-template
      image:
        checksum: http://172.22.0.1/images/CENTOS_9_NODE_IMAGE_K8S_v1.27.1-raw.img.sha256sum
        checksumType: sha256
        format: raw
        url: http://172.22.0.1/images/CENTOS_9_NODE_IMAGE_K8S_v1.27.1-raw.img
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: Metal3DataTemplate
metadata:
  name: test1-workers-template
  namespace: metal3
spec:
  clusterName: test1
  metaData:
    ipAddressesFromIPPool:
      - key: provisioningIP
        name: provisioning-pool
    objectNames:
      - key: name
        object: machine
      - key: local-hostname
        object: machine
      - key: local_hostname
        object: machine
    prefixesFromIPPool:
      - key: provisioningCIDR
        name: provisioning-pool
  networkData:
    links:
      ethernets:
        - id: enp1s0
          macAddress:
            fromHostInterface: enp1s0
          type: phy
        - id: enp2s0
          macAddress:
            fromHostInterface: enp2s0
          type: phy
    networks:
      ipv4:
        - id: externalv4
          ipAddressFromIPPool: externalv4-pool
          link: enp2s0
          routes:
            - gateway:
                fromIPPool: externalv4-pool
              network: 0.0.0.0
              prefix: 0
    services:
      dns:
        - 8.8.8.8
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: KubeadmConfigTemplate
metadata:
  name: test1-workers
  namespace: metal3
spec:
  template:
    spec:
      files:
        - content: |
            #!/bin/bash
            set -e
            url="$1"
            dst="$2"
            filename="$(basename $url)"
            tmpfile="/tmp/$filename"
            curl -sSL -w "%{http_code}" "$url" | sed "s:/usr/bin:/usr/local/bin:g" > /tmp/"$filename"
            http_status=$(cat "$tmpfile" | tail -n 1)
            if [ "$http_status" != "200" ]; then
              echo "Error: unable to retrieve $filename file";
              exit 1;
            else
              cat "$tmpfile"| sed '$d' > "$dst";
            fi
          owner: root:root
          path: /usr/local/bin/retrieve.configuration.files.sh
          permissions: "0755"
        - content: |
            [connection]
            id=eth0
            type=ethernet
            interface-name=eth0
            master=ironicendpoint
            slave-type=bridge
            autoconnect=yes
            autoconnect-priority=999
          owner: root:root
          path: /etc/NetworkManager/system-connections/eth0.nmconnection
          permissions: "0600"
        - content: |
            [connection]
            id=ironicendpoint
            type=bridge
            interface-name=ironicendpoint

            [bridge]
            stp=false

            [ipv4]
            address1={{ ds.meta_data.provisioningIP }}/{{ ds.meta_data.provisioningCIDR }}
            method=manual

            [ipv6]
            addr-gen-mode=eui64
            method=ignore
          owner: root:root
          path: /etc/NetworkManager/system-connections/ironicendpoint.nmconnection
          permissions: "0600"
        - content: |
            [kubernetes]
            name=Kubernetes
            baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
            enabled=1
            gpgcheck=1
            repo_gpgcheck=0
            gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
          owner: root:root
          path: /etc/yum.repos.d/kubernetes.repo
          permissions: "0644"
        - content: |
            [registries.search]
            registries = ['docker.io']

            [registries.insecure]
            registries = ['192.168.111.1:5000']
          owner: root:root
          path: /etc/containers/registries.conf
          permissions: "0644"
      joinConfiguration:
        nodeRegistration:
          kubeletExtraArgs:
            cgroup-driver: systemd
            container-runtime-endpoint: unix:///var/run/crio/crio.sock
            feature-gates: AllAlpha=false
            node-labels: metal3.io/uuid={{ ds.meta_data.uuid }}
            provider-id: metal3://{{ ds.meta_data.providerid }}
            runtime-request-timeout: 5m
          name: '{{ ds.meta_data.name }}'
      preKubeadmCommands:
        - rm /etc/cni/net.d/*
        - systemctl restart NetworkManager.service
        - nmcli connection load /etc/NetworkManager/system-connections/ironicendpoint.nmconnection
        - nmcli connection up ironicendpoint
        - nmcli connection load /etc/NetworkManager/system-connections/eth0.nmconnection
        - nmcli connection up eth0
        - systemctl enable --now crio
        - sleep 30
        - systemctl enable --now kubelet
        - sleep 120
      users:
        - name: metal3
          sshAuthorizedKeys:
            - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC36DCbgvUW001F1jWsWk0JW+Wva3SGgobhr5Uqlf++gAe/YbSnvBCQ/d8JHBGtBwqX9Gry8B+XZu4/NAPosxHk5HLJOTH973ilER77ORdDYD3qc/Litc6AnFu7Ozt028fhOUav5+9W8Jh0Nkw9eA1G8Na07ULUjlczKjerTikHB9+rFoLMeoqIK3hayuy5IYUoGhQzZ1erpu33k+4rw4ALxE4ka69SdsUKhdKRBUQkaQ05g32RtkQw47G2RN0BL3ZrcUdds5AoKi+KxulbycG/p4YxrMXzZ4O3EkkaeKXDs5UcK+6GXKnRFJCne63Dl20lXIavYqiDj5YAl4VpzYZRtsUG53T+/APOn9TvJEbx+CoJAejeff6Ro/jIU+yh19/3okVNVSMjBb9XASzFZD9JiiU1gqMUCu7HKWQvFVaOBRRc7w1MTGj1TKI6okd2afrLOkuLNmhCfHiwHl3/LSSXCUPp/Q03TCv6sKRts7yTjDTQGcLs884PaasvyLdFkbk=
              metal3@metal3
          sudo: ALL=(ALL) NOPASSWD:ALL
```

## Technical considerations

The said cluster has been created in the [Metal³ Development Environment](https://github.com/metal3-io/metal3-dev-env), as well as the CAPI manifests.

The `KamajiControlPlane` requires a valid address (`spec.network.serviceAddress`) in advance that must matches the `Metal3Cluster.spec.controlPlaneEndpoint.host` field.
As with the port, the `Metal3Cluster.spec.controlPlaneEndpoint.port` must matches the `Cluster.spec.clusterNetwork.apiServerPort` value.

Upon a `Metal3Cluster` resource creation, the Cluster API controller retrieve the provided Control Plane endpoint data, thus, the equality is required to ensure a proper setup.
