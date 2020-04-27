package vsphere

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
	BootstrapTemplate    string `yaml:"BootstrapTemplate" json:"bootstraptemplate"`
	NodeTemplate         string `yaml:"NodeTemplate" json:"nodetemplate,omitempty"`
	LoadbalancerTemplate string `yaml:"LoadbalancerTemplate" json:"loadbalancertemplate,omitempty"`
}
