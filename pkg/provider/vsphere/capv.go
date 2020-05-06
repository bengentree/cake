package vsphere

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
	log "github.com/sirupsen/logrus"
)

// Prepare bootstrap VM for capv deployment
func (v *MgmtBootstrapCAPV) Prepare() error {
	err := v.createFolders()
	if err != nil {
		return err
	}
	configYAML, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	prereqs := fmt.Sprintf(`wget -O /usr/local/bin/clusterctl https://github.com/kubernetes-sigs/cluster-api/releases/download/%s/clusterctl-$(uname | tr '[:upper:]' '[:lower:]')-amd64
	chmod +x /usr/local/bin/clusterctl
	wget -O /usr/local/bin/kind https://kind.sigs.k8s.io/dl/%s/kind-$(uname)-amd64
	chmod +x /usr/local/bin/kind
	curl https://get.docker.com/ | bash`, capvClusterctlVersion, capvKindVersion)
	v.Prerequisites = prereqs

	return v.MgmtBootstrap.prepare(configYAML)
}

func (v *MgmtBootstrap) createFolders() error {
	desiredFolders := []string{
		fmt.Sprintf("%s/%s", baseFolder, templatesFolder),
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
		v.TrackedResources.addTrackedFolder(fromConfig)
		v.Folder = fromConfig[filepath.Base(v.Folder)].InventoryPath
		v.Session.Folder = fromConfig[filepath.Base(v.Folder)]
	} else {
		tempFolder, err := v.Session.CreateVMFolders(fmt.Sprintf("%s/%s", baseFolder, mgmtFolder))
		if err != nil {
			return err
		}
		v.TrackedResources.addTrackedFolder(tempFolder)
		v.Folder = tempFolder[mgmtFolder].InventoryPath
		v.Session.Folder = tempFolder[mgmtFolder]
	}
	return nil
}

// Prepare the environment for bootstrapping
func (v *MgmtBootstrap) prepare(configYAML []byte) error {
	v.Session.Folder = v.TrackedResources.Folders[templatesFolder]
	ovas, err := v.Session.DeployOVATemplates(v.OVA.BootstrapTemplate, v.OVA.NodeTemplate, v.OVA.LoadbalancerTemplate)
	if err != nil {
		return err
	}
	v.TrackedResources.addTrackedVM(ovas)
	v.Session.Folder = v.TrackedResources.Folders[bootstrapFolder]

	script := fmt.Sprintf(`#!/bin/bash

# install socat, needed for TCP listeners
wget -O /usr/local/bin/socat https://github.com/andrew-d/static-binaries/raw/master/binaries/linux/x86_64/socat
chmod +x /usr/local/bin/socat

# TCP listener for uploading cake binary
%s

# TCP listener for running cake binary
%s

# engine specific prereqs to run
%s

# write cake config file to disk
cat <<EOF> %s
%s
EOF

`, fmt.Sprintf(uploadFileCmd, uploadPort, remoteExecutable), fmt.Sprintf(runRemoteCmd, commandPort), v.Prerequisites, remoteConfig, configYAML)
	bootstrapVM, err := v.Session.CloneTemplate(ovas[v.OVA.BootstrapTemplate], bootstrapVMName, script, v.SSH.AuthorizedKeys, v.SSH.Username)
	if err != nil {
		return err
	}
	v.TrackedResources.VMs[bootstrapVMName] = bootstrapVM

	return err
}

// Provision calls the process to create the management cluster for CAPV
func (v *MgmtBootstrapCAPV) Provision() error {
	bootstrapVMIP, err := GetVMIP(v.TrackedResources.VMs[bootstrapVMName])
	if err != nil {
		return err
	}
	log.Infof("bootstrap VM IP: %v", bootstrapVMIP)

	configYAML, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	err = uploadFilesToBootstrap(bootstrapVMIP, string(configYAML))
	if err != nil {
		return err
	}

	cakeCmd := fmt.Sprintf(runLocalCakeCmd, remoteExecutable, string(v.EngineType))
	tcp, err := newTCPConn(bootstrapVMIP + ":" + commandPort)
	if err != nil {
		return err
	}
	tcp.runAsyncCommand(cakeCmd)

	return err
}