package engine

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/netapp/cake/pkg/progress"

	"github.com/netapp/cake/pkg/config/cluster"
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
	Events() progress.Events
	// Spec returns the spec for the interface
	Spec() MgmtCluster
}

// MgmtCluster spec for the Engine
type MgmtCluster struct {
	LogFile                 string         `yaml:"LogFile" json:"logfile"`
	LogDir                  string         `yaml:"LogDir" json:"logdir"`
	SSH                     cluster.SSH    `yaml:"SSH" json:"ssh"`
	Addons                  cluster.Addons `yaml:"Addons,omitempty" json:"addons,omitempty"`
	cluster.K8sConfig       `yaml:",inline" json:",inline" mapstructure:",squash"`
	EventStream             progress.Events `yaml:"-" json:"-" mapstructure:"-"`
	ProgressEndpointEnabled bool            `yaml:"-" json:"-" mapstructure:"-"`
	FileDeliverables        []string
}

// Run provider bootstrap process
func Run(c Cluster) error {
	spec := c.Spec()
	if spec.ProgressEndpointEnabled {
		defer progress.ServeDuration()
		defer progress.UpdateProgressComplete(true)
		go progress.Serve(
			spec.LogFile,
			getLocalIP(),
			"8081",
			c.Events(),
			spec.FileDeliverables,
		)
	}
	// TODO poll for the endpoints to be up or something similar before starting to send messages
	// progress.Serve needs just a couple seconds to subscribe to the events msgs
	time.Sleep(3 * time.Second)
	exist := c.RequiredCommands()
	if len(exist) > 0 {
		errMsg := fmt.Sprintf("the following commands were not found in $PATH: [%v]", strings.Join(exist, ", "))
		return fmt.Errorf(errMsg)
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
	if spec.ProgressEndpointEnabled {
		progress.UpdateProgressCompletedSuccessfully(true)
	}

	return nil
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "<worker_node_ip>"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
