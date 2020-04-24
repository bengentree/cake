package vsphere

import (
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

// MgmtBootstrap spec for CAPV
type MgmtBootstrap struct {
	providers.Spec                `yaml:",inline" json:",inline" mapstructure:",squash"`
	vsphereConfig.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
	cluster.CAPIConfig            `yaml:",inline" json:",inline" mapstructure:",squash"`
	Session                       *Session               `yaml:"-" json:"-" mapstructure:"-"`
	Resources                     map[string]interface{} `yaml:"-" json:"-" mapstructure:"-"`
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
	v.Resources = make(map[string]interface{})

	return nil
}
