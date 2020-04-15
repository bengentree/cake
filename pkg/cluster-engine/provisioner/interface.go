package provisioner

// Cluster interface for deploying K8s clusters
type Cluster interface {
	CreateBootstrap() error
	InstallControlPlane() error
	CreatePermanent() error
	PivotControlPlane() error
	InstallAddons() error
	RequiredCommands() []string
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
