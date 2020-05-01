package rke_cli

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/ghodss/yaml"
	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config/events"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engines"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"io/ioutil"
	"os"
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
	//rancherClient           *v3.Client
	BootstrapIP             string            `yaml:"BootstrapIP"`
	Nodes                   map[string]string `yaml:"Nodes" json:"nodes"`
	Hostname				string `yaml:"Hostname"`
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

	/*
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
	*/

	//return nil
	var y map[string]interface{}
	err := yaml.Unmarshal([]byte(clusterYMLUnused), &y)
	if err != nil {
		return err
	}

	log.Info(len(c.Nodes))
	nodeKeys := make([]string, 0)
	for k := range c.Nodes {
		nodeKeys = append(nodeKeys, k)
	}
	log.Info(len(nodeKeys))

	log.Info(nodeKeys)
	nodes := y["nodes"].([]interface{})
	for i, key := range nodeKeys {
		log.Infof("i: %d key: %s", i, key)
		log.Infof("setting node %s with ip %s", key, c.Nodes[nodeKeys[i]])
		node := nodes[i].(map[string]interface{})
		node["address"] = c.Nodes[nodeKeys[i]]
		node["user"] = c.SSH.Username
	}




	clusterYML, err := yaml.Marshal(y)
	//clusterYML, err := yaml.Marshal(rkeConfig)
	if err != nil {
		return err
	}
	yamlFile := "rke-cluster.yml"
	err = ioutil.WriteFile(yamlFile, clusterYML, 0644)
	if err != nil {
		return err
	}

	//args := []string{
	//	"up",
	//	"--config",
	//	yamlFile,
	//}
	//err = c.osCli.GenericExecute(nil, "rke", args, nil)
	//if err != nil {
	//	return err
	//}

	// https://gist.github.com/hivefans/ffeaf3964924c943dd7ed83b406bbdea
	cmd := exec.Command("rke", "up", "--config", yamlFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {

	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	r := bufio.NewReader(stdout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute * 5)
	defer cancel()
	//stop := make(chan bool)
	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			default:
				line, _, _ := r.ReadLine()
				lineStr := string(line)
				log.Infoln(lineStr)
				if strings.Contains(lineStr, "FATA") ||
					strings.Contains(lineStr, "Finished") {
					return
				}

			}
			//if shouldStop := <- stop; shouldStop {
			//	return
			//}
		}
	}(ctx)

	err = cmd.Wait()
	ctx.Done()
	return err
}


// CreatePermanent deploys HA RKE cluster to vSphere
func (c *MgmtCluster) CreatePermanent() error {
	/*
	os.Setenv("HELM_KUBETOKEN", "kube_config_rke-cluster.yml")

	// https://github.com/PrasadG193/helm-clientgo-example/blob/master/main.go
	settings := cli.New()
	chRepo, err := repo.NewChartRepository(
		&repo.Entry{
			Name:                  "rancher-latest",
			URL:                   "https://releases.rancher.com/server-charts/latest",
			InsecureSkipTLSverify: true,
		},
		getter.All(settings))
	if err != nil {
		return err
	}

	_, err = chRepo.DownloadIndexFile()

	actionConfig := new(action.Configuration)
	err = actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Infof)
	if err != nil {
		return err
	}

	client := action.NewInstall(actionConfig)
	client.ChartPathOptions.RepoURL ="https://releases.rancher.com/server-charts/latest"
	cp, err := client.ChartPathOptions.LocateChart("rancher-latest", settings)
	if err != nil {
		return err
	}

	p := getter.All(settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		log.Fatal(err)
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	//validInstallableChart, err := isChartInstallable(chartRequested)
	//if !validInstallableChart {
	//	log.Fatal(err)
	//}
	//
	//if req := chartRequested.Metadata.Dependencies; req != nil {
	//	// If CheckDependencies returns an error, we have unfulfilled dependencies.
	//	// As of Helm 2.4.0, this is treated as a stopping condition:
	//	// https://github.com/helm/helm/issues/2209
	//	if err := action.CheckDependencies(chartRequested, req); err != nil {
	//		if client.DependencyUpdate {
	//			man := &downloader.Manager{
	//				Out:              os.Stdout,
	//				ChartPath:        cp,
	//				Keyring:          client.ChartPathOptions.Keyring,
	//				SkipUpdate:       false,
	//				Getters:          p,
	//				RepositoryConfig: settings.RepositoryConfig,
	//				RepositoryCache:  settings.RepositoryCache,
	//			}
	//			if err := man.Update(); err != nil {
	//				log.Fatal(err)
	//			}
	//		} else {
	//			log.Fatal(err)
	//		}
	//	}
	//}

	client.Namespace = settings.Namespace()
	release, err := client.Run(chartRequested, vals)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(release.Manifest)
	*/
	return nil
}

// PivotControlPlane deploys rancher server via helm chart to HA RKE cluster
func (c MgmtCluster) PivotControlPlane() error {
	args := []string{
		"repo",
		"add",
		"rancher-latest",
		"https://releases.rancher.com/server-charts/latest",
	}
	err := c.osCli.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return err
	}

	kubeConfigFile := "kube_config_rke-cluster.yml"
	kubeCfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return err
	}
	
	kube, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		return err
	}
	
	_, err = kube.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: v12.ObjectMeta{
			Name: "cattle-system",
		},
	})
	if err != nil {
		return err
	}

	args = []string{
		"install",
		"rancher",
		"rancher-latest/rancher",
		"--namespace=cattle-system",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
		"--set tls=external",
		//fmt.Sprintf("--set hostname=%s", c.Hostname),
	}
	err = c.osCli.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return err
	}

	//kube.CoreV1().Pods("").Watch()
	return nil
}

// Events returns the channel of progress messages
func (c MgmtCluster) Events() chan events.Event {
	return c.EventStream
}
