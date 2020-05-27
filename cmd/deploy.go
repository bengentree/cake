package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/engine/capv"
	"github.com/netapp/cake/pkg/engine/rke"
	"github.com/netapp/cake/pkg/engine/rkecli"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/provider"
	"github.com/netapp/cake/pkg/provider/vsphere"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	providerType            string
	localDeploy             bool
	progressEndpointEnabled bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a K8s CAPv or Rancher Management Cluster",
	Long:  `CAPv deploy will create an upstream CAPv management cluster, the Rancher/RKE option will deploy an RKE cluster with Rancher Server`,
}

func init() {
	deployCmd.PersistentFlags().BoolVar(&localDeploy, "local", false, "Run the engine locally")
	deployCmd.PersistentFlags().BoolVar(&progressEndpointEnabled, "progress", false, "Serve progress from HTTP endpoint")
	deployCmd.PersistentFlags().StringVarP(&specFile, "spec-file", "f", "", "Location of cluster-spec file corresponding to the cluster, default is at ~/.cake/<cluster name>/spec.yaml")
	deployCmd.PersistentFlags().StringVarP(&providerType, "provider", "p", "vsphere", "The provider to use for the engine [vsphere]")
	deployCmd.Flags().MarkHidden("progress")
	rootCmd.AddCommand(deployCmd)
}

// delay waits a few seconds for all events to come through before exiting
// or if the progress endpoints are enabled it waits a long time before exiting
// so that the progress endpoints stay available
func delay(start time.Time, progressEnabled bool) {
	// TODO is there a better way to wait for any final events?
	if progressEnabled {
		log.Infof("progress endpoints will be available for 24 hours from %s", time.Now())
		time.Sleep(24 * time.Hour)
	} else {
		time.Sleep(5 * time.Second)
	}
	stop := time.Now()
	log.Infof("mission duration: %v", stop.Sub(start).Round(time.Second))
}

func deployInit() error {
	var err error
	if specFile == "" {
		specFile = filepath.Join(specPath, defaultSpecFileName)
	}
	if !fileExists(specFile) {
		return fmt.Errorf("cluster spec file does not exist: %s\n", specFile)
	}
	specContents, err = ioutil.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("error reading config file (%s)", specFile)
	}
	err = progress.RunServer()
	if err != nil {
		return fmt.Errorf("error starting events server: %v", err)
	}
	return nil
}

func runProvider(engineType, providerType string) {
	var err error
	var controlPlaneCount int
	var workerCount int
	var bootstrap provider.Bootstrapper
	log.Info("Welcome to Mission Control")

	start := time.Now()
	defer delay(start, progressEndpointEnabled)
	log.DeferExitHandler(func() {
		delay(start, progressEndpointEnabled)
	})

	if engineType == "capv" {
		vsProvider := vsphere.NewMgmtBootstrapCAPV(new(vsphere.MgmtBootstrapCAPV))
		errJ := yaml.Unmarshal(specContents, &vsProvider)
		if errJ != nil {
			log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
		}
		if vsProvider.ClusterName != "" {
			clusterName = vsProvider.ClusterName
		} else {
			vsProvider.ClusterName = clusterName
		}
		controlPlaneCount = vsProvider.ControlPlaneCount
		workerCount = vsProvider.WorkerCount
		vsProvider.LogDir = specPath
		vsProvider.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
		if err != nil {
			log.Fatalf("unable to connect to events server: %v", err)
		}
		bootstrap = vsProvider
	} else if engineType == "rke" {
		vsProvider := vsphere.NewMgmtBootstrapRKE(new(vsphere.MgmtBootstrapRKE))
		errJ := yaml.Unmarshal(specContents, &vsProvider)
		if errJ != nil {
			log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
		}
		if vsProvider.ClusterName != "" {
			clusterName = vsProvider.ClusterName
		} else {
			vsProvider.ClusterName = clusterName
		}
		controlPlaneCount = vsProvider.ControlPlaneCount
		workerCount = vsProvider.WorkerCount
		vsProvider.LogDir = specPath
		vsProvider.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
		if err != nil {
			log.Fatalf("unable to connect to events server: %v", err)
		}
		bootstrap = vsProvider
	}
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneCount,
		"workerMachineCount":       workerCount,
	}).Info("Let's launch a cluster")
	status := bootstrap.Events()
	fn := func(p *progress.StatusEvent) {
		switch strings.ToLower(p.Level) {
		case "debug":
			log.WithFields(p.ToLogrusFields()).Debug("progress event")
		default:
			log.WithFields(p.ToLogrusFields()).Info("progress event")
		}
	}
	// this is an async method
	err = status.Subscribe(fn)
	if err != nil {
		log.Fatalf("error getting events: %v", err.Error())
	}
	err = provider.Run(bootstrap)
	if err != nil {
		log.Fatalf("error encountered during bootstrap: %v", err.Error())
	}
}

