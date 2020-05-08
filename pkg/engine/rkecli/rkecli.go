package rkecli

import (
	"fmt"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/util/cmd"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultConfigPath  = "/rke-config.yml"
	defaultHostname    = "my.rancher.org"
	certManagerCRDURL  = "https://raw.githubusercontent.com/jetstack/cert-manager/release-0.12/deploy/manifests/00-crds.yaml"
	certManagerVersion = "v0.12.0"
)

// NewMgmtClusterCli creates a new cluster interface with a full config from the client
func NewMgmtClusterCli() *MgmtCluster {
	mc := &MgmtCluster{}
	mc.EventStream = make(chan progress.Event)
	if mc.LogFile != "" {
		cmd.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	return mc
}

// MgmtCluster spec for RKE
type MgmtCluster struct {
	EventStream             chan progress.Event
	engine.MgmtCluster      `yaml:",inline" mapstructure:",squash"`
	vsphere.ProviderVsphere `yaml:",inline" mapstructure:",squash"`
	token                   string
	clusterURL              string
	RKEConfigPath           string            `yaml:"RKEConfigPath"`
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

// CreateBootstrap is not needed for rkecli
func (c MgmtCluster) CreateBootstrap() error {
	log.Info("CreateBootstrap nothing to do...")
	return nil
}

// InstallControlPlane helm installs rancher server
func (c *MgmtCluster) InstallControlPlane() error {
	log.Info("InstallControlPlan nothing to do...")
	return nil
}

// CreatePermanent deploys HA RKE cluster to provided nodes
func (c *MgmtCluster) CreatePermanent() error {
	c.EventStream <- progress.Event{Type: "progress", Msg: "install HA rke cluster"}

	if c.RKEConfigPath == "" {
		log.Warnf("RKEConfigPath not provided in cake config, defaulting to %s", defaultConfigPath)
		c.RKEConfigPath = defaultConfigPath
	}
	if c.Hostname == "" {
		log.Warnf("Hostname not provided in cake config, defaulting to %s", defaultHostname)
		c.Hostname = defaultHostname
	}

	var y map[string]interface{}
	err := yaml.Unmarshal([]byte(rawClusterYML), &y)
	if err != nil {
		return fmt.Errorf("error unmarshaling RKE cluster config file: %s", err)
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
			SSHKeyPath:       c.SSH.KeyPath,
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
	y["ssh_key_path"] = c.SSH.KeyPath

	clusterYML, err := yaml.Marshal(y)
	if err != nil {
		return fmt.Errorf("error marshaling RKE cluster config file: %s", err)
	}
	err = ioutil.WriteFile(c.RKEConfigPath, clusterYML, 0644)
	if err != nil {
		return fmt.Errorf("error writing RKE cluster config file to file %s: %s", c.RKEConfigPath, err)
	}

	cmd.FileLogLocation = c.LogFile
	args := []string{
		"up",
		"--config=" + c.RKEConfigPath,
	}
	err = cmd.GenericExecute(nil, "rke", args, nil)
	if err != nil {
		return fmt.Errorf("error running rke up cmd: %s", err)
	}

	return nil
}

// PivotControlPlane deploys rancher server via helm chart to HA RKE cluster
func (c MgmtCluster) PivotControlPlane() error {
	kubeConfigFile := filepath.Join(filepath.Dir(c.RKEConfigPath), fmt.Sprintf("kube_config_%s", filepath.Base(c.RKEConfigPath)))
	namespace := "cattle-system"
	rVersion := "rancher-stable"
	args := []string{
		"repo",
		"add",
		rVersion,
		"https://releases.rancher.com/server-charts/stable",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err := cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error adding rancher helm chart: %s", err)
	}
	log.Infof("added %s helm chart", rVersion)
	args = []string{
		"repo",
		"list",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error reading helm chart: %s", err)
	}

	args = []string{
		"repo",
		"add",
		"jetstack",
		"https://charts.jetstack.io",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error adding jetstack helm chart: %s", err)
	}
	log.Infof("added cert-manager helm chart")

	kubeCfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return err
	}

	kube, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		return err
	}

	_, err = kube.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		log.Warnf("Suppressing error creating %s namespace: %s", namespace, err.Error())
	}

	_, err = kube.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cert-manager",
		},
	})
	if err != nil {
		log.Warnf("Suppressing error creating cert-manager namespace: %s", err.Error())
	}

	log.Infof("created namespaces")

	args = []string{
		"apply",
		"-f",
		certManagerCRDURL,
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "kubectl", args, nil)
	if err != nil {
		return fmt.Errorf("error installing cert-manager CRD: %s", err)
	}
	log.Infof("installed cert-manager CRD")

	args = []string{
		"repo",
		"update",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error updating helm charts: %s", err)
	}
	log.Infof("updated helm chart")

	args = []string{
		"install",
		"cert-manager",
		"jetstack/cert-manager",
		fmt.Sprintf("--namespace=cert-manager"),
		fmt.Sprintf("--version=%s", certManagerVersion),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error installing cert-manager helm chart: %s", err)
	}
	log.Infof("helm installed cert-manager")

	log.Infof("waiting for cert-manager to be ready")
	args = []string{
		"rollout",
		"status",
		"deploy/cert-manager",
		"--namespace=cert-manager",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "kubectl", args, nil)
	if err != nil {
		return fmt.Errorf("error waiting for cert-manager: %s", err)
	}

	log.Infof("helm installing rancher")
	args = []string{
		"install",
		"rancher",
		fmt.Sprintf("%s/rancher", rVersion),
		fmt.Sprintf("--namespace=%s", namespace),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
		"--set",
		fmt.Sprintf("hostname=%s", c.Hostname),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		log.Warnf("suppressing error running rancher helm install: %s", err)
	}

	log.Infof("waiting for rancher to be ready")
	args = []string{
		"rollout",
		"status",
		"deploy/rancher",
		fmt.Sprintf("--namespace=%s", namespace),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "kubectl", args, nil)
	if err != nil {
		return fmt.Errorf("error waiting for rancher: %s", err)
	}

	log.Infof("waiting for nginx ingress to be ready")
	args = []string{
		"rollout",
		"status",
		"deploy/default-http-backend",
		"--namespace=ingress-nginx",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "kubectl", args, nil)
	if err != nil {
		return fmt.Errorf("error waiting for nginx ingress: %s", err)
	}

	var workerNode string
	for k, v := range c.Nodes {
		if strings.HasPrefix(k, "worker") {
			workerNode = v
			break
		}
	}

	rServerURL := fmt.Sprintf("https://%s", c.Hostname)

	log.Infof("Make sure hostname %s resolves to %s or a worker node IP", c.Hostname, workerNode)
	log.Infof("HA rancher install complete: %s", rServerURL)
	return nil
}

// Events returns the channel of progress messages
func (c MgmtCluster) Events() chan progress.Event {
	return c.EventStream
}
