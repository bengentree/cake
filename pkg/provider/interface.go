package provider

import (
	"github.com/netapp/cake/pkg/config/cluster"
	"github.com/netapp/cake/pkg/config/types"
	"github.com/netapp/cake/pkg/progress"
)

// Bootstrapper is the interface for creating infrastructure to run a cake engine against
type Bootstrapper interface {
	// Client setups up any client connections to remote provider
	Client() error
	// Prepare setups up any needed infrastructure
	Prepare() error
	// Provision runs the management cluster creation steps
	Provision() error
	// Progress watches the cluster creation for progress. One node will make the following HTTP endpoints available. The progress method will read all progress events from /progress
	// /progress - all events messages, overall complete status and overall success status
	// /log - the stdout of all commands run
	// /deliverable - is the URI discovery endpoint for all files that were created as part of the deploy
	// /deliverable/<file_name> - engines will implement any number of endpoints here where the file_name is an engine specific file created during the deployment process
	Progress() error
	// Finalize saves in the .cake/<cluster-name>/ directory /log and all /deliverable/<file_name> files and removes any created bootstrap infrastructure
	Finalize() error
	// Events are status messages from the implementation
	Events() progress.Events
}

// Spec for the Provider
type Spec struct {
	cluster.K8sConfig `yaml:",inline" json:",inline" mapstructure:",squash"`
	EventStream       progress.Events  `yaml:"-" json:"-" mapstructure:"-"`
	EngineType        types.EngineType `yaml:"EngineType" json:"enginetype"`
	LogFile           string           `yaml:"LogFile" json:"logfile"`
	LogDir            string           `yaml:"LogDir" json:"logdir"`
	SSH               cluster.SSH      `yaml:"SSH" json:"ssh"`
	BootstrapperIP    string           `yaml:"-" json:"-" mapstructure:"-"`
}

// Run provider bootstrap process
func Run(b Bootstrapper) error {
	log := b.Events()
	log.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Connecting to provider",
		Level: "info",
	})
	err := b.Client()
	if err != nil {
		return err
	}
	log.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Preparing environment",
		Level: "info",
	})
	defer finalize(b)
	err = b.Prepare()
	if err != nil {
		return err
	}
	log.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Provisioning cluster",
		Level: "info",
	})
	err = b.Provision()
	if err != nil {
		return err
	}
	log.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Provision Progress",
		Level: "info",
	})
	err = b.Progress()
	if err != nil {
		return err
	}

	return nil
}

func finalize(b Bootstrapper) {
	log := b.Events()
	log.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Finalizing",
		Level: "info",
	})
	b.Finalize()
}
