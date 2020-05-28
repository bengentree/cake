package rkecli

import (
	"fmt"
	"github.com/netapp/cake/pkg/config/cluster"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/netapp/cake/pkg/progress"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/netapp/cake/pkg/config"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/util/cmd"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultConfigPath  = "/rke-config.yml"
	defaultHostname    = "my.rancher.org"
	certManagerCRDURL  = "https://github.com/jetstack/cert-manager/releases/download/v0.15.0/cert-manager.crds.yaml"
	certManagerVersion = "v0.15.0"
	rancherVersion     = "2.4.3"
)

// NewMgmtClusterCli creates a new cluster interface with a full config from the client
func NewMgmtClusterCli() *MgmtCluster {
	mc := new(MgmtCluster)
	if mc.LogFile != "" {
		cmd.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	return mc
}

// MgmtCluster spec for RKE
type MgmtCluster struct {
	engine.MgmtCluster      `yaml:",inline" mapstructure:",squash"`
	vsphere.ProviderVsphere `yaml:",inline" mapstructure:",squash"`
	token                   string
	clusterURL              string
	RKEConfigPath           string            `yaml:"RKEConfigPath"`
	Nodes                   map[string]string `yaml:"Nodes" json:"nodes"`
	Hostname                string            `yaml:"Hostname"`
	// SSL configuration defaults to Rancher-generated TLS certificate
	SSL *cluster.SSL `yaml:"SSL,omitempty"`
}

// InstallAddons to HA RKE cluster
func (c MgmtCluster) InstallAddons() error {
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "TODO: install addons",
		Level: "debug",
	})
	return nil
}

// RequiredCommands provides validation for required commands
func (c MgmtCluster) RequiredCommands() []string {
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "TODO: provide required commands",
		Level: "debug",
	})
	return nil
}

// CreateBootstrap is not needed for rkecli
func (c MgmtCluster) CreateBootstrap() error {
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "CreateBootstrap - nothing to do",
		Level: "debug",
	})
	return nil
}

// InstallControlPlane helm installs rancher server
func (c *MgmtCluster) InstallControlPlane() error {
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "InstallControlPlane - nothing to do",
		Level: "debug",
	})
	return nil
}

// Spec returns the Spec
func (c *MgmtCluster) Spec() engine.MgmtCluster {
	c.MgmtCluster.FileDeliverables = []string{
		c.RKEConfigPath,
		"/kube_config_rke-config.yml",
	}
	return c.MgmtCluster
}

// CreatePermanent deploys HA RKE cluster to provided nodes
func (c *MgmtCluster) CreatePermanent() error {
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Installing HA RKE cluster",
		Level: "info",
	})

	if c.RKEConfigPath == "" {
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("RKEConfigPath not provided in cake config, defaulting to %s", defaultConfigPath),
			Level: "debug",
		})
		c.RKEConfigPath = defaultConfigPath
	}
	if c.Hostname == "" {
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("Hostname not provided in cake config, defaulting to %s", defaultHostname),
			Level: "debug",
		})
		c.Hostname = defaultHostname
	}

	var y map[string]interface{}
	err := yaml.Unmarshal([]byte(rawClusterYML), &y)
	if err != nil {
		return fmt.Errorf("error unmarshaling RKE cluster config file: %s", err)
	}

	var sans []string
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
		masterPrefix := fmt.Sprintf("%s-%s", c.ClusterName, config.ControlNode)
		if strings.HasPrefix(strings.ToLower(k), masterPrefix) {
			node.Role = append(node.Role, config.ControlNode)
			sans = append(sans, v)
		} else {
			node.Role = append(node.Role, config.WorkerNode)
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 1 {
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "Non-HA RKE deployment detected, at least 3 nodes recommended",
			Level: "debug",
		})
		nodes[0].Role = []string{"controlplane", "worker", "etcd"}
	}

	// etcd requires an odd number of nodes, first role on each node is etcd.
	if len(nodes)%2 == 0 {
		lastNode := nodes[len(nodes)-1]
		lastNode.Role = lastNode.Role[1:]
	}

	y["nodes"] = nodes
	y["ssh_key_path"] = c.SSH.KeyPath
	sans = append(sans, c.Hostname)
	y["authentication"] = map[string]interface{}{
		"sans":     sans,
		"strategy": "x509",
		"webhook":  nil,
	}

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
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("Added %s Helm chart repo", rVersion),
		Level: "info",
	})
	args = []string{
		"repo",
		"list",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error reading helm chart: %s", err)
	}

	if c.SSL == nil {
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
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "Added cert-manager Helm chart",
			Level: "info",
		})
	}

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
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("Suppressing error creating %s namespace: %s", namespace, err.Error()),
			Level: "debug",
		})
	}

	if c.SSL == nil {
		_, err = kube.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cert-manager",
			},
		})
		if err != nil {
			c.EventStream.Publish(&progress.StatusEvent{
				Type:  "progress",
				Msg:   fmt.Sprintf("Suppressing error creating cert-manager namespace: %s", err.Error()),
				Level: "debug",
			})
		}

	}

	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("Created %v namespace", namespace),
		Level: "info",
	})

	if c.SSL == nil {
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
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "Installed cert-manager CRD",
			Level: "info",
		})
	}

	args = []string{
		"repo",
		"update",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		return fmt.Errorf("error updating helm charts: %s", err)
	}
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Updated Helm Repos",
		Level: "info",
	})

	if c.SSL == nil {
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
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "Helm installed cert-manager",
			Level: "info",
		})

		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "Waiting for cert-manager to be ready",
			Level: "info",
		})
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
	}

	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Helm installing Rancher",
		Level: "info",
	})
	certManagerCompatVersion := certManagerVersion
	ingressTLSSource := "rancher"
	privateCA := false
	if c.SSL != nil {
		certManagerCompatVersion = ""
		ingressTLSSource = "secret"
		privateCA = c.SSL.PrivateCA
	}
	args = []string{
		"install",
		"rancher",
		fmt.Sprintf("%s/rancher", rVersion),
		fmt.Sprintf("--version=%s", rancherVersion),
		fmt.Sprintf("--namespace=%s", namespace),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
		"--set",
		fmt.Sprintf("%s,%s,%s,%s",
			fmt.Sprintf("hostname=%s", c.Hostname),
			fmt.Sprintf("certmanager.version=%s", certManagerCompatVersion),
			fmt.Sprintf("ingress.tls.source=%s", ingressTLSSource),
			fmt.Sprintf("privateCA=%s", privateCA),
		),
	}
	err = cmd.GenericExecute(nil, "helm", args, nil)
	if err != nil {
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("Suppressing error running rancher helm install: %s", err.Error()),
			Level: "debug",
		})
	}

	if c.SSL != nil {
		if err := c.addTLSSecret(namespace, kubeConfigFile); err != nil {
			return fmt.Errorf("unable to add TLS secrets: %v", err)
		}
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "TLS secret created",
			Level: "debug",
		})
	}

	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Waiting for Rancher to be ready",
		Level: "info",
	})
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

	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   "Waiting for nginx ingress to be ready",
		Level: "info",
	})
	args = []string{
		"rollout",
		"status",
		"deploy/default-http-backend",
		"--namespace=ingress-nginx",
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
	}
	err = cmd.GenericExecute(nil, "kubectl", args, nil)
	if err != nil {
		return fmt.Errorf("error waiting for nginx ingress: %s", err.Error())
	}

	if c.SSL == nil {
		if err := c.rancherIssuerWorkaround(kubeCfg, namespace, kubeConfigFile); err != nil {
			return fmt.Errorf("error attempting rancher issuer workaround: %s", err)
		}
	}

	var workerNode string
	workerPrefix := fmt.Sprintf("%s-%s", c.ClusterName, config.WorkerNode)
	for k, v := range c.Nodes {
		if strings.HasPrefix(k, workerPrefix) {
			workerNode = v
			break
		}
	}

	rServerURL := fmt.Sprintf("https://%s", c.Hostname)

	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("Make sure hostname %s resolves to %s or a worker node IP", c.Hostname, workerNode),
		Level: "info",
	})
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("HA Rancher install complete: %s", rServerURL),
		Level: "info",
	})
	return nil
}

