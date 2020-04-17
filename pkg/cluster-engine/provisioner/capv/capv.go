package capv

import (
	"os"

	"github.com/netapp/cake/pkg/cluster-engine/provisioner"
	"github.com/netapp/cake/pkg/cmds"
)

// NewMgmtCluster creates a new cluster interface with a full config from the client
func NewMgmtCluster(clusterConfig MgmtCluster) provisioner.Cluster {
	mc := new(MgmtCluster)
	mc = &clusterConfig
	mc.events = make(chan interface{})
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}

	return mc
}

// MgmtCluster spec for CAPV
type MgmtCluster struct {
	provisioner.MgmtCluster `yaml:",inline" mapstructure:",squash"`
	Vsphere                 `yaml:",inline" mapstructure:",squash"`
	Addons                  Addons `yaml:"Addons"`
	events                  chan interface{}
}

type Vsphere struct {
	Datacenter        string `yaml:"Datacenter"`
	Datastore         string `yaml:"Datastore"`
	Folder            string `yaml:"Folder"`
	ManagementNetwork string `yaml:"ManagementNetwork"`
	WorkloadNetwork   string `yaml:"WorkloadNetwork"`
	StorageNetwork    string `yaml:"StorageNetwork"`
	ResourcePool      string `yaml:"ResourcePool"`
	VcenterServer     string `yaml:"VcenterServer"`
	VsphereUsername   string `yaml:"VsphereUsername"`
	VspherePassword   string `yaml:"VspherePassword"`
}

type Addons struct {
	Solidfire     Solidfire     `yaml:"Solidfire"`
	Observability Observability `yaml:"Observability"`
}

type Solidfire struct {
	Enable   bool   `yaml:"Enable"`
	MVIP     string `yaml:"MVIP"`
	SVIP     string `yaml:"SVIP"`
	User     string `yaml:"User"`
	Password string `yaml:"Password"`
}

type Observability struct {
	Enable          bool   `yaml:"Enabled"`
	ArchiveLocation string `yaml:"ArchiveLocation"`
}

// Event spec
type Event struct {
	EventType string
	Event     string
}
