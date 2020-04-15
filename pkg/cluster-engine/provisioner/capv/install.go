package capv

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netapp/capv-bootstrap/pkg/cmds"
)

// InstallCAPV installs CAPv CRDs into the temporary bootstrap cluster
func (m *MgmtCluster) InstallCAPV() error {
	var err error
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, m.ClusterName, VsphereCredsSecret.Name)

	secretSpecContents := fmt.Sprintf(
		VsphereCredsSecret.Contents,
		m.VsphereUsername,
		m.VspherePassword,
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

	m.events <- Event{EventType: "progress", Event: "init capi in the bootstrap cluster"}
	envs = map[string]string{
		"VSPHERE_PASSWORD":           m.VspherePassword,
		"VSPHERE_USERNAME":           m.VsphereUsername,
		"VSPHERE_SERVER":             m.VcenterServer,
		"VSPHERE_DATACENTER":         m.Datacenter,
		"VSPHERE_DATASTORE":          m.Datastore,
		"VSPHERE_NETWORK":            m.ManagementNetwork,
		"VSPHERE_RESOURCE_POOL":      m.ResourcePool,
		"VSPHERE_FOLDER":             m.Folder,
		"VSPHERE_TEMPLATE":           m.NodeTemplate,
		"VSPHERE_HAPROXY_TEMPLATE":   m.LoadBalancerTemplate,
		"VSPHERE_SSH_AUTHORIZED_KEY": m.SSHAuthorizedKey,
		"KUBECONFIG":                 kubeConfig,
		"GITHUB_TOKEN":               "da4736762b7a0fd66db4fba10e6e001f7bb9ba65",
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

	m.events <- Event{EventType: "progress", Event: "writing CAPv spec file out"}
	args = []string{
		"config",
		"cluster",
		m.ClusterName,
		"--infrastructure=vsphere",
		"--kubernetes-version=" + m.KubernetesVersion,
		"--control-plane-machine-count=" + m.ControlPlaneMachineCount,
		"--worker-machine-count=" + m.WorkerMachineCount,
	}
	c := cmds.NewCommandLine(envs, string(clusterctl), args, nil)
	stdout, stderr, err := c.Program().Execute()
	if err != nil || string(stderr) != "" {
		return fmt.Errorf("err: %v, stderr: %v, cmd: %v %v", err, string(stderr), c.CommandName, c.Args)
	}

	err = writeToDisk(m.ClusterName, m.ClusterName+"-capi-config"+".yaml", []byte(stdout), 0644)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
