package capv

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/cake/pkg/util/cmd"
	v1 "k8s.io/api/core/v1"
)

// CreatePermanent creates the permanent CAPv management cluster
func (m MgmtCluster) CreatePermanent() error {
	var err error
	var capiConfig string

	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	kubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, bootstrapKubeconfig)
	if m.Addons.Solidfire.Enable {
		err = injectTridentPrereqs(m.ClusterName, m.StorageNetwork, kubeConfig, nil)
		if err != nil {
			return err
		}
		capiConfig = filepath.Join(home, ConfigDir, m.ClusterName, m.ClusterName+"-final"+".yaml")
	} else {
		capiConfig = filepath.Join(home, ConfigDir, m.ClusterName, m.ClusterName+"-base"+".yaml")
	}

	envs := map[string]string{
		"KUBECONFIG": kubeConfig,
	}
	args := []string{
		"apply",
		"--filename=" + capiConfig,
	}
	err = cmd.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}

	args = []string{
		"get",
		"machine",
	}
	timeout := 15 * time.Minute
	grepString := "Running"
	grepNum := m.ControlPlaneCount + m.WorkerCount
	if err != nil {
		return err
	}
	err = kubeRetry(nil, args, timeout, grepString, grepNum, nil, m.EventStream)
	if err != nil {
		return err
	}
	args = []string{
		"--namespace=default",
		"--output=json",
		"get",
		"secret",
		m.ClusterName + "-kubeconfig",
	}
	getKubeconfig, err := kubeGet(envs, args, v1.Secret{}, nil)
	if err != nil {
		return fmt.Errorf("get secret error: %v", err.Error())
	}
	workloadClusterKubeconfig := getKubeconfig.(v1.Secret).Data["value"]
	m.Kubeconfig = string(workloadClusterKubeconfig)
	err = writeToDisk(m.ClusterName, "kubeconfig", workloadClusterKubeconfig, 0644)
	if err != nil {
		return err
	}

	// apply cni
	permanentKubeconfig := filepath.Join(home, ConfigDir, m.ClusterName, "kubeconfig")
	envs = map[string]string{
		"KUBECONFIG": permanentKubeconfig,
	}
	args = []string{
		"apply",
		"--filename=https://docs.projectcalico.org/v3.12/manifests/calico.yaml",
	}
	err = cmd.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}

	args = []string{
		"get",
		"nodes",
	}
	grepString = "Ready"

	err = kubeRetry(envs, args, timeout, grepString, grepNum, nil, m.EventStream)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
