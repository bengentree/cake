package capv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config"
	"github.com/netapp/cake/pkg/engines"
	"golang.org/x/sync/errgroup"
)

const (
	specWithTrident = "%s-final.yaml"
)

// InstallAddons installs any optional Addons to a management cluster
func (m MgmtCluster) InstallAddons(spec *engines.Spec) error {
	var g errgroup.Group
	cf := new(ConfigFile)
	cf.Spec = *spec
	cf.MgmtCluster = spec.Provider.(MgmtCluster)

	g.Go(func() error {
		if cf.Addons.Solidfire.Enable {
			return installTrident(cf)
		}
		return nil
	})
	g.Go(func() error {
		if cf.Addons.Observability.Enable {

			return installObservability(cf)
		}
		return nil
	})

	return g.Wait()
}

func installObservability(m *ConfigFile) error {
	m.EventStream <- config.Event{EventType: "progress", Event: "installing the observability addon"}
	var err error

	//targetDir, err := extractLocalArchive(m, dir)
	/*
		check if there is a default storage class, if not install longhorn
		kubectl apply -f https://raw.githubusercontent.com/longhorn/longhorn/master/deploy/longhorn.yaml
		kubectl create -f https://raw.githubusercontent.com/longhorn/longhorn/master/examples/storageclass.yaml
		kubectl patch storageclass longhorn -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
	*/

	// create alias from helm to helm3

	/*
		helm3 repo add stable https://kubernetes-charts.storage.googleapis.com
		helm3 repo add loki https://grafana.github.io/loki/charts
		kubectl create ns nks-system
		helm3 install -n nks-system prometheus stable/prometheus
		helm3 install -n nks-system loki loki/loki-stack
	*/

	/*
		cd patch
		sed -i 's/prometheus.nks-system.svc.cluster.local:8080/prometheus-server.nks-system.svc.cluster.local/g' grafana/grafana-values.yaml
		make all
	*/
	m.EventStream <- config.Event{EventType: "progress", Event: "observability addon install complete"}
	return err
}

func installTrident(m *ConfigFile) error {
	m.EventStream <- config.Event{EventType: "progress", Event: "installing the trident addon"}
	var err error
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	permanentKubeConfig := filepath.Join(home, ConfigDir, m.ClusterName, "kubeconfig")
	envs := map[string]string{
		"KUBECONFIG": permanentKubeConfig,
	}
	args := []string{"install", "--namespace=trident"}
	err = cmds.GenericExecute(envs, string(tridentctl), args, nil)
	if err != nil {
		return err
	}

	backend := fmt.Sprintf(
		elementBackendJSON.Contents,
		m.Addons.Solidfire.User,
		m.Addons.Solidfire.Password,
		m.Addons.Solidfire.MVIP,
		m.Addons.Solidfire.SVIP,
		m.ClusterName,
	)
	err = writeToDisk(m.ClusterName, elementBackendJSON.Name, []byte(backend), 0644)
	if err != nil {
		return err
	}

	fpath := filepath.Join(home, ConfigDir, m.ClusterName, elementBackendJSON.Name)
	args = []string{
		"--namespace=trident",
		"create",
		"backend",
		"--filename=" + fpath,
	}
	err = cmds.GenericExecute(envs, string(tridentctl), args, nil)
	if err != nil {
		return err
	}

	err = writeToDisk(m.ClusterName, elementStorageClass.Name, []byte(elementStorageClass.Contents), 0644)
	if err != nil {
		return err
	}
	fpath = filepath.Join(home, ConfigDir, m.ClusterName, elementStorageClass.Name)

	args = []string{
		"--namespace=default",
		"--output=json",
		"apply",
		"--filename=" + fpath,
	}
	err = cmds.GenericExecute(envs, string(kubectl), args, nil)
	if err != nil {
		return err
	}
	m.EventStream <- config.Event{EventType: "progress", Event: "trident addon install complete"}
	return err
}

// injectTridentPrereqs runs a `kubectl kustomize` command to inject trident into CAPI machines
func injectTridentPrereqs(clusterName, storageNetwork, kubeconfigLocation string, ctx *context.Context) error {
	var err error
	var envs map[string]string

	kf := fmt.Sprintf(KustomizationFile.Contents, clusterName, clusterName+"-md-0")
	err = writeToDisk(clusterName, KustomizationFile.Name, []byte(kf), 0644)
	if err != nil {
		return err
	}
	po := fmt.Sprintf(PatchFileOne.Contents, storageNetwork)
	err = writeToDisk(clusterName, PatchFileOne.Name, []byte(po), 0644)
	if err != nil {
		return err
	}
	err = writeToDisk(clusterName, PatchFileTwo.Name, []byte(PatchFileTwo.Contents), 0644)
	if err != nil {
		return err
	}

	err = writeToDisk(clusterName, PatchFileThree.Name, []byte(PatchFileThree.Contents), 0644)
	if err != nil {
		return err
	}

	if kubeconfigLocation != "" {
		envs = map[string]string{"KUBECONFIG": kubeconfigLocation}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	loc := filepath.Join(home, ConfigDir, clusterName)
	args := []string{"kustomize", loc}

	c := cmds.NewCommandLine(envs, string(kubectl), args, ctx)

	stdout, stderr, err := c.Program().Execute()
	if err != nil || string(stderr) != "" {
		return fmt.Errorf("err: %v, stderr: %v", err, string(stderr))
	}
	err = writeToDisk(clusterName, fmt.Sprintf(specWithTrident, clusterName), stdout, 0644)
	if err != nil {
		return err
	}

	return err
}
