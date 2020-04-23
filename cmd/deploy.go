package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/netapp/cake/pkg/config/events"
	"github.com/netapp/cake/pkg/engines/rke"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/cake/pkg/engines"
	"github.com/netapp/cake/pkg/engines/capv"
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
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a K8s CAPv or Rancher Management Cluster",
	Long:  `CAPv deploy will create an upstream CAPv management cluster, the Rancher/RKE option will deploy an RKE cluster with Rancher Server`,
	Run: func(cmd *cobra.Command, args []string) {
		runProvisioner(controlPlaneMachineCount, workerMachineCount)
	},
}

var responseBody *progress

type progress struct {
	Complete bool     `json:"complete"`
	Messages []string `json:"messages"`
}

func init() {
	deployCmd.Flags().StringVarP(&deploymentType, "deployment-type", "d", "", "The type of deployment to create (capv, rke)")
	deployCmd.MarkFlagRequired("deployment-type")
	rootCmd.AddCommand(deployCmd)

	responseBody = new(progress)
	responseBody.Messages = []string{}
}

func getResponseData() progress {
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

func runProvisioner(controlPlaneMachineCount, workerMachineCount int) {
	// TODO dont log.Fatal, need the http endpoints to stay alive

	var engine engines.Cluster
	var logFile string

	if deploymentType == "capv" {
		engine = capv.MgmtCluster{}
	} else if deploymentType == "rke" {
		result := rke.MgmtCluster{
			EventStream: make(chan events.Event),
		}
		// TODO: Can this be UnmarshalExact?
		errJ := viper.Unmarshal(&result)
		if errJ != nil {
			log.Fatalf("unable to decode into struct, %v", errJ.Error())
		}
		logFile = result.LogFile
		engine = result
	} else {
		log.Fatal("Currently only implemented deployment-type is `capv`")
	}

	clusterName := "capv-mgmt-cluster"

	home, errH := homedir.Dir()
	if errH != nil {
		log.Fatalf(errH.Error())
	}
	log.Info(logFile)
	kubeconfigLocation := filepath.Join(home, capv.ConfigDir, clusterName, "kubeconfig")
	go serveProgress(logFile, kubeconfigLocation)

	start := time.Now()
	log.Info("Welcome to Mission Control")

	//cpmCount := strconv.Itoa(controlPlaneMachineCount)
	//nmCount := strconv.Itoa(workerMachineCount)

	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"workerMachineCount":       workerMachineCount,
	}).Info("Let's launch a cluster")
	progress := engine.Events()
	go func() {
		for {
			select {
			case event := <-progress:
				switch event.EventType {
				case "checkpoint":
					// update rest api
				default:
					e := event
					log.WithFields(log.Fields{
						"eventType": e.EventType,
						"event":     e.Event,
					}).Info("event received")
				}
			}
		}
	}()

	err := func() error {
		exist := engine.RequiredCommands()
		if len(exist) > 0 {
			return fmt.Errorf("the following commands were not found in $PATH: [%v]", strings.Join(exist, ", "))
		}

		err := engine.CreateBootstrap()
		if err != nil {
			return err
		}

		err = engine.InstallControlPlane()
		if err != nil {
			return err
		}

		err = engine.CreatePermanent()
		if err != nil {
			return err
		}

		err = engine.PivotControlPlane()
		if err != nil {
			return err
		}

		err = engine.InstallAddons()
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		log.Fatal(err)
	}

	stop := time.Now()
	log.Infof("missionDuration: %v", stop.Sub(start).Round(time.Second))
	time.Sleep(24 * time.Hour)
}
