package vsphere

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Prepare the environment for bootstrapping
func (v *MgmtBootstrapCAPV) Prepare() error {

	desiredFolders := []string{
		fmt.Sprintf("%s/%s", baseFolder, templatesFolder),
		fmt.Sprintf("%s/%s", baseFolder, workloadsFolder),
		fmt.Sprintf("%s/%s", baseFolder, mgmtFolder),
		fmt.Sprintf("%s/%s", baseFolder, bootstrapFolder),
	}

	for _, f := range desiredFolders {
		tempFolder, err := v.Session.CreateVMFolders(f)
		if err != nil {
			return err
		}
		v.TrackedResources.addTrackedFolder(tempFolder)
	}

	if v.Folder != "" {
		fromConfig, err := v.Session.CreateVMFolders(v.Folder)
		if err != nil {
			return err
		}
		v.Folder = fromConfig[filepath.Base(v.Folder)].InventoryPath

	} else {
		v.Folder = v.TrackedResources.Folders[mgmtFolder].InventoryPath
	}

	v.Session.Folder = v.TrackedResources.Folders[templatesFolder]
	ovas, err := v.Session.DeployOVATemplates(v.OVA.BootstrapTemplate, v.OVA.NodeTemplate, v.OVA.LoadbalancerTemplate)
	if err != nil {
		return err
	}
	v.TrackedResources.addTrackedVM(ovas)
	v.Session.Folder = v.TrackedResources.Folders[bootstrapFolder]

	configYaml, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	// TODO put in cloudinit engine specific bootstrap VM prereqs.
	script := fmt.Sprintf(`#!/bin/bash

	# install socat
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
	bootstrapVM, err := v.Session.CloneTemplate(ovas[v.OVA.BootstrapTemplate], bootstrapVMName, script, v.SSH.AuthorizedKey, v.SSH.Username)
	if err != nil {
		return err
	}
	v.TrackedResources.VMs[bootstrapVMName] = bootstrapVM

	return err
}
