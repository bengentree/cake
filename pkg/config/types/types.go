package types

// ProviderType for available providers
type ProviderType string

// EngineType for available engines
type EngineType string

// ConfigSpec holds information needed to register HCI with NKS
type ConfigSpec struct {
	Provider              string        `yaml:"Provider" json:"provider"`
	VCenterURL            string        `yaml:"VCenterURL" json:"vcenterurl"`
	VCenterUser           string        `yaml:"VCenterUser" json:"vcenteruser"`
	VCenterPassword       string        `yaml:"VCenterPassword" json:"vcenterpassword"`
	DatacenterID          string        `yaml:"DatacenterID" json:"datacenterid"`
	ResourcePoolID        string        `yaml:"ResourcePoolID" json:"resourcepoolid"`
	DatastoreID           string        `yaml:"DatastoreID" json:"datastoreid"`
	ManagementNetworkID   string        `yaml:"ManagementNetworkID" json:"managementnetworkid"`
	ManagementNetworkName string        `yaml:"ManagementNetworkName" json:"managementnetworkname"`
	WorkloadNetworkID     string        `yaml:"WorkloadNetworkID" json:"workloadnetworkid"`
	WorkloadNetworkName   string        `yaml:"WorkloadNetworkName" json:"workloadnetworkname"`
	StorageNetworkID      string        `yaml:"StorageNetworkID" json:"storagenetworkid"`
	StorageNetworkName    string        `yaml:"StorageNetworkName" json:"storagenetworkname"`
	RegionName            string        `yaml:"RegionName" json:"regionname"`
	OrganizationID        string        `yaml:"OrganizationID" json:"organizationid"`
	CloudCentralKey       string        `yaml:"CloudCentralKey" json:"cloudcentralkey"`
	Solidfire             Solidfire     `yaml:"Solidfire,omitempty" json:"solidfire,omitempty"`
	IPAM                  IPAMConfig    `yaml:"IPAM,omitempty" json:"ipam,omitempty"`
	ProxySettings         ProxySettings `yaml:"ProxySettings,omitempty" json:"proxysettings,omitempty"`
	OptionalConfiguration Configuration `yaml:"Configuration" json:"configuration,omitempty"`
}

// Configuration holds optional configuration values
type Configuration struct {
	DisableCleanup        bool `yaml:"-" json:"-"`
	DisableHALoadbalancer bool `yaml:"-" json:"-"`

	OVA        OVASpec       `yaml:"OVA,omitempty" json:"ova,omitempty"`
	Cluster    ClusterSpec   `yaml:"Cluster,omitempty" json:"cluster,omitempty"`
	Components ComponentSpec `yaml:"Components,omitempty" json:"components,omitempty"`
	Bintray    BintraySpec   `yaml:"Bintray,omitempty" json:"bintray,omitempty"`

	Observability ObservabilitySpec `yaml:"Observability,omitempty" json:"observability,omitempty"`
}

// ClusterSpec specifies the service cluster
type ClusterSpec struct {
	Name                  string `yaml:"Name,omitempty" json:"name,omitempty"`
	MasterCount           int    `yaml:"MasterCount,omitempty" json:"mastercount,omitempty"`
	MasterSize            string `yaml:"MasterSize,omitempty" json:"mastersize,omitempty"`
	WorkerCount           int    `yaml:"WorkerCount,omitempty" json:"workercount,omitempty"`
	WorkerSize            string `yaml:"WorkerSize,omitempty" json:"workersize,omitempty"`
	KubernetesVersion     string `yaml:"KubernetesVersion,omitempty" json:"kubernetesversion,omitempty"`
	KubernetesPodCidr     string `yaml:"KubernetesPodCidr,omitempty" json:"kubernetespodcidr,omitempty"`
	KubernetesServiceCidr string `yaml:"KubernetesServiceCidr,omitempty" json:"kubernetesservicecidr,omitempty"`
}

// NodeCount returns the total numbers of node in a cluster
func (spec *ClusterSpec) NodeCount() int {
	return spec.MasterCount + spec.WorkerCount
}

// ComponentSpec sets versions for binaries and images that must be downloaded
type ComponentSpec struct {
	ChandlerImage                 string `yaml:"ChandlerImage,omitempty" json:"chandlerimage,omitempty"`
	XDSImage                      string `yaml:"XDSImage,omitempty" json:"xdsimage,omitempty"`
	VSphereManagerImage           string `yaml:"VSphereManagerImage,omitempty" json:"vspheremanagerimage,omitempty"`
	CAPVImage                     string `yaml:"CAPVImage,omitempty" json:"capvimage,omitempty"`
	CABPKImage                    string `yaml:"CABPKImage,omitempty" json:"cabpkimage,omitempty"`
	CAPIImage                     string `yaml:"CAPIImage,omitempty" json:"capiimage,omitempty"`
	ImageManagerVSphereImage      string `yaml:"ImageManagerVSphereImage,omitempty" json:"imagemanagervsphereimage,omitempty"`
	ClusterUpgradeControllerImage string `yaml:"ClusterUpgradeControllerImage,omitempty" json:"clusterupgradecontrollerimage,omitempty"`
}

