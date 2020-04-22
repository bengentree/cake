package providers

import (
	"github.com/netapp/cake/pkg/config"
)

// Bootstrap is the interface for creating a bootstrap vm and running cluster provisioning
type Bootstrap interface {
	// Prepare setups up any needed infrastructure
	Prepare(*Spec) error
	// Provision runs the management cluster creation steps
	Provision(*Spec) error
	// Progress watches the cluster creation for progress
	Progress(*Spec) error
	// Finalize saves any deliverables and removes any created bootstrap infrastructure
	Finalize(*Spec) error
	// Events are status messages from the implementation
	Events(*Spec) chan interface{}
}

// Spec for the Provider
type Spec struct {
	config.Spec `yaml:",inline" json:",inline" mapstructure:",squash"`
}

// Run provider bootstrap process
func (s *Spec) Run() error {
	return nil
}