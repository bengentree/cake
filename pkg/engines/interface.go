package engines

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
	Events() chan interface{}
}

// MgmtCluster spec
type MgmtCluster struct {
	K8s                      `yaml:",inline" mapstructure:",squash"`
	LoadBalancerTemplate     string `yaml:"LoadBalancerTemplate"`
	NodeTemplate             string `yaml:"NodeTemplate"`
	SSHAuthorizedKey         string `yaml:"SshAuthorizedKey"`
	ControlPlaneMachineCount string `yaml:"ControlPlaneMachineCount"`
	WorkerMachineCount       string `yaml:"WorkerMachineCount"`
	LogFile                  string `yaml:"LogFile"`
}

// K8s spec
type K8s struct {
	ClusterName           string `yaml:"ClusterName"`
	CapiSpec              string `yaml:"CapiSpec"`
	KubernetesVersion     string `yaml:"KubernetesVersion"`
	Namespace             string `yaml:"Namespace"`
	Kubeconfig            string `yaml:"Kubeconfig"`
	KubernetesPodCidr     string `yaml:"KubernetesPodCidr"`
	KubernetesServiceCidr string `yaml:"KubernetesServiceCidr"`
}