func runEngine(engineType string) {
	var err error
	var controlPlaneCount int
	var workerCount int
	var engineName engine.Cluster
	log.Info("Start your engines!")

	start := time.Now()
	defer delay(start, progressEndpointEnabled)
	log.DeferExitHandler(func() {
		delay(start, progressEndpointEnabled)
	})

	if engineType == "capv" {
		e := capv.NewMgmtClusterCAPV()
		errJ := yaml.Unmarshal(specContents, &e)
		if errJ != nil {
			log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
		}
		if e.ClusterName != "" {
			clusterName = e.ClusterName
		} else {
			e.ClusterName = clusterName
		}
		controlPlaneCount = e.ControlPlaneCount
		workerCount = e.WorkerCount
		e.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
		if err != nil {
			log.Fatalf("unable to connect to events server: %v", err)
		}
		e.ProgressEndpointEnabled = progressEndpointEnabled
		engineName = e
	} else if engineType == "rke" {
		// CAKE_RKE_DOCKER will deploy RKE from a docker container,
		// else RKE will be deployed using rke cli (default)
		rkeDockerEnv := os.Getenv("CAKE_RKE_DOCKER")
		if rkeDockerEnv != "" {
			e := rke.NewMgmtClusterFullConfig()
			errJ := yaml.Unmarshal(specContents, &e)
			if errJ != nil {
				log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
			}
			if e.ClusterName != "" {
				clusterName = e.ClusterName
			} else {
				e.ClusterName = clusterName
			}
			controlPlaneCount = e.ControlPlaneCount
			workerCount = e.WorkerCount
			e.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
			if err != nil {
				log.Fatalf("unable to connect to events server: %v", err)
			}
			e.ProgressEndpointEnabled = progressEndpointEnabled
			engineName = e
		} else {
			e := rkecli.NewMgmtClusterCli()
			errJ := yaml.Unmarshal(specContents, &e)
			if errJ != nil {
				log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
			}
			if e.ClusterName != "" {
				clusterName = e.ClusterName
			} else {
				e.ClusterName = clusterName
			}
			controlPlaneCount = e.ControlPlaneCount
			workerCount = e.WorkerCount
			e.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
			if err != nil {
				log.Fatalf("unable to connect to events server: %v", err)
			}
			e.ProgressEndpointEnabled = progressEndpointEnabled
			engineName = e
		}
	} else {
		log.Fatalf("[%v] engine is not implemented. please use one of [capv, rke]", engineType)
	}
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneCount,
		"workerMachineCount":       workerCount,
	}).Info("Ready, Set, Go!")
	status := engineName.Events()

	fn := func(p *progress.StatusEvent) {
		log.WithFields(p.ToLogrusFields()).Info("progress event")
	}
	err = status.Subscribe(fn)
	if err != nil {
		status.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("error subscribing to events: %v", err.Error()),
			Level: "info",
		})
		log.Fatalf(err.Error())
	}
	err = engine.Run(engineName)
	if err != nil {
		status.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("error in engine run: %v", err.Error()),
			Level: "info",
		})
		log.Error(err.Error())
	}
}
