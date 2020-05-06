package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/netapp/cake/pkg/engine/rke"
	"github.com/netapp/cake/pkg/engine/rkecli"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/provider"
	"github.com/netapp/cake/pkg/provider/vsphere"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/engine/capv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logLevel string
	//cfgFile                         string
	controlPlaneMachineCount        int
	workerMachineCount              int
	controlPlaneMachineCountDefault = 1
	workerMachineCountDefault       = 2
	logLevelDefault                 = "info"
	appName                         = "cluster-engine"
	deploymentType                  string
	localDeploy                     bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a K8s CAPv or Rancher Management Cluster",
	Long:  `CAPv deploy will create an upstream CAPv management cluster, the Rancher/RKE option will deploy an RKE cluster with Rancher Server`,
	Run: func(cmd *cobra.Command, args []string) {
		if localDeploy {
			runEngine()
		} else {
			runProvider()
		}
	},
}

var responseBody *status

type status struct {
	Complete bool     `json:"complete"`
	Messages []string `json:"messages"`
}

func init() {
	deployCmd.Flags().BoolVarP(&localDeploy, "local", "l", false, "Run the engine locally")
	deployCmd.Flags().StringVarP(&deploymentType, "deployment-type", "d", "", "The type of deployment to create (capv, rke)")
	deployCmd.MarkFlagRequired("deployment-type")
	rootCmd.AddCommand(deployCmd)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logInit()
		return nil
	}

	responseBody = new(status)
	responseBody.Messages = []string{}
}

func logInit() {
	log.SetOutput(os.Stdout)
}

func getResponseData() status {
	return *responseBody
}

func serveProgress(logfile string, kubeconfig string) {
	http.HandleFunc("/progress", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(responseBody)
	})
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		logs, _ := ioutil.ReadFile(logfile)
		fmt.Fprintf(w, string(logs))
	})
	http.HandleFunc("/kubeconfig", func(w http.ResponseWriter, r *http.Request) {
		kconfig, _ := ioutil.ReadFile(kubeconfig)
		if len(kconfig) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(w, string(kconfig))
	})
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func runProvider() {
	var clusterName string
	var controlPlaneCount int
	var workerCount int
	var bootstrap provider.Bootstrapper
	if deploymentType == "capv" {
		vsProvider := new(vsphere.MgmtBootstrapCAPV)
		errJ := viper.Unmarshal(&vsProvider)
		if errJ != nil {
			log.Fatalf("unable to decode into struct, %v", errJ.Error())
		}
		clusterName = vsProvider.ClusterName
		controlPlaneCount = vsProvider.ControlPlaneCount
		workerCount = vsProvider.WorkerCount
		vsProvider.EventStream = make(chan progress.Event)
		bootstrap = vsProvider
	} else if deploymentType == "rke" {
		vsProvider := new(vsphere.MgmtBootstrapRKE)
		errJ := viper.Unmarshal(&vsProvider)
		if errJ != nil {
			log.Fatalf("unable to decode into struct, %v", errJ.Error())
		}
		clusterName = vsProvider.ClusterName
		controlPlaneCount = vsProvider.ControlPlaneCount
		workerCount = vsProvider.WorkerCount
		vsProvider.EventStream = make(chan progress.Event)
		bootstrap = vsProvider
	}

	start := time.Now()
	log.Info("Welcome to Mission Control")
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneCount,
		"workerMachineCount":       workerCount,
	}).Info("Let's launch a cluster")
	cakeProgress := bootstrap.Events()
	go func() {
		for {
			select {
			case evnt := <-cakeProgress:
				switch evnt.Type {
				case "checkpoint":
					// update rest api
				default:
					e := evnt
					log.WithFields(log.Fields{
						"eventType": e.Type,
						"progress":     e.Msg,
					}).Info("progress received")
				}
			}
		}
	}()

	err := provider.Run(bootstrap)
	if err != nil {
		log.Fatal(err.Error())
	}
	stop := time.Now()
	log.Infof("missionDuration: %v", stop.Sub(start).Round(time.Second))
}

func runEngine() {
	// TODO dont log.Fatal, need the http endpoints to stay alive

	var clusterName string
	var controlPlaneCount int
	var workerCount int
	var logFile string
	var engineName engine.Cluster

	if deploymentType == "capv" {
		engine := capv.MgmtCluster{}
		errJ := viper.Unmarshal(&engine)
		if errJ != nil {
			log.Fatalf("unable to decode into struct, %v", errJ.Error())
		}
		clusterName = engine.ClusterName
		controlPlaneCount = engine.ControlPlaneCount
		workerCount = engine.WorkerCount
		logFile = engine.LogFile
		engine.EventStream = make(chan progress.Event)
		engineName = engine

	} else if deploymentType == "rke" {
		// CAKE_RKE_DOCKER will deploy RKE from a docker container,
		// else RKE will be deployed using rke cli (default)
		rkeDockerEnv := os.Getenv("CAKE_RKE_DOCKER")
		if rkeDockerEnv != "" {
			engine := rke.NewMgmtClusterFullConfig()
			errJ := viper.Unmarshal(&engine)
			if errJ != nil {
				log.Fatalf("unable to decode into struct, %v", errJ.Error())
			}
			clusterName = engine.ClusterName
			controlPlaneCount = engine.ControlPlaneCount
			workerCount = engine.WorkerCount
			logFile = engine.LogFile
			engine.EventStream = make(chan progress.Event)
			engineName = engine
		} else {
			engine := rkecli.NewMgmtClusterCli()
			errJ := viper.Unmarshal(&engine)
			if errJ != nil {
				log.Fatalf("unable to decode into struct, %v", errJ.Error())
			}
			clusterName = engine.ClusterName
			controlPlaneCount = engine.ControlPlaneCount
			workerCount = engine.WorkerCount
			logFile = engine.LogFile
			engine.EventStream = make(chan progress.Event)
			engineName = engine
		}
	} else {
		log.Fatal("Currently only implemented deployment-type is `capv`")
	}

	file, err := os.Create(logFile)
	if err != nil {
		log.Error(err)
		time.Sleep(24 * time.Hour)
	}
	defer file.Close()
	// this seems to prepend to the log file and overwrite existing data
	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)

	home, errH := homedir.Dir()
	if errH != nil {
		log.Fatalf(errH.Error())
	}
	kubeconfigLocation := filepath.Join(home, capv.ConfigDir, clusterName, "kubeconfig")
	go serveProgress(logFile, kubeconfigLocation)

	start := time.Now()
	log.Info("Welcome to Mission Control")
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneCount,
		"workerMachineCount":       workerCount,
	}).Info("Let's launch a cluster")
	progress := engineName.Events()

	go func() {
		for {
			select {
			case evnt := <-progress:
				switch evnt.Type {
				case "checkpoint":
					// update rest api
				default:
					e := evnt
					log.WithFields(log.Fields{
						"eventType": e.Type,
						"progress":     e.Msg,
					}).Info("progress received")
				}
			}
		}
	}()

	err = engine.Run(engineName)
	if err != nil {
		log.Error(err.Error())
	}
	stop := time.Now()
	log.Infof("missionDuration: %v", stop.Sub(start).Round(time.Second))
	time.Sleep(24 * time.Hour)
}
