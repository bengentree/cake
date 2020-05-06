package capv

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/cake/pkg/util/cmd"
)

// PivotControlPlane moves CAPv from the bootstrap cluster to the permanent management cluster
func (m MgmtCluster) PivotControlPlane() error {
	var err error
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	secretSpecLocation := filepath.Join(home, ConfigDir, m.ClusterName, vsphereCredsSecret.Name)
	permanentKubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, "kubeconfig")
	bootstrapKubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, bootstrapKubeconfig)
	envs := map[string]string{
		"KUBECONFIG": permanentKubeConfig,
	}
	args := []string{
		"apply",
		"--filename=" + secretSpecLocation,
	}
	err = cmd.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}
	args = []string{
		"create",
		"ns",
		m.Namespace,
	}
	err = cmd.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}
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
		"KUBECONFIG":                 permanentKubeConfig,
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
	err = kubeRetry(envs, args, timeout, grepString, 1, nil, m.EventStream)
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
	err = cmd.GenericExecute(envs, string(clusterctl), args, nil)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
