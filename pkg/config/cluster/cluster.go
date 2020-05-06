package cluster

// K8sConfig specifies the details about the management cluster
type K8sConfig struct {
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
	Username       string   `yaml:"Username" json:"username"`
	AuthorizedKeys []string `yaml:"AuthorizedKeys" json:"authorizedkeys"`
	KeyPath        string   `yaml:"KeyPath,omitempty" json:"keyPath,omitempty"`
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
