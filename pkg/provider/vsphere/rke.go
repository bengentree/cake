package vsphere

import (
	"fmt"
	log "github.com/sirupsen/logrus"

	"github.com/netapp/cake/pkg/util/ssh"
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
	v.Prerequisites = fmt.Sprintf(rkePrereqs, v.SSH.Username)
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

	baseNodeScript := newNodeBaseScript(v.Prerequisites, string(v.EngineType)).ToString()
	bootstrapperScript := newNodeBaseScript(v.Prerequisites, string(v.EngineType))
	bootstrapperScript.MakeNodeBootstrapper()
	bootstrapperScript.AddLines(
		fmt.Sprintf(helmInstall, helmVersion),
		rkeBinaryInstall,
		fmt.Sprintf(privateKeyToDisk, v.GeneratedKey.PrivateKey),
	)

	nodes := []cloneSpec{}
	bootstrapNode := cloneSpec{
		template:   ovas[v.OVA.NodeTemplate],
		name:       fmt.Sprintf("%s1", rkeControlNodePrefix),
		bootScript: bootstrapperScript.ToString(),
		publicKey:  v.SSH.AuthorizedKeys,
		osUser:     v.SSH.Username,
	}
	nodes = append(nodes, bootstrapNode)
	for vm := 2; vm <= v.ControlPlaneCount; vm++ {
		vmName := fmt.Sprintf("%s%v", rkeControlNodePrefix, vm)
		spec := cloneSpec{
			template:   ovas[v.OVA.NodeTemplate],
			name:       vmName,
			bootScript: baseNodeScript,
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
			bootScript: baseNodeScript,
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

// Provision calls the process to create the management cluster for RKE
func (v *MgmtBootstrapRKE) Provision() error {
	var bootstrapVMIP string
	v.Nodes = map[string]string{}
	for name, vm := range v.TrackedResources.VMs {
		vmIP, err := GetVMIP(vm)
		if err != nil {
			return err
		}
		if name == fmt.Sprintf("%s1", rkeControlNodePrefix) {
			bootstrapVMIP = vmIP
			v.BootstrapIP = vmIP
		}
		v.Nodes[name] = vmIP
		// TODO switch log message to eents on the eventstream chan
		log.WithFields(log.Fields{
			"nodeName": name,
			"nodeIP":   vmIP,
		}).Info("vm IP received")
	}

	configYAML, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	err = uploadFilesToBootstrap(bootstrapVMIP, string(configYAML))
	if err != nil {
		return err
	}
	return nil
}
