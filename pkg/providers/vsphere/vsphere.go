package vsphere

import (
	"fmt"

	"github.com/netapp/cake/pkg/config/cluster"
	vsphereConfig "github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/providers"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
)

// Session holds govmomi connection details
type Session struct {
	Conn         *govmomi.Client
	Datacenter   *object.Datacenter
	Datastore    *object.Datastore
	Folder       *object.Folder
	ResourcePool *object.ResourcePool
	Network      object.NetworkReference
}

type trackedResources struct {
	folders map[string]*object.Folder
	vms     map[string]*object.VirtualMachine
}

// MgmtBootstrap spec for CAPV
type MgmtBootstrap struct {
	providers.Spec                `yaml:",inline" json:",inline" mapstructure:",squash"`
	vsphereConfig.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
	Session                       *Session         `yaml:"-" json:"-" mapstructure:"-"`
	trackedResources              trackedResources `yaml:"-" json:"-" mapstructure:"-"`
}

type MgmtBootstrapCAPV struct {
	MgmtBootstrap      `yaml:",inline" json:",inline" mapstructure:",squash"`
	cluster.CAPIConfig `yaml:",inline" json:",inline" mapstructure:",squash"`
}

type MgmtBootstrapRKE struct {
	MgmtBootstrap `yaml:",inline" json:",inline" mapstructure:",squash"`
	BootstrapIP   string `yaml:"BootstrapIP" json:"BootstrapIP"`
}

func (v *MgmtBootstrapRKE) Client() error {
	fmt.Println(v.BootstrapIP)
	return nil
}

// Client setups connection to remote vCenter
func (v *MgmtBootstrapCAPV) Client() error {
	c, err := NewClient(v.URL, v.Username, v.Password)
	if err != nil {
		return err
	}
	c.Datacenter, err = c.GetDatacenter(v.Datacenter)
	if err != nil {
		return err
	}
	c.Network, err = c.GetNetwork(v.ManagementNetwork)
	if err != nil {
		return err
	}
	c.Datastore, err = c.GetDatastore(v.Datastore)
	if err != nil {
		return err
	}
	c.ResourcePool, err = c.GetResourcePool(v.ResourcePool)
	if err != nil {
		return err
	}
	v.Session = c
	v.trackedResources.folders = make(map[string]*object.Folder)
	v.trackedResources.vms = make(map[string]*object.VirtualMachine)

	return nil
}

func (tr *trackedResources) addTrackedFolder(resources map[string]*object.Folder) {
	for key, value := range resources {
		tr.folders[key] = value
	}
}

func (tr *trackedResources) addTrackedVM(resources map[string]*object.VirtualMachine) {
	for key, value := range resources {
		tr.vms[key] = value
	}
}
