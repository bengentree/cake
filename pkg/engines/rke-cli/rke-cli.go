package rke_cli

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/ghodss/yaml"
	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config/events"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engines"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type requiredCmd string

const (
	docker requiredCmd = "docker"
)

// RequiredCommands for capv provisioner
var RequiredCommands = cmds.ProvisionerCommands{Name: "required CAPV bootstrap commands"}

func init() {
	d := cmds.NewCommandLine(nil, string(docker), nil, nil)
	RequiredCommands.AddCommand(d.CommandName, d)
}

// NewMgmtClusterFullConfig creates a new cluster interface with a full config from the client
func NewMgmtClusterCli() *MgmtCluster {
	mc := &MgmtCluster{}
	mc.EventStream = make(chan events.Event)
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	mc.dockerCli = new(dockerCli)
	mc.osCli = new(osCli)
	return mc
}

// MgmtCluster spec for RKE
type MgmtCluster struct {
	EventStream             chan events.Event
	engines.MgmtCluster     `yaml:",inline" mapstructure:",squash"`
	vsphere.ProviderVsphere `yaml:",inline" mapstructure:",squash"`
	token                   string
	clusterURL              string
	rancherClient           *v3.Client
	BootstrapIP             string            `yaml:"BootstrapIP"`
	Nodes                   map[string]string `yaml:"Nodes" json:"nodes"`
	dockerCli               dockerCmds
	osCli                   genericCmds
}

type dockerCmds interface {
	NewEnvClient() (*client.Client, error)
	ContainerCreate(ctx context.Context, cli *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, cli *client.Client, containerID string, options types.ContainerStartOptions) error
}

type dockerCli struct{}

func (dockerCli) NewEnvClient() (*client.Client, error) {
	return client.NewEnvClient()
}

func (dockerCli) ContainerCreate(ctx context.Context, cli *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	return cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
}

func (dockerCli) ContainerStart(ctx context.Context, cli *client.Client, containerID string, options types.ContainerStartOptions) error {
	return cli.ContainerStart(ctx, containerID, options)
}

type genericCmds interface {
	GenericExecute(envs map[string]string, name string, args []string, ctx *context.Context) error
}

type osCli struct{}

func (osCli) GenericExecute(envs map[string]string, name string, args []string, ctx *context.Context) error {
	return cmds.GenericExecute(envs, name, args, ctx)
}

// InstallAddons to HA RKE cluster
func (c MgmtCluster) InstallAddons() error {
	log.Infof("TODO: install addons")
	return nil
}

// RequiredCommands provides validation for required commands
func (c MgmtCluster) RequiredCommands() []string {
	log.Infof("TODO: provide required commands")
	return nil
}

// CreateBootstrap deploys a rancher container as single node RKE cluster
func (c MgmtCluster) CreateBootstrap() error {
	//c.EventStream <- events.Event{EventType: "progress", Event: "configure control plane node for RKE install"}
	//
	//args := []string{
	//	"sudo",
	//	"usermod",
	//	"-aG",
	//	"docker",
	//	c.SSH.Username,
	//	"&&",
	//	"newgrp",
	//	"docker",
	//}
	//err := c.osCli.GenericExecute(nil, "Add SSH user to docker group", args, nil)
	//if err != nil {
	//	return err
	//}
	//
	//cli, err := c.dockerCli.NewEnvClient()
	//if err != nil {
	//	return err
	//}
	//
	//// Ensure docker connectivity as SSH user: https://rancher.com/docs/rke/latest/en/troubleshooting/ssh-connectivity-errors/
	//_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	//if err != nil {
	//	return err
	//}
	//
	//args = []string{
	//	"ssh-keygen",
	//	"-f",
	//	"~/.ssh/id_rsa",
	//	"-t",
	//	"rsa",
	//	"-N",
	//	"",
	//}
	//err = c.osCli.GenericExecute(nil, "Generate SSH keys", args, nil)
	//if err != nil {
	//	return err
	//}
	//
	//for k, v := range c.Nodes {
	//	if !strings.HasPrefix(k, "workerNode") {
	//		continue
	//	}
	//
	//
	//}
	//
	//// I could not find away to exec RKE install without SSH tunnel, so allowing SSH from auth user
	//args = []string{
	//	"cat",
	//	c.SSH.AuthorizedKey,
	//	">>",
	//	"~/.ssh/authorized_keys",
	//}
	//err = c.osCli.GenericExecute(nil, "Add SSH key to authorized keys", args, nil)
	//if err != nil {
	//	return err
	//}
	return nil
}

