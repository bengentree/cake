package capv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/netapp/capv-bootstrap/pkg/cmds"
)

const (
	kustomizeFinalSpecFile = "%s-final.yaml"
)

// kubectlKustomize runs a `kubectl kustomize` command
func kubectlKustomize(clusterName, storageNetwork, kubeconfigLocation string, ctx *context.Context) error {
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
	err = writeToDisk(clusterName, fmt.Sprintf(kustomizeFinalSpecFile, clusterName), stdout, 0644)
	if err != nil {
		return err
	}

	return err
}
