package capv

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/cake/pkg/progress"

	"github.com/netapp/cake/pkg/util/cmd"
)

// InstallControlPlane installs CAPv CRDs into the temporary bootstrap cluster
func (m MgmtCluster) InstallControlPlane() error {
	var err error
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, m.ClusterName, vsphereCredsSecret.Name)

	secretSpecContents := fmt.Sprintf(
		vsphereCredsSecret.Contents,
		m.Username,
		m.Password,
	)
	err = writeToDisk(m.ClusterName, vsphereCredsSecret.Name, []byte(secretSpecContents), 0644)
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
	err = cmd.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		fmt.Printf("envs: %v\n", envs)
		return err
	}

	m.EventStream <- progress.Event{Type: "progress", Msg: "init capi in the bootstrap cluster"}
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
		"VSPHERE_SSH_AUTHORIZED_KEY": m.SSH.AuthorizedKeys[0],
		"KUBECONFIG":                 kubeConfig,
	}
	if m.GithubToken != "" {
		envs["GITHUB_TOKEN"] = m.GithubToken
	}
	args = []string{
		"init",
		"--infrastructure=vsphere",
	}

	err = cmd.GenericExecute(envs, string(clusterctl), args, nil)
	if err != nil {
		return err
	}

	// TODO wait for CAPv deployment in k8s to be ready
	time.Sleep(30 * time.Second)

	m.EventStream <- progress.Event{Type: "progress", Msg: "writing CAPv spec file out"}
	args = []string{
		"config",
		"cluster",
		m.ClusterName,
		"--infrastructure=vsphere",
		"--kubernetes-version=" + m.KubernetesVersion,
		fmt.Sprintf("--control-plane-machine-count=%v", m.ControlPlaneCount),
		fmt.Sprintf("--worker-machine-count=%v", m.WorkerCount),
	}
	c := cmd.NewCommandLine(envs, string(clusterctl), args, nil)
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