// InstallControlPlane configures a single node RKE cluster
func (c *MgmtCluster) InstallControlPlane() error {
	c.EventStream <- events.Event{EventType: "progress", Event: "install HA rke cluster"}

	rkeConfig := &v3.RancherKubernetesEngineConfig{
		Nodes: make([]v3.RKEConfigNode, len(c.Nodes)),
		Services: v3.RKEConfigServices{
			Etcd:           v3.ETCDService{},
			KubeAPI:        v3.KubeAPIService{},
			KubeController: v3.KubeControllerService{},
			Scheduler:      v3.SchedulerService{},
			Kubelet:        v3.KubeletService{},
			Kubeproxy:      v3.KubeproxyService{},
		},
		Network:             v3.NetworkConfig{},
		Authentication:      v3.AuthnConfig{},
		Addons:              "",
		AddonsInclude:       nil,
		SystemImages:        v3.RKESystemImages{},
		SSHKeyPath:          "",
		SSHCertPath:         "",
		SSHAgentAuth:        false,
		Authorization:       v3.AuthzConfig{},
		IgnoreDockerVersion: false,
		Version:             "",
		PrivateRegistries:   nil,
		Ingress:             v3.IngressConfig{},
		ClusterName:         "",
		CloudProvider:       v3.CloudProvider{},
		PrefixPath:          "",
		AddonJobTimeout:     0,
		BastionHost:         v3.BastionHost{},
		Monitoring:          v3.MonitoringConfig{},
		Restore:             v3.RestoreConfig{},
		RotateCertificates:  nil,
		DNS:                 nil,
	}

	for k, v := range c.Nodes {
		node := &v3.RKEConfigNode{
			Address:          v,
			Port:             "22",
			InternalAddress:  "",
			Role:             []string{"etcd"},
			HostnameOverride: "",
			User:             c.SSH.Username,
			DockerSocket:     "/var/run/docker.sock",
			SSHKeyPath:       "~/.ssh/id_rsa",
			SSHCert:          "",
			SSHCertPath:      "",
			Labels:           make(map[string]string),
			Taints:           make([]v3.RKETaint, 0),
		}
		if strings.HasPrefix(k, "controlPlane") {
			node.Role = append(node.Role, "controlplane")
		} else {
			node.Role = append(node.Role, "worker")
		}
	}

	// etcd requires an odd number of nodes, first role on each node is etcd.
	if len(rkeConfig.Nodes)%2 == 0 {
		lastNode := rkeConfig.Nodes[len(rkeConfig.Nodes)-1]
		lastNode.Role = lastNode.Role[1:]
	}
	clusterYML, err := yaml.Marshal(rkeConfig)
	if err != nil {
		return err
	}
	yamlFile := "~/rke-cluster.yml"
	err = ioutil.WriteFile(yamlFile, clusterYML, 0644)
	if err != nil {
		return err
	}

	args := []string{
		"rke",
		"up",
		"--config",
		yamlFile,
	}
	err = c.osCli.GenericExecute(nil, "rke up", args, nil)
	if err != nil {
		return err
	}

	return nil
}

// CreatePermanent deploys HA RKE cluster to vSphere
func (c *MgmtCluster) CreatePermanent() error {
	return nil
}

// PivotControlPlane deploys rancher server via helm chart to HA RKE cluster
func (c MgmtCluster) PivotControlPlane() error {
	return nil
}

// Events returns the channel of progress messages
func (c MgmtCluster) Events() chan events.Event {
	return c.EventStream
}
