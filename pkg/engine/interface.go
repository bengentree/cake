package engine

import (
	"fmt"
	"strings"

	"github.com/netapp/cake/pkg/config/cluster"
	"github.com/netapp/cake/pkg/progress"
)

// Cluster interface for deploying K8s clusters
type Cluster interface {
	// CreateBootstrap sets up the boostrap cluster
	CreateBootstrap() error
	// InstallControlPlane puts the control plane on the boostrap cluster
	InstallControlPlane() error
	// CreatePermanent provisions the permanent management cluster
	CreatePermanent() error
	// PivotControlPlane moves the control plane from bootstrap to permanent management cluster
	PivotControlPlane() error
	// InstallAddons will install any addons into the permanent management cluster
	InstallAddons() error
	// RequiredCommands returns the command like binaries need to run the engine
	RequiredCommands() []string
	// Events are messages from the implementation
	Events() chan progress.Event
}

// MgmtCluster spec for the Engine
type MgmtCluster struct {
	LogFile           string         `yaml:"LogFile" json:"logfile"`
	SSH               cluster.SSH    `yaml:"SSH" json:"ssh"`
	Addons            cluster.Addons `yaml:"Addons,omitempty" json:"addons,omitempty"`
	cluster.K8sConfig `yaml:",inline" json:",inline" mapstructure:",squash"`
	EventStream       chan progress.Event `yaml:"-" json:"-" mapstructure:"-"`
}

// Run provider bootstrap process
func Run(c Cluster) error {
	exist := c.RequiredCommands()
	if len(exist) > 0 {
		return fmt.Errorf("the following commands were not found in $PATH: [%v]", strings.Join(exist, ", "))
	}

	err := c.CreateBootstrap()
	if err != nil {
		return err
	}

	err = c.InstallControlPlane()
	if err != nil {
		return err
	}

	err = c.CreatePermanent()
	if err != nil {
		return err
	}

	err = c.PivotControlPlane()
	if err != nil {
		return err
	}

	err = c.InstallAddons()
	if err != nil {
		return err
	}

	return nil
}
