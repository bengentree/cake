package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/cake/pkg/engines/capv"
	"github.com/netapp/cake/pkg/engines/rke"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		if deploymentType == "capv" {
			runCapvProvisioner(controlPlaneMachineCount, workerMachineCount)
		} else if deploymentType == "rke" {
			runRKEProvisioner(controlPlaneMachineCount, workerMachineCount)
		} else {
			log.Fatal("Currently only implemented deployment-type is `capv`")
		}
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

func runCapvProvisioner(controlPlaneMachineCount, workerMachineCount int) {
	// TODO dont log.Fatal, need the http endpoints to stay alive

	clusterName := "capv-mgmt-cluster"

	C := capv.MgmtCluster{}

	errJ := viper.UnmarshalExact(&C)
	if errJ != nil {
		log.Fatalf("unable to decode into struct, %v", errJ.Error())
	}

	home, errH := homedir.Dir()
	if errH != nil {
		log.Fatalf(errH.Error())
	}
	kubeconfigLocation := filepath.Join(home, capv.ConfigDir, clusterName, "kubeconfig")
	go serveProgress(C.LogFile, kubeconfigLocation)

	start := time.Now()
	log.Info("Welcome to CAPV Mission Control")

	//cpmCount := strconv.Itoa(controlPlaneMachineCount)
	//nmCount := strconv.Itoa(workerMachineCount)

	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"workerMachineCount":       workerMachineCount,
	}).Info("Let's launch a cluster")

	cluster := capv.NewMgmtCluster(C)
	exist := cluster.RequiredCommands()
	if len(exist) > 0 {
		log.Fatalf("ERROR: the following commands were not found in $PATH: [%v]\n", strings.Join(exist, ", "))
	}
	progress := cluster.Events()

	go func() {
		for {
			select {
			case event := <-progress:
				switch event.(capv.Event).EventType {
				case "checkpoint":
					// update rest api
				default:
					e := event.(capv.Event)
					log.WithFields(log.Fields{
						"eventType": e.EventType,
						"event":     e.Event,
					}).Info("event received")
				}
			}
		}
	}()

	log.Info("Creating bootstrap cluster...")
	err := cluster.CreateBootstrap()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Bootstrap cluster created.")
	responseBody.Messages = append(responseBody.Messages, "Bootstrap cluster created")

	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"WorkerMachineCount":       workerMachineCount,
	}).Info("Installing CAPv into Bootstrap cluster...")
	err = cluster.InstallControlPlane()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("CAPv installed successfully.")
	responseBody.Messages = append(responseBody.Messages, "CAPv installed successfully")

	log.Info("Creating permanent management cluster...")
	err = cluster.CreatePermanent()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Permanent management cluster created.")
	responseBody.Messages = append(responseBody.Messages, "Permanent management cluster created")

	log.Info("Moving CAPv to permanent management cluster...")
	err = cluster.PivotControlPlane()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Move to Permanent management cluster complete.")
	responseBody.Messages = append(responseBody.Messages, "Move to Permanent management cluster complete")

	log.Info("Installing Addons...")
	err = cluster.InstallAddons()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Addon installation complete.")
	responseBody.Messages = append(responseBody.Messages, "Addon installation complete")

	responseBody.Complete = true
	stop := time.Now()
	log.WithFields(log.Fields{
		"ClusterName":              clusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"WorkerMachineCount":       workerMachineCount,
		"MissionDuration":          stop.Sub(start).Round(time.Second),
	}).Info("Mission Complete")
	time.Sleep(24 * time.Hour)
}

func runRKEProvisioner(controlPlaneMachineCount, workerMachineCount int) {

	//exist := rke.RequiredCommands.Exist()
	//if exist != nil {
	//	log.Fatalf("ERROR: the following commands were not found in $PATH: [%v]\n", strings.Join(exist, ", "))
	//}

	C := rke.MgmtCluster{}

	errJ := viper.Unmarshal(&C)
	if errJ != nil {
		log.Fatalf("unable to decode into struct, %v", errJ)
	}

	go serveProgress(C.LogFile, "")

	start := time.Now()
	log.Info("Welcome to RKE Mission Control")

	//cpmCount := strconv.Itoa(controlPlaneMachineCount)
	//nmCount := strconv.Itoa(workerMachineCount)

	log.WithFields(log.Fields{
		"ClusterName":              C.ClusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"workerMachineCount":       workerMachineCount,
	}).Info("Let's launch a cluster")

	//cluster := capv.NewMgmtCluster(cpmCount, nmCount, clusterName)
	cluster := rke.NewMgmtClusterFullConfig(C)
	progress := cluster.Events()

	go func() {
		for {
			select {
			case event := <-progress:
				switch event.(capv.Event).EventType {
				case "checkpoint":
					// update rest api
				default:
					e := event.(capv.Event)
					log.WithFields(log.Fields{
						"eventType": e.EventType,
						"event":     e.Event,
					}).Info("event received")
				}
			}
		}
	}()

	log.Info("Creating bootstrap cluster...")
	err := cluster.CreateBootstrap()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Bootstrap cluster created.")
	responseBody.Messages = append(responseBody.Messages, "Bootstrap cluster created")

	log.WithFields(log.Fields{
		"ClusterName":              C.ClusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"WorkerMachineCount":       workerMachineCount,
	}).Info("Installing CAPv into Bootstrap cluster...")
	err = cluster.InstallControlPlane()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("CAPv installed successfully.")
	responseBody.Messages = append(responseBody.Messages, "CAPv installed successfully")

	log.Info("Creating permanent management cluster...")
	err = cluster.CreatePermanent()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Permanent management cluster created.")
	responseBody.Messages = append(responseBody.Messages, "Permanent management cluster created")

	log.Info("Moving CAPv to permanent management cluster...")
	err = cluster.PivotControlPlane()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Move to Permanent management cluster complete.")
	responseBody.Messages = append(responseBody.Messages, "Move to Permanent management cluster complete")
	responseBody.Complete = true

	stop := time.Now()
	log.WithFields(log.Fields{
		"ClusterName":              C.ClusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"WorkerMachineCount":       workerMachineCount,
		"MissionDuration":          stop.Sub(start).Round(time.Second),
	}).Info("Mission Complete")
	time.Sleep(24 * time.Hour)
}
