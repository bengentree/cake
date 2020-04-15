package capv

import (
	"os"
	"path/filepath"
	"time"

	"github.com/netapp/capv-bootstrap/pkg/cmds"
)

// CAPvPivot moves CAPv from the bootstrap cluster to the permanent management cluster
func (m *MgmtCluster) CAPvPivot() error {
	var err error

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, m.ClusterName, VsphereCredsSecret.Name)
	permanentKubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, "kubeconfig")
	bootstrapKubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, bootstrapKubeconfig)
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
		m.Namespace,
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}

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
		"KUBECONFIG":                 permanentKubeConfig,
	}

	args = []string{
		"init",
		"--infrastructure=vsphere",
	}
	err = cmds.GenericExecute(envs, string(clusterctl), args, nil)
	if err != nil {
		return err
	}

	time.Sleep(60 * time.Second)
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
