package config

// ProviderType for available providers
type ProviderType string

// EngineType for available engines
type EngineType string

// Supported Provider and Engine Types
const (
	VsphereProvider = ProviderType("VSPHERE")
	KVMProvider     = ProviderType("KVM")
	EngineRKE       = EngineType("RKE")
	EngineCAPI      = EngineType("CAPI")
)

// Spec holds information needed to provision a K8s management cluster
type Spec struct {
	ProviderType ProviderType `yaml:"ProviderType" json:"providertype"`
	Provider     interface{}  `yaml:"Provider" json:"provider"`
	Engine       interface{}  `yaml:"Engine" json:"engine"`
	EngineType   EngineType   `yaml:"EngineType" json:"enginetype"`
	SSH          SSH          `yaml:"SSH" json:"ssh"`
	Local        bool         `yaml:"Local" json:"local"`
	LogFile      string       `yaml:"LogFile" json:"logfile"`
	Addons       Addons       `yaml:"Addons,omitempty" json:"addons,omitempty"`
	Cluster      `yaml:",inline" json:",inline" mapstructure:",squash"`
	//EventStream  chan Event
}

// Cluster specifies the details about the management cluster
type Cluster struct {
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

// ProviderVsphere is vsphere specifc data
type ProviderVsphere struct {
	URL               string  `yaml:"URL" json:"url"`
	Username          string  `yaml:"Username" json:"username"`
	Password          string  `yaml:"Password" json:"password"`
	Datacenter        string  `yaml:"Datacenter" json:"datacenter"`
	ResourcePool      string  `yaml:"ResourcePool" json:"resourcepool"`
	Datastore         string  `yaml:"Datastore" json:"datastore"`
	ManagementNetwork string  `yaml:"ManagementNetwork" json:"managementnetwork"`
	StorageNetwork    string  `yaml:"StorageNetwork" json:"storagenetwork"`
	Folder            string  `yaml:"Folder" json:"folder"`
	OVA               OVASpec `yaml:"OVA" json:"ova"`
}

// OVASpec sets OVA information used for virtual machine templates
type OVASpec struct {
	NodeTemplate         string `yaml:"NodeTemplate" json:"nodetemplate"`
	LoadbalancerTemplate string `yaml:"LoadbalancerTemplate" json:"loadbalancertemplate"`
}

// Event spec
type Event struct {
	EventType string
	Event     string
}
