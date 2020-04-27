package vsphere

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Prepare the environment for bootstrapping
func (v *MgmtBootstrapCAPV) Prepare() error {

	tFolderName := fmt.Sprintf("%s/%s", baseFolder, templatesFolder)
	tFolder, err := v.Session.CreateVMFolders(tFolderName)
	if err != nil {
		return err
	}
	v.trackedResources.addTrackedFolder(tFolder)

	wFolderName := fmt.Sprintf("%s/%s", baseFolder, workloadsFolder)
	wFolder, err := v.Session.CreateVMFolders(wFolderName)
	if err != nil {
		return err
	}
	v.trackedResources.addTrackedFolder(wFolder)

	mFolderName := fmt.Sprintf("%s/%s", baseFolder, mgmtFolder)
	mFolder, err := v.Session.CreateVMFolders(mFolderName)
	if err != nil {
		return err
	}
	v.trackedResources.addTrackedFolder(mFolder)

	bootFolderName := fmt.Sprintf("%s/%s", baseFolder, bootstrapFolder)
	bootFolder, err := v.Session.CreateVMFolders(bootFolderName)
	if err != nil {
		return err
	}
	v.trackedResources.addTrackedFolder(bootFolder)

	if v.Folder != "" {
		if strings.HasPrefix(v.Folder, "/") {
			_, err := v.Session.CreateVMFolders(v.Folder)
			if err != nil {
				return err
			}
		} else {
			_, err := v.Session.CreateVMFolders(baseFolder + "/" + v.Folder)
			if err != nil {
				return err
			}
			v.Folder = baseFolder + "/" + v.Folder
		}
	} else {
		v.Folder = mFolder[mgmtFolder].InventoryPath
	}

	v.Session.Folder = tFolder[templatesFolder]
	ovas, err := v.Session.DeployOVATemplates(v.OVA.BootstrapTemplate, v.OVA.NodeTemplate, v.OVA.LoadbalancerTemplate)
	if err != nil {
		return err
	}
	v.Session.Folder = bootFolder[bootstrapFolder]
	v.trackedResources.addTrackedVM(ovas)

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
	v.trackedResources.vms[bootstrapVMName] = bootstrapVM

	return err
}
