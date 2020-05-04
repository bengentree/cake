package vsphere

const (
	uploadPort                   string = "50000"
	commandPort                  string = "50001"
	uploadConfigPort             string = "50002"
	remoteExecutable             string = "/tmp/cake"
	remoteConfig                 string = "~/.cake.yaml"
	remoteConfigRoot             string = "/root/.cake.yaml"
	baseFolder                   string = "cake"
	templatesFolder              string = "templates"
	workloadsFolder              string = "workloads"
	mgmtFolder                   string = "mgmt"
	bootstrapFolder              string = "bootstrap"
	bootstrapVMName              string = "BootstrapVM"
	uploadFileCmd                string = "socat -u TCP-LISTEN:%s,fork CREATE:%s,group=root,perm=0755 & disown"
	runRemoteCmd                 string = "socat TCP-LISTEN:%s,reuseaddr,fork EXEC:'/bin/bash',pty,setsid,setpgid,stderr,ctty & disown"
	runLocalCakeCmd              string = "%s deploy --local --deployment-type %s > /tmp/cake.out"
	cakeLinuxBinaryPkgerLocation string = "/cake-linux-embedded"
	capvClusterctlVersion        string = "v0.3.3"
	capvKindVersion              string = "v0.7.0"
	rkeControlNodePrefix         string = "controlPlaneNode"
	rkeWorkerNodePrefix          string = "workerNode"
	privateKeyToDisk             string = "umask 133; mkdir -p ~/.ssh && umask 177; touch ~/.ssh/id_rsa && echo -e \"%s\" > ~/.ssh/id_rsa"
	rkeBinaryInstall             string = `wget -O /usr/local/bin/rke https://github.com/rancher/rke/releases/download/v1.1.0/rke_linux-amd64 && chmod +x /usr/local/bin/rke`
	rkePrereqs                   string = `curl https://releases.rancher.com/install-docker/18.09.2.sh | sh
for module in br_netfilter ip6_udp_tunnel ip_set ip_set_hash_ip ip_set_hash_net iptable_filter iptable_nat iptable_mangle iptable_raw nf_conntrack_netlink nf_conntrack nf_conntrack_ipv4   nf_defrag_ipv4 nf_nat nf_nat_ipv4 nf_nat_masquerade_ipv4 nfnetlink udp_tunnel veth vxlan x_tables xt_addrtype xt_conntrack xt_comment xt_mark xt_multiport xt_nat xt_recent xt_set  xt_statistic xt_tcpudp;
do
	if ! lsmod | grep -q $module; then
	echo "module $module is not present, installing now...";
		modprobe $module
	fi;
done

echo "net.bridge.bridge-nf-call-iptables=1" >> /etc/sysctl.conf`
)
