package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/netapp/cake/pkg/progress"

	"gopkg.in/yaml.v3"

	"github.com/netapp/cake/pkg/engine/rke"
	"github.com/netapp/cake/pkg/engine/rkecli"
	"github.com/netapp/cake/pkg/provider"
	"github.com/netapp/cake/pkg/provider/vsphere"

	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/engine/capv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	deploymentType          string
	localDeploy             bool
	progressEndpointEnabled bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a K8s CAPv or Rancher Management Cluster",
	Long:  `CAPv deploy will create an upstream CAPv management cluster, the Rancher/RKE option will deploy an RKE cluster with Rancher Server`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if specFile == "" {
			specFile = filepath.Join(specPath, defaultSpecFileName)
		}
		if !fileExists(specFile) {
			log.Fatalf("cluster spec file doesnt exist: %s\n", specFile)
		}
		specContents, err = ioutil.ReadFile(specFile)
		if err != nil {
			log.Fatalf("error reading config file (%s)", specFile)
		}
		err = progress.RunServer()
		if err != nil {
			log.Fatalf("error starting events server: %v", err)
		}
		if localDeploy {
			runEngine()
		} else {
			runProvider()
		}
	},
}

func init() {
	deployCmd.Flags().BoolVar(&localDeploy, "local", false, "Run the engine locally")
	deployCmd.Flags().BoolVarP(&progressEndpointEnabled, "progress", "p", false, "Serve progress from HTTP endpoint")
	deployCmd.Flags().StringVarP(&deploymentType, "deployment-type", "d", "", "The type of deployment to create (capv, rke)")
	deployCmd.PersistentFlags().StringVarP(&specFile, "spec-file", "f", "", "Location of cluster-spec file corresponding to the cluster, default is at ~/.cake/<cluster name>/spec.yaml")
	deployCmd.MarkFlagRequired("deployment-type")
	deployCmd.Flags().MarkHidden("progress")
	rootCmd.AddCommand(deployCmd)
}

func delay(start time.Time) {
	for x := 1; x <= 5; x++ {
		time.Sleep(1 * time.Second)
	}
	stop := time.Now()
	log.Infof("missionDuration: %v", stop.Sub(start).Round(time.Second))
}

func runProvider() {
	var err error
	var controlPlaneCount int
	var workerCount int
	var bootstrap provider.Bootstrapper

	// TODO better way to wait for any final events
	// wait a few seconds for all events to come through before exiting
	start := time.Now()
	defer delay(start)
	log.DeferExitHandler(func() {
		delay(start)
	})

	if deploymentType == "capv" {
		vsProvider := vsphere.NewMgmtBootstrapCAPV(new(vsphere.MgmtBootstrapCAPV))
		errJ := yaml.Unmarshal(specContents, &vsProvider)
		if errJ != nil {
			log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
		}
		clusterName = vsProvider.ClusterName
		controlPlaneCount = vsProvider.ControlPlaneCount
		workerCount = vsProvider.WorkerCount
		vsProvider.LogDir = specPath
		vsProvider.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
		if err != nil {
			log.Fatalf("unable to connect to events server: %v", err)
		}
		bootstrap = vsProvider
	} else if deploymentType == "rke" {
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

	log.Info("Welcome to Mission Control")
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneCount,
		"workerMachineCount":       workerCount,
	}).Info("Let's launch a cluster")
	status := bootstrap.Events()

	fn := func(p *progress.StatusEvent) {
		log.WithFields(p.ToLogrusFields()).Info("progress event")
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

func runEngine() {
	// TODO dont log.Fatal, need the http endpoints to stay alive

	var err error
	var controlPlaneCount int
	var workerCount int
	var logFile string
	var engineName engine.Cluster

	// TODO better way to wait for any final events
	// wait a few seconds for all events to come through before ending
	start := time.Now()
	defer delay(start)
	log.DeferExitHandler(func() {
		delay(start)
	})

	if deploymentType == "capv" {
		engine := capv.NewMgmtClusterCAPV()
		errJ := yaml.Unmarshal(specContents, &engine)
		if errJ != nil {
			log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
		}
		clusterName = engine.ClusterName
		controlPlaneCount = engine.ControlPlaneCount
		workerCount = engine.WorkerCount
		engine.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
		if err != nil {
			log.Fatalf("unable to connect to events server: %v", err)
		}
		logFile = engine.LogFile
		engine.ProgressEndpointEnabled = progressEndpointEnabled
		engineName = engine

	} else if deploymentType == "rke" {
		// CAKE_RKE_DOCKER will deploy RKE from a docker container,
		// else RKE will be deployed using rke cli (default)
		rkeDockerEnv := os.Getenv("CAKE_RKE_DOCKER")
		if rkeDockerEnv != "" {
			engine := rke.NewMgmtClusterFullConfig()
			errJ := yaml.Unmarshal(specContents, &engine)
			if errJ != nil {
				log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
			}
			if engine.ClusterName != "" {
				clusterName = engine.ClusterName
			} else {
				engine.ClusterName = clusterName
			}
			controlPlaneCount = engine.ControlPlaneCount
			workerCount = engine.WorkerCount
			engine.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
			if err != nil {
				log.Fatalf("unable to connect to events server: %v", err)
			}
			logFile = engine.LogFile
			engine.LogDir = filepath.Join(cakeBaseDirPath(), clusterName)
			engine.ProgressEndpointEnabled = progressEndpointEnabled
			engineName = engine
		} else {
			engine := rkecli.NewMgmtClusterCli()
			errJ := yaml.Unmarshal(specContents, &engine)
			if errJ != nil {
				log.Fatalf("unable to parse config (%s), %v", specFile, errJ.Error())
			}
			if engine.ClusterName != "" {
				clusterName = engine.ClusterName
			} else {
				engine.ClusterName = clusterName
			}
			controlPlaneCount = engine.ControlPlaneCount
			workerCount = engine.WorkerCount
			engine.EventStream, err = progress.NewNatsPubSub(nats.DefaultURL, clusterName)
			if err != nil {
				log.Fatalf("unable to connect to events server: %v", err)
			}
			logFile = engine.LogFile
			engine.LogDir = filepath.Join(cakeBaseDirPath(), clusterName)
			engine.ProgressEndpointEnabled = progressEndpointEnabled
			engineName = engine
		}
	} else {
		log.Fatal("Currently only implemented deployment-type is `capv`")
	}

	file, err := os.Create(logFile)
	if err != nil {
		log.Errorf("failed to create logfile: %v\n", err)
		// For now we're doing this to keep  the http endpoints?
		time.Sleep(24 * time.Hour)
	}
	defer file.Close()
	// this seems to prepend to the log file and overwrite existing data
	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)

	log.Info("Welcome to Mission Control")
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneCount,
		"workerMachineCount":       workerCount,
	}).Info("Let's launch a cluster")
	status := engineName.Events()

	fn := func(p *progress.StatusEvent) {
		log.WithFields(p.ToLogrusFields()).Info("progress event")
	}
	err = status.Subscribe(fn)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = engine.Run(engineName)
	if err != nil {
		log.Error(err.Error())
	}
}
