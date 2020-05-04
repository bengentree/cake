package vsphere

import (
	"fmt"

	"github.com/netapp/cake/pkg/utils/ssh"
	"github.com/vmware/govmomi/object"
	"gopkg.in/yaml.v3"
)

// Prepare bootstrap VM for rke deployment
func (v *MgmtBootstrapRKE) Prepare() error {
	err := v.createFolders()
	if err != nil {
		return err
	}
	// generate key pair
	privateKey, publicKey, err := ssh.GenerateRSAKeyPair()
	if err != nil {
		return err
	}
	v.SSH.AuthorizedKeys = append(v.SSH.AuthorizedKeys, publicKey)
	v.GeneratedKey.PrivateKey = privateKey
	v.GeneratedKey.PublicKey = publicKey
	configYAML, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	// TODO make prereqs less hacky than this
	// v.Prerequisites = rkePrereqs
	return v.prepareRKE(configYAML)
}

// Prepare the environment for bootstrapping
func (v *MgmtBootstrapRKE) prepareRKE(configYAML []byte) error {
	mFolder := v.Session.Folder
	v.Session.Folder = v.TrackedResources.Folders[templatesFolder]
	ovas, err := v.Session.DeployOVATemplates(v.OVA.BootstrapTemplate, v.OVA.NodeTemplate, v.OVA.LoadbalancerTemplate)
	if err != nil {
		return err
	}
	// TODO save ova templates to TrackedResources?
	v.Session.Folder = mFolder

	baseScript := fmt.Sprintf(`#!/bin/bash

# engine specifc prereqs
%s
`, v.Prerequisites)

	script := fmt.Sprintf(`#!/bin/bash

# install socat, needed for TCP listeners
wget -O /usr/local/bin/socat https://github.com/andrew-d/static-binaries/raw/master/binaries/linux/x86_64/socat
chmod +x /usr/local/bin/socat

# TCP listener for uploading cake binary
%s

# TCP listener for uploading cake binary
%s

# TCP listener for running cake binary
%s

# engine specific prereqs to run
%s

# write the generated private key to disk
%s

`, fmt.Sprintf(
		uploadFileCmd,
		uploadPort,
		remoteExecutable,
	),
		fmt.Sprintf(
			uploadFileCmd,
			uploadConfigPort,
			remoteConfigRoot,
		),
		fmt.Sprintf(
			runRemoteCmd,
			commandPort,
		),
		v.Prerequisites,
		fmt.Sprintf(privateKeyToDisk, v.GeneratedKey.PrivateKey),
	)

	nodes := []cloneSpec{}
	bootstrapNode := cloneSpec{
		template:   ovas[v.OVA.NodeTemplate],
		name:       fmt.Sprintf("%s1", rkeControlNodePrefix),
		bootScript: script,
		publicKey:  v.SSH.AuthorizedKeys,
		osUser:     v.SSH.Username,
	}
	nodes = append(nodes, bootstrapNode)
	for vm := 2; vm <= v.ControlPlaneCount; vm++ {
		vmName := fmt.Sprintf("%s%v", rkeControlNodePrefix, vm)
		spec := cloneSpec{
			template:   ovas[v.OVA.NodeTemplate],
			name:       vmName,
			bootScript: baseScript,
			publicKey:  v.SSH.AuthorizedKeys,
			osUser:     v.SSH.Username,
		}
		nodes = append(nodes, spec)
	}
	for vm := 1; vm <= v.WorkerCount; vm++ {
		vmName := fmt.Sprintf("%s%v", rkeWorkerNodePrefix, vm)
		spec := cloneSpec{
			template:   ovas[v.OVA.NodeTemplate],
			name:       vmName,
			bootScript: baseScript,
			publicKey:  v.SSH.AuthorizedKeys,
			osUser:     v.SSH.Username,
		}
		nodes = append(nodes, spec)
	}
	vmsCreated, err := v.Session.CloneTemplates(nodes...)
	for name, vm := range vmsCreated {
		v.TrackedResources.addTrackedVM(map[string]*object.VirtualMachine{name: vm})
	}

	return err
}
