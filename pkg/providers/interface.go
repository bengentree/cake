package providers

import (
	"github.com/netapp/cake/pkg/config/cluster"
	"github.com/netapp/cake/pkg/config/events"
	"github.com/netapp/cake/pkg/config/types"
)

// Bootstrap is the interface for creating a bootstrap vm and running cluster provisioning
type Bootstrap interface {
	// Client setups up any client connections to remote providers
	Client() error
	// Prepare setups up any needed infrastructure
	Prepare() error
	// Provision runs the management cluster creation steps
	Provision() error
	// Progress watches the cluster creation for progress
	Progress() error
	// Finalize saves any deliverables and removes any created bootstrap infrastructure
	Finalize() error
	// Events are status messages from the implementation
	Events() chan events.Event
}

// Spec for the Provider
type Spec struct {
	cluster.ClusterConfig `yaml:",inline" json:",inline" mapstructure:",squash"`
	EventStream           chan events.Event `yaml:"-" json:"-" mapstructure:"-"`
	EngineType            types.EngineType  `yaml:"EngineType" json:"enginetype"`
	LogFile               string            `yaml:"LogFile" json:"logfile"`
}

// Run provider bootstrap process
func Run(b Bootstrap) error {
	err := b.Client()
	if err != nil {
		return err
	}
	err = b.Prepare()
	if err != nil {
		return err
	}
	err = b.Provision()
	if err != nil {
		return err
	}
	err = b.Progress()
	if err != nil {
		return err
	}
	err = b.Finalize()
	if err != nil {
		return err
	}
	return nil
}
