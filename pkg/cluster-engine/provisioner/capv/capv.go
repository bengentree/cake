package capv

import (
	"os"

	"github.com/netapp/capv-bootstrap/pkg/cluster-engine/provisioner"
	"github.com/netapp/capv-bootstrap/pkg/cmds"
)

type requiredCmd string

const (
	kind       requiredCmd = "kind"
	clusterctl requiredCmd = "clusterctl"
	kubectl    requiredCmd = "kubectl"
	docker     requiredCmd = "docker"
	helm       requiredCmd = "helm"
	tridentctl requiredCmd = "tridentctl"
)

// RequiredCommands for capv provisioner
var RequiredCommands = cmds.ProvisionerCommands{Name: "required CAPV bootstrap commands"}

func init() {
	kd := cmds.NewCommandLine(nil, string(kind), nil, nil)
	c := cmds.NewCommandLine(nil, string(clusterctl), nil, nil)
	k := cmds.NewCommandLine(nil, string(kubectl), nil, nil)
	d := cmds.NewCommandLine(nil, string(docker), nil, nil)
	h := cmds.NewCommandLine(nil, string(helm), nil, nil)
	t := cmds.NewCommandLine(nil, string(tridentctl), nil, nil)

	RequiredCommands.AddCommand(kd.CommandName, kd)
	RequiredCommands.AddCommand(c.CommandName, c)
	RequiredCommands.AddCommand(k.CommandName, k)
	RequiredCommands.AddCommand(d.CommandName, d)
	RequiredCommands.AddCommand(h.CommandName, h)
	RequiredCommands.AddCommand(t.CommandName, t)

}

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
	Datacenter               string `yaml:"Datacenter"`
	Datastore                string `yaml:"Datastore"`
	Folder                   string `yaml:"Folder"`
	LoadBalancerTemplate     string `yaml:"LoadBalancerTemplate"`
	NodeTemplate             string `yaml:"NodeTemplate"`
	ManagementNetwork        string `yaml:"ManagementNetwork"`
	WorkloadNetwork          string `yaml:"WorkloadNetwork"`
	StorageNetwork           string `yaml:"StorageNetwork"`
	ResourcePool             string `yaml:"ResourcePool"`
	VcenterServer            string `yaml:"VcenterServer"`
	VsphereUsername          string `yaml:"VsphereUsername"`
	VspherePassword          string `yaml:"VspherePassword"`
	ClusterName              string `yaml:"ClusterName"`
	CapiSpec                 string `yaml:"CapiSpec"`
	KubernetesVersion        string `yaml:"KubernetesVersion"`
	Namespace                string `yaml:"Namespace"`
	Kubeconfig               string `yaml:"Kubeconfig"`
	SSHAuthorizedKey         string `yaml:"SshAuthorizedKey"`
	ControlPlaneMachineCount string `yaml:"ControlPlaneMachineCount"`
	WorkerMachineCount       string `yaml:"WorkerMachineCount"`
	LogFile                  string `yaml:"LogFile"`
	events                   chan interface{}
	Addons                   struct {
		Solidfire struct {
			Enable   bool   `yaml:"Enable"`
			MVIP     string `yaml:"MVIP"`
			SVIP     string `yaml:"SVIP"`
			User     string `yaml:"User"`
			Password string `yaml:"Password"`
		} `yaml:"Solidfire"`
		Observability struct {
			Enable          bool   `yaml:"Enabled"`
			ArchiveLocation string `yaml:"ArchiveLocation"`
		} `yaml:"Observability"`
	} `yaml:"Addons"`
	Configuration struct {
		Cluster struct {
			KubernetesPodCidr     string `yaml:"KubernetesPodCidr"`
			KubernetesServiceCidr string `yaml:"KubernetesServiceCidr"`
		} `yaml:"Cluster"`
	} `yaml:"Configuration"`
}

// Event spec
type Event struct {
	EventType string
	Event     string
}
