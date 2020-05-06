package rkecli

// Due to dependency issues, copied from
// https://github.com/rancher/types/blob/master/apis/management.cattle.io/v3/rke_types.go

type rkeConfigNode struct {
	// Name of the host provisioned via docker machine
	NodeName string `yaml:"nodeName,omitempty" json:"nodeName,omitempty" norman:"type=reference[node]"`
	// IP or FQDN that is fully resolvable and used for SSH communication
	Address string `yaml:"address" json:"address,omitempty"`
	// Port used for SSH communication
	Port string `yaml:"port" json:"port,omitempty"`
	// Optional - Internal address that will be used for components communication
	InternalAddress string `yaml:"internal_address" json:"internalAddress,omitempty"`
	// Node role in kubernetes cluster (controlplane, worker, or etcd)
	Role []string `yaml:"role" json:"role,omitempty" norman:"type=array[enum],options=etcd|worker|controlplane"`
	// Optional - Hostname of the node
	HostnameOverride string `yaml:"hostname_override" json:"hostnameOverride,omitempty"`
	// SSH usesr that will be used by RKE
	User string `yaml:"user" json:"user,omitempty"`
	// Optional - Docker socket on the node that will be used in tunneling
	DockerSocket string `yaml:"docker_socket" json:"dockerSocket,omitempty"`
	// SSH Agent Auth enable
	SSHAgentAuth bool `yaml:"ssh_agent_auth,omitempty" json:"sshAgentAuth,omitempty"`
	// SSH Private Key
	SSHKey string `yaml:"ssh_key" json:"sshKey,omitempty" norman:"type=password"`
	// SSH Private Key Path
	SSHKeyPath string `yaml:"ssh_key_path" json:"sshKeyPath,omitempty"`
	// SSH Certificate
	SSHCert string `yaml:"ssh_cert" json:"sshCert,omitempty"`
	// SSH Certificate Path
	SSHCertPath string `yaml:"ssh_cert_path" json:"sshCertPath,omitempty"`
	// Node Labels
	Labels map[string]string `yaml:"labels" json:"labels,omitempty"`
	// Node Taints
	Taints []rkeTaint `yaml:"taints" json:"taints,omitempty"`
}

type rkeTaint struct {
	Key       string         `json:"key,omitempty" yaml:"key"`
	Value     string         `json:"value,omitempty" yaml:"value"`
}


