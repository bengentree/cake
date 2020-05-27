package vsphere

type baseScript struct {
	script         string
	deploymentType string
}

const (
	bootstrapNodeCommandsHeader string = "\n# Bootstrap node commands section"
	baseNodeCommandsHeader      string = "\n# Base node commands section"
	baseScriptHeader            string = "#!/usr/bin/env bash"
	uploadPort                  string = "50000"
	commandPort                 string = "50001"
	uploadConfigPort            string = "50002"
	remoteExecutable            string = "/tmp/cake"
	remoteConfig                string = "~/.cake/cake.yaml"
	remoteConfigRoot            string = "/root/cake.yaml"
	baseFolder                  string = "cake"
	templatesFolder             string = "templates"
	workloadsFolder             string = "workloads"
	mgmtFolder                  string = "mgmt"
	bootstrapFolder             string = "bootstrap"
	bootstrapVMName             string = "BootstrapVM"
	seedISOName                 string = "seed.iso"
	retryFunction               string = `# Retries a command on failure.
# $1 - the max number of attempts
# $2... - the command to run
retry() {
    local -r -i max_attempts="$1"; shift
    local -r cmd="$@"
    local -i attempt_num=1

    until $cmd
    do
        if (( attempt_num == max_attempts ))
        then
            echo "Attempt $attempt_num failed and there are no more attempts left!"
            return 1
        else
            echo "Attempt $attempt_num failed! Trying again in $attempt_num seconds..."
            sleep $(( attempt_num++ ))
        fi
    done
}`
	installSocatCmd string = `# install socat, needed for TCP listeners
retry 10 wget -O /usr/local/bin/socat https://github.com/andrew-d/static-binaries/raw/master/binaries/linux/x86_64/socat
chmod +x /usr/local/bin/socat`
	runCake string = `# wait until cake.yaml exists on disk and then run cake command
until [[ $(stat -c%%s "%s" 2>/dev/null) > 0 ]]
do
	echo "waiting for config file to exist"
	sleep 2
done
until (%s >/dev/null 2>&1);
do
    echo "waiting for cake binary to be ready to run"
    sleep 2
done
sleep 2
%s & disown`
	uploadFileCmd                string = "socat -u TCP-LISTEN:%s,fork CREATE:%s,group=root,perm=0755 & disown"
	runRemoteCmd                 string = "socat TCP-LISTEN:%s,reuseaddr,fork EXEC:'/bin/bash -li',pty,setsid,setpgid,stderr,ctty & disown"
	runLocalCakeCmd              string = "%s deploy %s --local --spec-file %s --progress > /tmp/cake.out"
	cakeLinuxBinaryPkgerLocation string = "/cake-linux-embedded"
	capvClusterctlVersion        string = "v0.3.3"
	capvKindVersion              string = "v0.7.0"
	rkeControlNodePrefix         string = "controlPlaneNode"
	rkeWorkerNodePrefix          string = "workerNode"
	privateKeyToDisk             string = "umask 133; mkdir -p ~/.ssh && umask 177; touch ~/.ssh/id_rsa && echo -e \"%s\" > ~/.ssh/id_rsa"
	rkeBinaryInstall             string = `retry 5 wget -O /usr/local/bin/rke https://github.com/rancher/rke/releases/download/v1.1.1/rke_linux-amd64 && chmod +x /usr/local/bin/rke`
	rkePrereqs                   string = `sh -c 'curl https://releases.rancher.com/install-docker/18.09.2.sh | sh' || curl -L http://get.docker.com | sh
for module in br_netfilter ip6_udp_tunnel ip_set ip_set_hash_ip ip_set_hash_net iptable_filter iptable_nat iptable_mangle iptable_raw nf_conntrack_netlink nf_conntrack nf_conntrack_ipv4   nf_defrag_ipv4 nf_nat nf_nat_ipv4 nf_nat_masquerade_ipv4 nfnetlink udp_tunnel veth vxlan x_tables xt_addrtype xt_conntrack xt_comment xt_mark xt_multiport xt_nat xt_recent xt_set  xt_statistic xt_tcpudp;
do
	if ! lsmod | grep -q $module; then
	echo "module $module is not present, installing now...";
		modprobe $module
	fi;
done

echo "net.bridge.bridge-nf-call-iptables=1" >> /etc/sysctl.conf
usermod -aG docker %s
which kubectl || wget -O /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/%s/bin/linux/amd64/kubectl
chmod +x /usr/local/bin/kubectl`

	helmInstall string = `retry 5 curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh --version %s`
	helmVersion string = "v3.2.1"
)