// Events returns the channel of progress messages
func (c MgmtCluster) Events() progress.Events {
	return c.EventStream
}

func (c MgmtCluster) addTLSSecret(namespace, kubeConfigFile string) error {
	if !c.SSL.PrivateCA {
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "WARNING: public CA not yet supported, manually add TLS secrets!",
			Level: "info",
		})
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		exe = "/tmp/cake"
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("Suppressing error getting the executable, defaulting to '%s': %v", exe, err),
			Level: "info",
		})
	}
	certFile := path.Join(path.Dir(exe), "cacerts.pem")
	if err := ioutil.WriteFile(certFile, []byte(c.SSL.Certificate), 0644); err != nil {
		return fmt.Errorf("error writing the cert to file %s: %v", certFile, err)
	}

	args := []string{
		"create",
		"secret",
		"generic",
		"tls-ca",
		fmt.Sprintf("--namespace=%s", namespace),
		fmt.Sprintf("--kubeconfig=%s", kubeConfigFile),
		fmt.Sprintf("--from-file=%s", certFile),
	}
	if err := cmd.GenericExecute(nil, "kubectl", args, nil); err != nil {
		return fmt.Errorf("error creating the private TLS CA: %v", err)
	}
	return nil
}

func (c MgmtCluster) rancherIssuerWorkaround(kubeCfg *restclient.Config, ns, kubeCfgFile string) error {
	err := waitForRancherIssuer(ns, kubeCfgFile)
	if err == nil {
		c.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   "Rancher Issuer deployed successfully",
			Level: "info",
		})
		return nil
	}

	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("Creating Rancher Issuer"),
		Level: "info",
	})
	c.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("Rancher Issuer failed to deploy, recreating: %s", err.Error()),
		Level: "debug",
	})

	client, err := dynamic.NewForConfig(kubeCfg)
	if err != nil {
		return fmt.Errorf("unable to create dynamic k8s client: %s", err.Error())
	}

	issuerRes := schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1alpha2",
		Resource: "issuers",
	}

	// https://github.com/rancher/rancher/blob/master/chart/templates/issuer-rancher.yaml
	issuer := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1alpha2",
			"kind":       "Issuer",
			"metadata": map[string]interface{}{
				"name": "rancher",
				"labels": map[string]interface{}{
					"app":      "rancher",
					"chart":    fmt.Sprintf("rancher-%s", rancherVersion),
					"heritage": "Helm",
					"release":  "rancher",
				},
			},
			"spec": map[string]interface{}{
				"ca": map[string]interface{}{
					"secretName": "tls-rancher",
				},
			},
		},
	}

	_, err = client.Resource(issuerRes).Namespace(ns).Create(issuer, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create issuer resource: %s", err)
	}
	return waitForRancherIssuer(ns, kubeCfgFile)
}

func waitForRancherIssuer(ns, kubeCfg string) error {
	args := []string{
		"wait",
		"issuer",
		"rancher",
		"--for",
		"condition=ready",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--kubeconfig=%s", kubeCfg),
	}
	return cmd.GenericExecute(nil, "kubectl", args, nil)
}
