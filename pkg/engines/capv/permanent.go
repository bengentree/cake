package capv

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netapp/cake/pkg/cmds"
	v1 "k8s.io/api/core/v1"
)

// CreatePermanent creates the permanent CAPv management cluster
func (m MgmtCluster) CreatePermanent() error {
	var err error
	var capiConfig string
	cf := new(ConfigFile)
	//cf.Spec = *spec
	//cf.MgmtCluster = spec.Provider.(MgmtCluster)
	cf.MgmtCluster =m

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	kubeConfig := filepath.Join(home, ConfigDir, cf.ClusterName, bootstrapKubeconfig)
	if cf.Addons.Solidfire.Enable {
		//err = injectTridentPrereqs(cf.ClusterName, cf.StorageNetwork, kubeConfig, nil)
		if err != nil {
			return err
		}
		capiConfig = filepath.Join(home, ConfigDir, cf.ClusterName, cf.ClusterName+"-final"+".yaml")
	} else {
		capiConfig = filepath.Join(home, ConfigDir, cf.ClusterName, cf.ClusterName+"-base"+".yaml")
	}

	envs := map[string]string{
		"KUBECONFIG": kubeConfig,
	}
	args := []string{
		"apply",
		"--filename=" + capiConfig,
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}

	args = []string{
		"get",
		"machine",
	}
	timeout := 15 * time.Minute
	grepString := "Running"
	grepNum := cf.ControlPlaneCount + cf.WorkerCount
	if err != nil {
		return err
	}
	err = kubeRetry(nil, args, timeout, grepString, grepNum, nil, cf.EventStream)
	if err != nil {
		return err
	}
	args = []string{
		"--namespace=default",
		"--output=json",
		"get",
		"secret",
		cf.ClusterName + "-kubeconfig",
	}
	getKubeconfig, err := kubeGet(envs, args, v1.Secret{}, nil)
	if err != nil {
		return fmt.Errorf("get secret error: %v", err.Error())
	}
	workloadClusterKubeconfig := getKubeconfig.(v1.Secret).Data["value"]
	cf.Kubeconfig = string(workloadClusterKubeconfig)
	err = writeToDisk(cf.ClusterName, "kubeconfig", workloadClusterKubeconfig, 0644)
	if err != nil {
		return err
	}

	// apply cni
	permanentKubeconfig := filepath.Join(home, ConfigDir, cf.ClusterName, "kubeconfig")
	envs = map[string]string{
		"KUBECONFIG": permanentKubeconfig,
	}
	args = []string{
		"apply",
		"--filename=https://docs.projectcalico.org/v3.12/manifests/calico.yaml",
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}

	args = []string{
		"get",
		"nodes",
	}
	grepString = "Ready"

	err = kubeRetry(envs, args, timeout, grepString, grepNum, nil, cf.EventStream)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
