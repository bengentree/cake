package capv

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/netapp/cake/pkg/progress"
	"time"

	"github.com/netapp/cake/pkg/util/cmd"
	v1 "k8s.io/api/core/v1"
	v3 "sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	capiv3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
)

// kubeRetry runs `kubectl` commands where the output doesnt need to be parsed or saved
func kubeRetry(envs map[string]string, args []string, timeout time.Duration, grepString string, grepCount int, ctx *context.Context, es chan progress.Event) error {
	var err error

	c := cmd.NewCommandLine(envs, string(kubectl), args, ctx)

	eventChan := make(chan string)
	done := make(chan bool, 1)

	go func() {
		for {
			select {
			case e := <-eventChan:
				es <- progress.Event{Type: "progress", Msg: e}
			case <-done:
				break
			}
		}
	}()
	ok := cmd.Retry(c, timeout, grepString, grepCount, eventChan)
	done <- ok

	if !ok {
		return fmt.Errorf("error waiting for workload cluster to be provisioned")
	}

	return err
}

// kubeGet runs a `kubectl get` command
func kubeGet(envs map[string]string, args []string, resource interface{}, ctx *context.Context) (interface{}, error) {
	var err error

	c := cmd.NewCommandLine(envs, string(kubectl), args, ctx)

	stdout, stderr, err := c.Program().Execute()
	if err != nil || string(stderr) != "" {
		return nil, fmt.Errorf("err: %v, stderr: %v", err, string(stderr))
	}

	switch resource.(type) {
	case v1.ConfigMap:
		var cMap v1.ConfigMap
		err = json.Unmarshal(stdout, &cMap)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		return cMap, nil
	case v1.Secret:
		var cMap v1.Secret
		err = json.Unmarshal(stdout, &cMap)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		return cMap, nil
	case v3.HAProxyLoadBalancer:
		var cList v1.List

		err = json.Unmarshal(stdout, &cList)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		var cMap v3.HAProxyLoadBalancer
		err = json.Unmarshal(cList.Items[0].Raw, &cMap)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		return cMap, nil
	case v3.VSphereMachineTemplate:
		var cList v1.List

		err = json.Unmarshal(stdout, &cList)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		var cMap v3.VSphereMachineTemplate
		err = json.Unmarshal(cList.Items[0].Raw, &cMap)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		return cMap, nil
	case capiv3.KubeadmControlPlane:
		var cList v1.List

		err = json.Unmarshal(stdout, &cList)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		var cMap capiv3.KubeadmControlPlane
		err = json.Unmarshal(cList.Items[0].Raw, &cMap)
		if err != nil {
			return nil, fmt.Errorf("error with unmarshal: %v", err.Error())
		}
		return cMap, nil
	}

	return nil, fmt.Errorf("unknown err")
}
