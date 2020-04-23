package engines

import (
	"github.com/netapp/cake/pkg/config/events"
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
	// SpecConvert changes the unmarshalled map into a provider specifc struct
	SpecConvert() error
	// Events are messages from the implementation
	Events() chan events.Event
}

// Spec for the Engine
type MgmtCluster struct {
	//config.Spec `yaml:",inline" json:",inline" mapstructure:",squash"`
	EventStream chan events.Event
	LogFile      string       `yaml:"LogFile" json:"logfile"`
	ClusterConfig      `yaml:",inline" json:",inline" mapstructure:",squash"`
	SSH          SSH          `yaml:"SSH" json:"ssh"`
	Addons       Addons       `yaml:"Addons,omitempty" json:"addons,omitempty"`
}

// Cluster specifies the details about the management cluster
type ClusterConfig struct {
	ClusterName           string `yaml:"ClusterName" json:"clustername"`
	ControlPlaneCount     int    `yaml:"ControlPlaneCount" json:"controlplanecount"`
	WorkerCount           int    `yaml:"WorkerCount" json:"workercount"`
	KubernetesVersion     string `yaml:"KubernetesVersion" json:"kubernetesversion"`
	KubernetesPodCidr     string `yaml:"KubernetesPodCidr" json:"kubernetespodcidr"`
	KubernetesServiceCidr string `yaml:"KubernetesServiceCidr" json:"kubernetesservicecidr"`
	Kubeconfig            string `yaml:"Kubeconfig" json:"kubeconfig"`
	Namespace             string `yaml:"Namespace" json:"namespace"`
}
// SSH holds ssh info
type SSH struct {
	Username      string `yaml:"Username" json:"username"`
	AuthorizedKey string `yaml:"AuthorizedKey" json:"authorizedkey"`
}

// Addons holds optional configuration values
type Addons struct {
	Observability ObservabilitySpec `yaml:"Observability,omitempty" json:"observability,omitempty"`
	Solidfire     Solidfire         `yaml:"Solidfire,omitempty" json:"solidfire,omitempty"`
}

// ObservabilitySpec holds values for the observability archive file
type ObservabilitySpec struct {
	Enable          bool   `yaml:"Enable" json:"enable"`
	ArchiveLocation string `yaml:"ArchiveLocation" json:"archivelocation"`
}

// Solidfire Addon info
type Solidfire struct {
	Enable   bool   `yaml:"Enable"`
	MVIP     string `yaml:"MVIP"`
	SVIP     string `yaml:"SVIP"`
	User     string `yaml:"User"`
	Password string `yaml:"Password"`
}

/*
// Run provider bootstrap process
func (s *MgmtCluster) Run() error {
	if s.LogFile != "" {
		cmds.FileLogLocation = s.LogFile
		os.Truncate(s.LogFile, 0)
	}

	err := s.SpecConvert()
	if err != nil {
		return err
	}
	//fmt.Printf("inside RUN: %+v\n", s)

	exist := s.Engine.(Cluster).RequiredCommands()
	if len(exist) > 0 {
		return fmt.Errorf("the following commands were not found in $PATH: [%v]", strings.Join(exist, ", "))
	}

	err = s.Engine.(Cluster).CreateBootstrap()
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).InstallControlPlane()
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).CreatePermanent()
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).PivotControlPlane()
	if err != nil {
		return err
	}

	err = s.Engine.(Cluster).InstallAddons()
	if err != nil {
		return err
	}

	return nil
}

 */
