package engines

import (
	"fmt"
	"os"
	"strings"

	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config"
)

// Cluster interface for deploying K8s clusters
type Cluster interface {
	// CreateBootstrap sets up the boostrap cluster
	CreateBootstrap(*Spec) error
	// InstallControlPlane puts the control plane on the boostrap cluster
	InstallControlPlane(*Spec) error
	// CreatePermanent provisions the permanent management cluster
	CreatePermanent(*Spec) error
	// PivotControlPlane moves the control plane from bootstrap to permanent management cluster
	PivotControlPlane(*Spec) error
	// InstallAddons will install any addons into the permanent management cluster
	InstallAddons(*Spec) error
	// RequiredCommands returns the command like binaries need to run the engine
	RequiredCommands(*Spec) []string
	// SpecConvert changes the unmarshalled map into a provider specifc struct
	SpecConvert(*Spec) error
	// Events are messages from the implementation
	Events(*Spec) chan config.Event
}

// Spec for the Engine
type Spec struct {
	config.Spec `yaml:",inline" json:",inline" mapstructure:",squash"`
	EventStream chan config.Event
}

// Run provider bootstrap process
func (s *Spec) Run() error {
	if s.LogFile != "" {
		cmds.FileLogLocation = s.LogFile
		os.Truncate(s.LogFile, 0)
	}

	err := s.Engine.(Cluster).SpecConvert(s)
	if err != nil {
		return err
	}
	//fmt.Printf("inside RUN: %+v\n", s)

	exist := s.Engine.(Cluster).RequiredCommands(s)
	if len(exist) > 0 {
		return fmt.Errorf("the following commands were not found in $PATH: [%v]", strings.Join(exist, ", "))
	}

	err = s.Engine.(Cluster).CreateBootstrap(s)
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).InstallControlPlane(s)
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).CreatePermanent(s)
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).PivotControlPlane(s)
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).InstallAddons(s)
	if err != nil {
		return err
	}

	return nil
}
