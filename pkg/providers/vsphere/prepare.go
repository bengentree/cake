package vsphere

import (
	"fmt"

	"github.com/vmware/govmomi/object"
	"gopkg.in/yaml.v3"
)

// Prepare the environment for bootstrapping
func (v *MgmtBootstrap) Prepare() error {
	tFolder, err := v.Session.CreateVMFolder(baseFolder + "/" + templatesFolder)
	if err != nil {
		return err
	}
	v.Resources[baseFolder] = tFolder[baseFolder]
	v.Resources[templatesFolder] = tFolder[templatesFolder]

	wFolder, err := v.Session.CreateVMFolder(baseFolder + "/" + workloadsFolder)
	if err != nil {
		return err
	}
	v.Resources[workloadsFolder] = wFolder[workloadsFolder]

	mFolder, err := v.Session.CreateVMFolder(baseFolder + "/" + mgmtFolder)
	if err != nil {
		return err
	}
	v.Resources[mgmtFolder] = mFolder[mgmtFolder]

	bootFolder, err := v.Session.CreateVMFolder(baseFolder + "/" + bootstrapFolder)
	if err != nil {
		return err
	}
	v.Resources[bootstrapFolder] = bootFolder[bootstrapFolder]

	if v.Folder != "" {
		FolderFromConfig, err := v.Session.CreateVMFolder(baseFolder + "/" + v.Folder)
		if err != nil {
			return err
		}
		v.Resources["FolderFromConfig"] = FolderFromConfig[v.Folder]
		v.Folder = v.Resources["FolderFromConfig"].(*object.Folder).InventoryPath
	} else {
		v.Folder = v.Resources[mgmtFolder].(*object.Folder).InventoryPath
	}

	v.Session.Folder = bootFolder[bootstrapFolder]
	ovas, err := v.Session.DeployOVATemplates(v.OVA.NodeTemplate, v.OVA.LoadbalancerTemplate)

	v.Resources[v.OVA.NodeTemplate] = ovas[v.OVA.NodeTemplate]
	v.Resources[v.OVA.LoadbalancerTemplate] = ovas[v.OVA.LoadbalancerTemplate]

	configYaml, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	// TODO put in cloudinit engine specific bootstrap VM prereqs.
	script := fmt.Sprintf(`#!/bin/bash

%s
%s
wget -O /usr/local/bin/clusterctl https://github.com/kubernetes-sigs/cluster-api/releases/download/v0.3.0/clusterctl-$(uname | tr '[:upper:]' '[:lower:]')-amd64
chmod +x /usr/local/bin/clusterctl
wget -O /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.7.0/kind-$(uname)-amd64
chmod +x /usr/local/bin/kind
curl https://get.docker.com/ | bash
cat <<EOF> %s
%s
EOF

`, fmt.Sprintf(uploadFileCmd, uploadPort, remoteExecutable), fmt.Sprintf(runRemoteCmd, commandPort), remoteConfig, configYaml)
	bootstrapVM, err := v.Session.CloneTemplate(ovas[v.OVA.NodeTemplate], bootstrapVMName, script, v.SSH.AuthorizedKey, v.SSH.Username)
	if err != nil {
		return err
	}
	v.Resources[bootstrapVMName] = bootstrapVM

	return err
}
