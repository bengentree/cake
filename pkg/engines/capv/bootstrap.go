package capv

import (
	"fmt"
	"github.com/netapp/cake/pkg/config/events"
	"time"

	"github.com/netapp/cake/pkg/cmds"
)

// CreateBootstrap creates the temporary CAPv bootstrap cluster
func (m MgmtCluster) CreateBootstrap() error {
	var err error
	cf := new(ConfigFile)
	//cf.Spec = *spec
	//cf.MgmtCluster = spec.Provider.(MgmtCluster)
	cf.MgmtCluster =m

	cf.EventStream <- events.Event{EventType: "progress", Event: "kind create cluster (bootstrap cluster)"}

	args := []string{
		"create",
		"cluster",
	}
	err = cmds.GenericExecute(nil, string(kind), args, nil)
	if err != nil {
		return err
	}

	cf.EventStream <- events.Event{EventType: "progress", Event: "getting and writing bootstrap cluster kubeconfig to disk"}
	args = []string{
		"get",
		"kubeconfig",
	}
	c := cmds.NewCommandLine(nil, string(kind), args, nil)
	stdout, stderr, err := c.Program().Execute()
	if err != nil || string(stderr) != "" {
		return fmt.Errorf("err: %v, stderr: %v", err, string(stderr))
	}

	err = writeToDisk(cf.ClusterName, bootstrapKubeconfig, []byte(stdout), 0644)
	if err != nil {
		return err
	}

	// TODO wait for cluster components to be running
	cf.EventStream <- events.Event{EventType: "progress", Event: "sleeping 20 seconds, need to fix this"}
	time.Sleep(20 * time.Second)

	return err
}
