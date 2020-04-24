package capv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netapp/cake/pkg/config/events"

	"github.com/netapp/cake/pkg/cmds"
)

// InstallControlPlane installs CAPv CRDs into the temporary bootstrap cluster
func (m MgmtCluster) InstallControlPlane() error {
	var err error
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, m.ClusterName, VsphereCredsSecret.Name)

	secretSpecContents := fmt.Sprintf(
		VsphereCredsSecret.Contents,
		m.Username,
		m.Password,
	)
	err = writeToDisk(m.ClusterName, VsphereCredsSecret.Name, []byte(secretSpecContents), 0644)
	if err != nil {
		return err
	}
	time.Sleep(10 * time.Second)

	kubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, bootstrapKubeconfig)
	envs := map[string]string{
		"KUBECONFIG": kubeConfig,
	}
	args := []string{
		"apply",
		"--filename=" + secretSpecLocation,
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		fmt.Printf("envs: %v\n", envs)
		return err
	}

	m.EventStream <- events.Event{EventType: "progress", Event: "init capi in the bootstrap cluster"}
	nodeTemplate := strings.Split(filepath.Base(m.OVA.NodeTemplate), ".ova")[0]
	LoadBalancerTemplate := strings.Split(filepath.Base(m.OVA.LoadbalancerTemplate), ".ova")[0]
	envs = map[string]string{
		"VSPHERE_PASSWORD":           m.Password,
		"VSPHERE_USERNAME":           m.Username,
		"VSPHERE_SERVER":             m.URL,
		"VSPHERE_DATACENTER":         m.Datacenter,
		"VSPHERE_DATASTORE":          m.Datastore,
		"VSPHERE_NETWORK":            m.ManagementNetwork,
		"VSPHERE_RESOURCE_POOL":      m.ResourcePool,
		"VSPHERE_FOLDER":             m.Folder,
		"VSPHERE_TEMPLATE":           nodeTemplate,
		"VSPHERE_HAPROXY_TEMPLATE":   LoadBalancerTemplate,
		"VSPHERE_SSH_AUTHORIZED_KEY": m.SSH.AuthorizedKey,
		"KUBECONFIG":                 kubeConfig,
	}
	if m.GithubToken != "" {
		envs["GITHUB_TOKEN"] = m.GithubToken
	}
	args = []string{
		"init",
		"--infrastructure=vsphere",
	}

	err = cmds.GenericExecute(envs, string(clusterctl), args, nil)
	if err != nil {
		return err
	}

	// TODO wait for CAPv deployment in k8s to be ready
	time.Sleep(30 * time.Second)

	m.EventStream <- events.Event{EventType: "progress", Event: "writing CAPv spec file out"}
	args = []string{
		"config",
		"cluster",
		m.ClusterName,
		"--infrastructure=vsphere",
		"--kubernetes-version=" + m.KubernetesVersion,
		fmt.Sprintf("--control-plane-machine-count=%v", m.ControlPlaneCount),
		fmt.Sprintf("--worker-machine-count=%v", m.WorkerCount),
	}
	c := cmds.NewCommandLine(envs, string(clusterctl), args, nil)
	stdout, stderr, err := c.Program().Execute()
	if err != nil || string(stderr) != "" {
		return fmt.Errorf("err: %v, stderr: %v, cmd: %v %v", err, string(stderr), c.CommandName, c.Args)
	}

	err = writeToDisk(m.ClusterName, m.ClusterName+"-base"+".yaml", []byte(stdout), 0644)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
