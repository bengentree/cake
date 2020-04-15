package capv

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/netapp/capv-bootstrap/pkg/cmds"

	v1 "k8s.io/api/core/v1"
)

// CreatePermanent creates the permanent CAPv management cluster
func (m *MgmtCluster) CreatePermanent() error {
	var err error
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	capiConfig := filepath.Join(home, ConfigDir, m.ClusterName, m.ClusterName+"-capi-config"+".yaml")
	kubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, bootstrapKubeconfig)
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
	controlCount, err := strconv.Atoi(m.ControlPlaneMachineCount)
	if err != nil {
		return err
	}
	workerCount, err := strconv.Atoi(m.WorkerMachineCount)
	if err != nil {
		return err
	}
	grepNum := controlCount + workerCount
	if err != nil {
		return err
	}
	err = kubeRetry(nil, args, timeout, grepString, grepNum, nil, m.events)
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
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}

	args = []string{
		"get",
		"nodes",
	}
	grepString = "Ready"

	err = kubeRetry(envs, args, timeout, grepString, grepNum, nil, m.events)
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return err
}
