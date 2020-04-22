package capv

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/engines"
)

// PivotControlPlane moves CAPv from the bootstrap cluster to the permanent management cluster
func (m MgmtCluster) PivotControlPlane(spec *engines.Spec) error {
	var err error
	cf := new(ConfigFile)
	cf.Spec = *spec
	cf.MgmtCluster = spec.Provider.(MgmtCluster)

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, cf.ClusterName, VsphereCredsSecret.Name)
	permanentKubeConfig := filepath.Join(home, ConfigDir, cf.ClusterName, "kubeconfig")
	bootstrapKubeConfig := filepath.Join(home, ConfigDir, cf.ClusterName, bootstrapKubeconfig)
	envs := map[string]string{
		"KUBECONFIG": permanentKubeConfig,
	}
	args := []string{
		"apply",
		"--filename=" + secretSpecLocation,
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}
	args = []string{
		"create",
		"ns",
		cf.Namespace,
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}
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
		"KUBECONFIG":                 permanentKubeConfig,
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

	timeout := 5 * time.Minute
	grepString := "true"
	envs = map[string]string{
		"KUBECONFIG": bootstrapKubeConfig,
	}
	args = []string{
		"get",
		"KubeadmControlPlane",
		"--output=jsonpath='{.items[0].status.ready}'",
	}
	err = kubeRetry(envs, args, timeout, grepString, 1, nil, cf.EventStream)
	if err != nil {
		return err
	}

	envs = map[string]string{
		"KUBECONFIG": bootstrapKubeConfig,
	}
	args = []string{
		"move",
		"--to-kubeconfig=" + permanentKubeConfig,
	}
	err = cmds.GenericExecute(envs, string(clusterctl), args, nil)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
