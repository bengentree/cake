package vsphere

import (
	"github.com/netapp/cake/pkg/config/cluster"
	vsphereConfig "github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/provider"
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

// TrackedResources are vmware objects created during the bootstrap process
type TrackedResources struct {
	Folders map[string]*object.Folder
	VMs     map[string]*object.VirtualMachine
}

// GeneratedKey is the key pair generated for the run
type GeneratedKey struct {
	PrivateKey string
	PublicKey  string
}

// MgmtBootstrap spec for CAPV
type MgmtBootstrap struct {
	provider.Spec                 `yaml:",inline" json:",inline" mapstructure:",squash"`
	vsphereConfig.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
	Session                       *Session         `yaml:"-" json:"-" mapstructure:"-"`
	TrackedResources              TrackedResources `yaml:"-" json:"-" mapstructure:"-"`
	Prerequisites                 string           `yaml:"-" json:"-" mapstructure:"-"`
}

// MgmtBootstrapCAPV is the spec for bootstrapping a CAPV management cluster
type MgmtBootstrapCAPV struct {
	MgmtBootstrap      `yaml:",inline" json:",inline" mapstructure:",squash"`
	cluster.CAPIConfig `yaml:",inline" json:",inline" mapstructure:",squash"`
}

// MgmtBootstrapRKE is the spec for bootstrapping a RKE management cluster
type MgmtBootstrapRKE struct {
	MgmtBootstrap `yaml:",inline" json:",inline" mapstructure:",squash"`
	BootstrapIP   string            `yaml:"BootstrapIP" json:"bootstrapIP"`
	Nodes         map[string]string `yaml:"Nodes" json:"nodes"`
	RKEConfigPath string            `yaml:"RKEConfigPath"`
	Hostname      string            `yaml:"Hostname" json:"hostname"`
	GeneratedKey  GeneratedKey      `yaml:"-" json:"-" mapstructure:"-"`
}

// Client setups connection to remote vCenter
func (v *MgmtBootstrap) Client() error {
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
	v.TrackedResources.Folders = make(map[string]*object.Folder)
	v.TrackedResources.VMs = make(map[string]*object.VirtualMachine)

	return nil
}

// Progress monitors the of the management cluster bootstrapping process
func (v *MgmtBootstrap) Progress() error {
	var err error

	return err
}

// Finalize handles saving deliverables and cleaning up the bootstrap VM
func (v *MgmtBootstrap) Finalize() error {
	var err error

	return err
}

// Events returns the channel of progress messages
func (v *MgmtBootstrap) Events() chan progress.Event {
	return v.EventStream
}

func (tr *TrackedResources) addTrackedFolder(resources map[string]*object.Folder) {
	for key, value := range resources {
		tr.Folders[key] = value
	}
}

func (tr *TrackedResources) addTrackedVM(resources map[string]*object.VirtualMachine) {
	for key, value := range resources {
		tr.VMs[key] = value
	}
}
