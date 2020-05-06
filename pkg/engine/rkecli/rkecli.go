package rkecli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/util/cmd"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	c.EventStream <- progress.Event{Type: "progress", Msg: "install HA rke cluster"}

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
		return err
	}
	err = ioutil.WriteFile(c.RKEConfigPath, clusterYML, 0644)
	if err != nil {
		return err
	}

	cmd.FileLogLocation = c.LogFile
	args := []string{
		"up",
		"--config=" + c.RKEConfigPath,
	}
	err = cmd.GenericExecute(nil, "rke", args, nil)
	if err != nil {
		return err
	}

	return nil
}

// PivotControlPlane deploys rancher server via helm chart to HA RKE cluster
func (c MgmtCluster) PivotControlPlane() error {
	kubeConfigFile := fmt.Sprintf("kube_config_%s", filepath.Base(c.RKEConfigPath))
	namespace := "cattle-system"
	rVersion := "rancher-stable"
	args := []string{
		"repo",
		"add",
		rVersion,
		"https://releases.rancher.com/server-charts/stable",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err := exec.Command("helm", args...).Start()
	if err != nil {
		return err
	}
	log.Infof("added %s helm chart", rVersion)

	args = []string{
		"repo",
		"add",
		"jetstack",
		"https://charts.jetstack.io",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = exec.Command("helm", args...).Start()
	if err != nil {
		return nil
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
		"https://github.com/jetstack/cert-manager/releases/download/v0.14.3/cert-manager.crds.yaml",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	cmd := exec.Command("kubectl", args...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	log.Infof("installed cert-manager CRD")

	args = []string{
		"repo",
		"update",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = exec.Command("helm", args...).Start()
	if err != nil {
		return err
	}
	log.Infof("updated helm chart")

	args = []string{
		"install",
		"cert-manager",
		"jetstack/cert-manager",
		fmt.Sprintf("--namespace=cert-manager"),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	cmd = exec.Command("helm", args...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
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
	cmd = exec.Command("kubectl", args...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	time.Sleep(30 * time.Second)
	log.Infof("helm installing rancher")
	escapedHostname := strings.Replace(c.Hostname, ".", "\\.", 0)
	args = []string{
		"install",
		"rancher",
		fmt.Sprintf("%s/rancher", rVersion),
		fmt.Sprintf("--namespace=%s", namespace),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
		"--set",
		fmt.Sprintf("hostname=%s", escapedHostname),
	}
	log.Infof("helm %s", strings.Join(args, " "))
	cmd = exec.Command("helm", args...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	log.Infof("waiting for rancher to be ready")
	args = []string{
		"rollout",
		"status",
		"deploy/rancher",
		fmt.Sprintf("--namespace=%s", namespace),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	cmd = exec.Command("kubectl", args...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	log.Infof("waiting for nginx ingress to be ready")
	args = []string{
		"rollout",
		"status",
		"deploy/default-http-backend",
		"--namespace=ingress-nginx",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	cmd = exec.Command("kubectl", args...)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
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
