package rkecli

import (
	"bufio"
	"context"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config/events"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engines"
	log "github.com/sirupsen/logrus"
)

// NewMgmtClusterCli creates a new cluster interface with a full config from the client
func NewMgmtClusterCli() *MgmtCluster {
	mc := &MgmtCluster{}
	mc.EventStream = make(chan events.Event)
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	return mc
}

// MgmtCluster spec for RKE
type MgmtCluster struct {
	EventStream             chan events.Event
	engines.MgmtCluster     `yaml:",inline" mapstructure:",squash"`
	vsphere.ProviderVsphere `yaml:",inline" mapstructure:",squash"`
	token                   string
	clusterURL              string
	BootstrapIP             string            `yaml:"BootstrapIP"`
	Nodes                   map[string]string `yaml:"Nodes" json:"nodes"`
	Hostname                string            `yaml:"Hostname"`
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

// CreateBootstrap is not needed for rke-cli
func (c MgmtCluster) CreateBootstrap() error {
	log.Info("Nothing to do...")
	return nil
}

// InstallControlPlane helm installs rancher server
func (c *MgmtCluster) InstallControlPlane() error {
	log.Info("Nothing to do...")
	return nil
}

// CreatePermanent deploys HA RKE cluster to provided nodes
func (c *MgmtCluster) CreatePermanent() error {
	c.EventStream <- events.Event{EventType: "progress", Event: "install HA rke cluster"}

	var y map[string]interface{}
	err := yaml.Unmarshal([]byte(rawClusterYML), &y)
	if err != nil {
		return err
	}

	nodes := make([]*rkeConfigNode, 0)
	for k, v := range c.Nodes {
		node := &rkeConfigNode{
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
			Taints:           make([]rkeTaint, 0),
		}
		if strings.HasPrefix(k, "controlplane") {
			node.Role = append(node.Role, "controlplane")
		} else {
			node.Role = append(node.Role, "worker")
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 1 {
		log.Warnf("Non-HA RKE deployment, at least 3 nodes recommended")
		nodes[0].Role = []string{"controlplane", "worker", "etcd"}
	}

	// etcd requires an odd number of nodes, first role on each node is etcd.
	if len(nodes)%2 == 0 {
		lastNode := nodes[len(nodes)-1]
		lastNode.Role = lastNode.Role[1:]
	}

	y["nodes"] = nodes

	clusterYML, err := yaml.Marshal(y)
	if err != nil {
		return err
	}
	yamlFile := "rke-cluster.yml"
	err = ioutil.WriteFile(yamlFile, clusterYML, 0644)
	if err != nil {
		return err
	}

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
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
		}
	}(ctx)

	err = cmd.Wait()
	ctx.Done()
	return err
}

// PivotControlPlane deploys rancher server via helm chart to HA RKE cluster
func (c MgmtCluster) PivotControlPlane() error {
	log.Infof("TODO: helm install rancher server")
	return nil
}

// Events returns the channel of progress messages
func (c MgmtCluster) Events() chan events.Event {
	return c.EventStream
}
