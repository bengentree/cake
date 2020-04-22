package capv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config"
	"github.com/netapp/cake/pkg/engines"
)

// InstallControlPlane installs CAPv CRDs into the temporary bootstrap cluster
func (m MgmtCluster) InstallControlPlane(spec *engines.Spec) error {
	var err error
	cf := new(ConfigFile)
	cf.Spec = *spec
	cf.MgmtCluster = spec.Provider.(MgmtCluster)

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, cf.ClusterName, VsphereCredsSecret.Name)

	secretSpecContents := fmt.Sprintf(
		VsphereCredsSecret.Contents,
		cf.Username,
		cf.Password,
	)
	err = writeToDisk(cf.ClusterName, VsphereCredsSecret.Name, []byte(secretSpecContents), 0644)
	if err != nil {
		return err
	}
	time.Sleep(10 * time.Second)

	kubeConfig := filepath.Join(home, ConfigDir, cf.ClusterName, bootstrapKubeconfig)
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

	cf.EventStream <- config.Event{EventType: "progress", Event: "init capi in the bootstrap cluster"}
	nodeTemplate := strings.Split(filepath.Base(cf.OVA.NodeTemplate), ".ova")[0]
	LoadBalancerTemplate := strings.Split(filepath.Base(cf.OVA.LoadbalancerTemplate), ".ova")[0]
	envs = map[string]string{
		"VSPHERE_PASSWORD":           cf.Password,
		"VSPHERE_USERNAME":           cf.Username,
		"VSPHERE_SERVER":             cf.URL,
		"VSPHERE_DATACENTER":         cf.Datacenter,
		"VSPHERE_DATASTORE":          cf.Datastore,
		"VSPHERE_NETWORK":            cf.ManagementNetwork,
		"VSPHERE_RESOURCE_POOL":      cf.ResourcePool,
		"VSPHERE_FOLDER":             cf.Folder,
		"VSPHERE_TEMPLATE":           nodeTemplate,
		"VSPHERE_HAPROXY_TEMPLATE":   LoadBalancerTemplate,
		"VSPHERE_SSH_AUTHORIZED_KEY": cf.SSH.AuthorizedKey,
		"KUBECONFIG":                 kubeConfig,
		//"GITHUB_TOKEN":               "",
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

	cf.EventStream <- config.Event{EventType: "progress", Event: "writing CAPv spec file out"}
	args = []string{
		"config",
		"cluster",
		cf.ClusterName,
		"--infrastructure=vsphere",
		"--kubernetes-version=" + cf.KubernetesVersion,
		fmt.Sprintf("--control-plane-machine-count=%v", cf.ControlPlaneCount),
		fmt.Sprintf("--worker-machine-count=%v", cf.WorkerCount),
	}
	c := cmds.NewCommandLine(envs, string(clusterctl), args, nil)
	stdout, stderr, err := c.Program().Execute()
	if err != nil || string(stderr) != "" {
		return fmt.Errorf("err: %v, stderr: %v, cmd: %v %v", err, string(stderr), c.CommandName, c.Args)
	}

	err = writeToDisk(cf.ClusterName, cf.ClusterName+"-base"+".yaml", []byte(stdout), 0644)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