// BintraySpec sets the connection information for bintray
type BintraySpec struct {
	Target   string `yaml:"Target,omitempty" json:"target,omitempty"`
	Subject  string `yaml:"Subject,omitempty" json:"subject,omitempty"`
	BasePath string `yaml:"BasePath,omitempty" json:"basepath,omitempty"`
	User     string `yaml:"User,omitempty" json:"user,omitempty"`
	Token    string `yaml:"Token,omitempty" json:"token,omitempty"`
}

// OVASpec sets OVA information used for virtual machine templates
type OVASpec struct {
	NodeTemplate                string `yaml:"NodeTemplate,omitempty" json:"nodetemplate,omitempty"`
	LoadbalancerTemplate        string `yaml:"LoadbalancerTemplate,omitempty" json:"loadbalancertemplate,omitempty"`
	LoadbalancerTemplateVersion string `yaml:"LoadbalancerTemplateVersion,omitempty" json:"loadbalancertemplateversion,omitempty"`
}

// Solidfire holds information needed to configure Trident against element
type Solidfire struct {
	Enable   bool   `yaml:"Enable" json:"enable"`
	MVIP     string `yaml:"MVIP" json:"mvip"`
	SVIP     string `yaml:"SVIP" json:"svip"`
	User     string `yaml:"User" json:"user"`
	Password string `yaml:"Password" json:"password"`
}

// IPAMProvider controls what IP address management provider will be used for the region
type IPAMProvider string

const (
	DHCP           IPAMProvider = "DHCP"
	MNodeIPService IPAMProvider = "MNodeIPService"
	Infoblox       IPAMProvider = "Infoblox"
)

type IPAMConfig struct {
	Provider IPAMProvider   `yaml:"Provider" json:"provider"`
	MNode    MNodeConfig    `yaml:"MNodeConfig,omitempty" json:"mnodeconfig,omitempty"`
	Infoblox InfobloxConfig `yaml:"InfobloxConfig,omitempty" json:"infobloxconfig,omitempty"`
}

type MNodeConfig struct {
	IP          string `yaml:"IP" json:"ip"`
	Path        string `yaml:"Path" json:"path"`
	Version     string `yaml:"Version" json:"version"`
	AuthHostURL string `yaml:"AuthHostURL" json:"authhosturl"`
	AuthSecret  string `yaml:"AuthSecret" json:"authsecret"`
	TLSInsecure bool   `yaml:"TLSInsecure" json:"tlsinsecure"`
}

type InfobloxConfig struct {
	Host      string            `yaml:"Host" json:"host"`
	Port      string            `yaml:"Port" json:"port"`
	User      string            `yaml:"User" json:"user"`
	Password  string            `yaml:"Password" json:"password"`
	TenantID  string            `yaml:"TenantID" json:"tenantid" `
	Version   string            `yaml:"Version" json:"version"`
	SSLVerify bool              `yaml:"SSLVerify" json:"sslverify"`
	Networks  []InfobloxNetwork `yaml:"Networks" json:"networks"`
}

type InfobloxNetwork struct {
	NetworkCIDR   string   `yaml:"NetworkCIDR" json:"networkcidr"`
	Gateway       string   `yaml:"Gateway" json:"gateway"`
	NetworkTypes  []string `yaml:"NetworkTypes" json:"networktypes"`
	DNSServers    []string `yaml:"DNSServers" json:"dnsservers"`
	NTPServers    []string `yaml:"NTPServers" json:"ntpservers"`
	SearchDomains []string `yaml:"SearchDomains" json:"searchdomains"`
	EnableHostDNS bool     `yaml:"EnableHostDNS" json:"enablehostdns"`
	HostDNSSuffix string   `yaml:"HostDNSSuffix" json:"hostdnssuffix"`
}

type ProxySettings struct {
	Enable   bool   `yaml:"Enable" json:"enable"`
	HostIp   string `yaml:"HostIP" json:"hostip"`
	Port     int    `yaml:"Port" json:"port"`
	SshPort  int    `yaml:"SSHPort" json:"sshport"`
	Username string `yaml:"Username" json:"username"`
	Password string `yaml:"Password" json:"password"`
}

// ObservabilitySpec holds values for the observability archive file
type ObservabilitySpec struct {
	Enabled         bool   `yaml:"Enabled" json:"enabled"`
	ArchiveLocation string `yaml:"ArchiveLocation" json:"archivelocation"`
}
