package capv

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
)

const (
	clusterName = "affectionate-albattani"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	newpath := filepath.Join(home, ConfigDir, clusterName, "/")
	os.MkdirAll(newpath, os.ModePerm)
	err = ioutil.WriteFile(newpath+"/"+clusterName+"-base.yaml", []byte(baseYaml), 0644)
}

func shutdown() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	newpath := filepath.Join(home, ConfigDir, clusterName)
	err = os.RemoveAll(newpath)
	if err != nil {
		log.Fatal(err)
	}
}

func TestExec(t *testing.T) {
	err := injectTridentPrereqs(clusterName, "test", "", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	// TODO add tests here
}

const baseYaml = `apiVersion: cluster.x-k8s.io/v1alpha3
kind: Cluster
metadata:
  name: affectionate-albattani
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
      - 192.168.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1alpha3
    kind: KubeadmControlPlane
    name: affectionate-albattani
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
    kind: VSphereCluster
    name: affectionate-albattani
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: HAProxyLoadBalancer
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: affectionate-albattani
  name: affectionate-albattani
  namespace: default
spec:
  user:
    name: capv
    authorizedKeys:
    - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDW7BP54hSp3TrQjQq7O+oprZdXH8zbKBww/YJyCD9ksM/Y3BiFaCDwzN/vcRSslkn0kJDUq7TxmKp9bEZLTXqAiRe7GflNGoiAUuNY9EWnxt305HIkBs+OEdV6KDtnlm9sRAADflzbDi6YiMjbwNcfoRoxTgpo6BNlzv9Y3prDXiwEjxvosK+4WWIVTTEh33nNvQ5iQhPqBNgURmjQx9EDXFIRdZzA8OykPNLIqFdzmxGZWWxFbW/n6nEl/96b6w7Gx0YgzTSLs+6WAQl8SMP9l22L6puitpjihRw9cWRJ9r6x1eLqgc5Sv7gDKOMXghbmS6hy+AtrxCPPJgq7Mguc5bPAqTZlYMy98dxpHVqtAnBso/9aLOzAXX6At/0QUIwMP693B11NTGniIMtBxnD/yWvGoxTXNmXcTvj13cTzSv9czaGSJ+MTRIugtgyouZADfs8v59NV9KoaEq8umy6WEhmtw5wkjzvC5KK4N2bsM1N+8lSIKxYWxWZFsdYBP8ep442Z/2T5R8y8c5cp7tQqqapDt8JPJ0OPq3sn30BO3X8MgvmoB39j4Cqok1y9VuouPH4RalRLMR7KrASdlFengjt0vWBUoNaEuxRdJR2eOM6SpZh6YGqLdQH1MLaBOzDTH2tTLyTXCOSJpve6ZHOPbjS2BF34a1Kj52NTFtiYTw==
  virtualMachineConfiguration:
    cloneMode: linkedClone
    datacenter: NetApp-HCI-Datacenter-01
    datastore: NetApp-HCI-Datastore-02
    diskGiB: 25
    folder: k8s
    memoryMiB: 8192
    network:
      devices:
      - dhcp4: true
        networkName: NetApp HCI VDS 01-HCI_Internal_mNode_Network
    numCPUs: 2
    resourcePool: capi
    server: 172.60.0.150
    template: capv-haproxy-v0.6.0-rc.2
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: VSphereCluster
metadata:
  name: affectionate-albattani
  namespace: default
spec:
  cloudProviderConfiguration:
    global:
      insecure: true
      secretName: cloud-provider-vsphere-credentials
      secretNamespace: kube-system
    network:
      name: NetApp HCI VDS 01-HCI_Internal_mNode_Network
    providerConfig:
      cloud:
        controllerImage: gcr.io/cloud-provider-vsphere/cpi/release/manager:v1.0.0
      storage:
        attacherImage: quay.io/k8scsi/csi-attacher:v1.1.1
        controllerImage: gcr.io/cloud-provider-vsphere/csi/release/driver:v1.0.2
        livenessProbeImage: quay.io/k8scsi/livenessprobe:v1.1.0
        metadataSyncerImage: gcr.io/cloud-provider-vsphere/csi/release/syncer:v1.0.2
        nodeDriverImage: gcr.io/cloud-provider-vsphere/csi/release/driver:v1.0.2
        provisionerImage: quay.io/k8scsi/csi-provisioner:v1.2.1
        registrarImage: quay.io/k8scsi/csi-node-driver-registrar:v1.1.0
    virtualCenter:
      172.60.0.150:
        datacenters: NetApp-HCI-Datacenter-01
    workspace:
      datacenter: NetApp-HCI-Datacenter-01
      datastore: NetApp-HCI-Datastore-02
      folder: k8s
      resourcePool: capi
      server: 172.60.0.150
  loadBalancerRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
    kind: HAProxyLoadBalancer
    name: affectionate-albattani
  server: 172.60.0.150
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
kind: VSphereMachineTemplate
metadata:
  name: affectionate-albattani
  namespace: default
spec:
  template:
    spec:
      cloneMode: linkedClone
      datacenter: NetApp-HCI-Datacenter-01
      datastore: NetApp-HCI-Datastore-02
      diskGiB: 25
      folder: k8s
      memoryMiB: 8192
      network:
        devices:
        - dhcp4: true
          networkName: NetApp HCI VDS 01-HCI_Internal_mNode_Network
      numCPUs: 2
      resourcePool: capi
      server: 172.60.0.150
      template: ubuntu-1804-kube-v1.17.3
---
apiVersion: controlplane.cluster.x-k8s.io/v1alpha3
kind: KubeadmControlPlane
metadata:
  name: affectionate-albattani
  namespace: default
spec:
  infrastructureTemplate:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
    kind: VSphereMachineTemplate
    name: affectionate-albattani
  kubeadmConfigSpec:
    clusterConfiguration:
      apiServer:
        extraArgs:
          cloud-provider: external
      controllerManager:
        extraArgs:
          cloud-provider: external
    initConfiguration:
      nodeRegistration:
        criSocket: /var/run/containerd/containerd.sock
        kubeletExtraArgs:
          cloud-provider: external
        name: '{{ ds.meta_data.hostname }}'
    joinConfiguration:
      nodeRegistration:
        criSocket: /var/run/containerd/containerd.sock
        kubeletExtraArgs:
          cloud-provider: external
        name: '{{ ds.meta_data.hostname }}'
    preKubeadmCommands:
    - hostname "{{ ds.meta_data.hostname }}"
    - echo "::1         ipv6-localhost ipv6-loopback" >/etc/hosts
    - echo "127.0.0.1   localhost" >>/etc/hosts
    - echo "127.0.0.1   {{ ds.meta_data.hostname }}" >>/etc/hosts
    - echo "{{ ds.meta_data.hostname }}" >/etc/hostname
    users:
    - name: capv
      sshAuthorizedKeys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDW7BP54hSp3TrQjQq7O+oprZdXH8zbKBww/YJyCD9ksM/Y3BiFaCDwzN/vcRSslkn0kJDUq7TxmKp9bEZLTXqAiRe7GflNGoiAUuNY9EWnxt305HIkBs+OEdV6KDtnlm9sRAADflzbDi6YiMjbwNcfoRoxTgpo6BNlzv9Y3prDXiwEjxvosK+4WWIVTTEh33nNvQ5iQhPqBNgURmjQx9EDXFIRdZzA8OykPNLIqFdzmxGZWWxFbW/n6nEl/96b6w7Gx0YgzTSLs+6WAQl8SMP9l22L6puitpjihRw9cWRJ9r6x1eLqgc5Sv7gDKOMXghbmS6hy+AtrxCPPJgq7Mguc5bPAqTZlYMy98dxpHVqtAnBso/9aLOzAXX6At/0QUIwMP693B11NTGniIMtBxnD/yWvGoxTXNmXcTvj13cTzSv9czaGSJ+MTRIugtgyouZADfs8v59NV9KoaEq8umy6WEhmtw5wkjzvC5KK4N2bsM1N+8lSIKxYWxWZFsdYBP8ep442Z/2T5R8y8c5cp7tQqqapDt8JPJ0OPq3sn30BO3X8MgvmoB39j4Cqok1y9VuouPH4RalRLMR7KrASdlFengjt0vWBUoNaEuxRdJR2eOM6SpZh6YGqLdQH1MLaBOzDTH2tTLyTXCOSJpve6ZHOPbjS2BF34a1Kj52NTFtiYTw== jacob.weinstock@netapp.com
      sudo: ALL=(ALL) NOPASSWD:ALL
  replicas: 1
  version: v1.17.3
---
apiVersion: bootstrap.cluster.x-k8s.io/v1alpha3
kind: KubeadmConfigTemplate
metadata:
  name: affectionate-albattani-md-0
  namespace: default
spec:
  template:
    spec:
      joinConfiguration:
        nodeRegistration:
          criSocket: /var/run/containerd/containerd.sock
          kubeletExtraArgs:
            cloud-provider: external
          name: '{{ ds.meta_data.hostname }}'
      preKubeadmCommands:
      - hostname "{{ ds.meta_data.hostname }}"
      - echo "::1         ipv6-localhost ipv6-loopback" >/etc/hosts
      - echo "127.0.0.1   localhost" >>/etc/hosts
      - echo "127.0.0.1   {{ ds.meta_data.hostname }}" >>/etc/hosts
      - echo "{{ ds.meta_data.hostname }}" >/etc/hostname
      users:
      - name: capv
        sshAuthorizedKeys:
        - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDW7BP54hSp3TrQjQq7O+oprZdXH8zbKBww/YJyCD9ksM/Y3BiFaCDwzN/vcRSslkn0kJDUq7TxmKp9bEZLTXqAiRe7GflNGoiAUuNY9EWnxt305HIkBs+OEdV6KDtnlm9sRAADflzbDi6YiMjbwNcfoRoxTgpo6BNlzv9Y3prDXiwEjxvosK+4WWIVTTEh33nNvQ5iQhPqBNgURmjQx9EDXFIRdZzA8OykPNLIqFdzmxGZWWxFbW/n6nEl/96b6w7Gx0YgzTSLs+6WAQl8SMP9l22L6puitpjihRw9cWRJ9r6x1eLqgc5Sv7gDKOMXghbmS6hy+AtrxCPPJgq7Mguc5bPAqTZlYMy98dxpHVqtAnBso/9aLOzAXX6At/0QUIwMP693B11NTGniIMtBxnD/yWvGoxTXNmXcTvj13cTzSv9czaGSJ+MTRIugtgyouZADfs8v59NV9KoaEq8umy6WEhmtw5wkjzvC5KK4N2bsM1N+8lSIKxYWxWZFsdYBP8ep442Z/2T5R8y8c5cp7tQqqapDt8JPJ0OPq3sn30BO3X8MgvmoB39j4Cqok1y9VuouPH4RalRLMR7KrASdlFengjt0vWBUoNaEuxRdJR2eOM6SpZh6YGqLdQH1MLaBOzDTH2tTLyTXCOSJpve6ZHOPbjS2BF34a1Kj52NTFtiYTw== jacob.weinstock@netapp.com
        sudo: ALL=(ALL) NOPASSWD:ALL
---
apiVersion: cluster.x-k8s.io/v1alpha3
kind: MachineDeployment
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: affectionate-albattani
  name: affectionate-albattani-md-0
  namespace: default
spec:
  clusterName: affectionate-albattani
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/cluster-name: affectionate-albattani
  template:
    metadata:
      labels:
        cluster.x-k8s.io/cluster-name: affectionate-albattani
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1alpha3
          kind: KubeadmConfigTemplate
          name: affectionate-albattani-md-0
      clusterName: affectionate-albattani
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1alpha3
        kind: VSphereMachineTemplate
        name: affectionate-albattani
    version: v1.17.3
`
