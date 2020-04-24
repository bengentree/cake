package vsphere

import (
	"fmt"

	"github.com/vmware/govmomi/object"
	"gopkg.in/yaml.v3"
)

// Prepare the environment for bootstrapping
func (v *MgmtBootstrap) Prepare() error {
	templateFolder, err := v.Session.CreateVMFolder("cake/templates")
	if err != nil {
		return err
	}
	v.Resources["cakeFolder"] = templateFolder["cake"]
	v.Resources["templatesFolder"] = templateFolder["templates"]

	workloadFolder, err := v.Session.CreateVMFolder("cake/workloads")
	if err != nil {
		return err
	}
	v.Resources["workloadsFolder"] = workloadFolder["workloads"]

	mgmtFolder, err := v.Session.CreateVMFolder("cake/mgmt")
	if err != nil {
		return err
	}
	v.Resources["mgmtFolder"] = mgmtFolder["mgmt"]

	bootstrapFolder, err := v.Session.CreateVMFolder("cake/bootstrap")
	if err != nil {
		return err
	}
	v.Resources["bootstrapFolder"] = bootstrapFolder["bootstrap"]

	if v.Folder != "" {
		FolderFromConfig, err := v.Session.CreateVMFolder(v.Folder)
		if err != nil {
			return err
		}
		v.Resources["FolderFromConfig"] = FolderFromConfig[v.Folder]
	}

	v.Session.Folder = bootstrapFolder["bootstrap"]
	//v.Folder = "templates"

	ovas, err := v.Session.DeployOVATemplates(v.OVA.NodeTemplate, v.OVA.LoadbalancerTemplate)

	v.Resources[v.OVA.NodeTemplate] = ovas[v.OVA.NodeTemplate]
	v.Resources[v.OVA.LoadbalancerTemplate] = ovas[v.OVA.LoadbalancerTemplate]

	v.Folder = v.Resources["FolderFromConfig"].(*object.Folder).InventoryPath
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDW7BP54hSp3TrQjQq7O+oprZdXH8zbKBww/YJyCD9ksM/Y3BiFaCDwzN/vcRSslkn0kJDUq7TxmKp9bEZLTXqAiRe7GflNGoiAUuNY9EWnxt305HIkBs+OEdV6KDtnlm9sRAADflzbDi6YiMjbwNcfoRoxTgpo6BNlzv9Y3prDXiwEjxvosK+4WWIVTTEh33nNvQ5iQhPqBNgURmjQx9EDXFIRdZzA8OykPNLIqFdzmxGZWWxFbW/n6nEl/96b6w7Gx0YgzTSLs+6WAQl8SMP9l22L6puitpjihRw9cWRJ9r6x1eLqgc5Sv7gDKOMXghbmS6hy+AtrxCPPJgq7Mguc5bPAqTZlYMy98dxpHVqtAnBso/9aLOzAXX6At/0QUIwMP693B11NTGniIMtBxnD/yWvGoxTXNmXcTvj13cTzSv9czaGSJ+MTRIugtgyouZADfs8v59NV9KoaEq8umy6WEhmtw5wkjzvC5KK4N2bsM1N+8lSIKxYWxWZFsdYBP8ep442Z/2T5R8y8c5cp7tQqqapDt8JPJ0OPq3sn30BO3X8MgvmoB39j4Cqok1y9VuouPH4RalRLMR7KrASdlFengjt0vWBUoNaEuxRdJR2eOM6SpZh6YGqLdQH1MLaBOzDTH2tTLyTXCOSJpve6ZHOPbjS2BF34a1Kj52NTFtiYTw== jacob.weinstock@netapp.com"
	v.LogFile = "/tmp/cake.log"
	configYaml, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	script := fmt.Sprintf(`#!/bin/bash

socat -u TCP-LISTEN:%s,fork CREATE:%s,group=root,perm=0755 & disown
socat TCP-LISTEN:%s,reuseaddr,fork EXEC:"/bin/bash",pty,setsid,setpgid,stderr,ctty & disown
wget -O /usr/local/bin/clusterctl https://github.com/kubernetes-sigs/cluster-api/releases/download/v0.3.0/clusterctl-$(uname | tr '[:upper:]' '[:lower:]')-amd64
chmod +x /usr/local/bin/clusterctl
wget -O /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.7.0/kind-$(uname)-amd64
chmod +x /usr/local/bin/kind
curl https://get.docker.com/ | bash
cat <<EOF> %s
%s
EOF

`, uploadPort, remoteExecutable, commandPort, remoteConfig, configYaml)
	bootstrapVM, err := v.Session.CloneTemplate(ovas[v.OVA.NodeTemplate], "bootstrap-vm", script, publicKey, "jacob")
	if err != nil {
		return err
	}
	v.Resources["bootstrapVM"] = bootstrapVM

	return err
}
