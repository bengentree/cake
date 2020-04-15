package clusterengine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/netapp/capv-bootstrap/pkg/cluster-engine/provisioner/capv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var capvCmd = &cobra.Command{
	Use:   "capv",
	Short: "Launch Cluster API Provider-vSphere (CAPV) Management Cluster",
	Long:  `Launch Cluster API Provider-vSphere (CAPV) Management Cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		runCapvProvisioner(controlPlaneMachineCount, workerMachineCount)
	},
}

var responseBody *progress

type progress struct {
	Complete bool     `json:"complete"`
	Messages []string `json:"messages"`
}

func init() {
	rootCmd.AddCommand(capvCmd)
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

	clusterName := "capv-mgmt-cluster"
	exist := capv.RequiredCommands.Exist()
	if exist != nil {
		log.Fatalf("ERROR: the following commands were not found in $PATH: [%v]\n", strings.Join(exist, ", "))
	}

	C := capv.MgmtCluster{}

	errJ := viper.Unmarshal(&C)
	if errJ != nil {
		log.Fatalf("unable to decode into struct, %v", errJ)
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

	//cluster := capv.NewMgmtCluster(cpmCount, nmCount, clusterName)
	cluster := capv.NewMgmtClusterFullConfig(C)
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
	err = cluster.InstallCAPV()
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
	err = cluster.CAPvPivot()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Move to Permanent management cluster complete.")
	responseBody.Messages = append(responseBody.Messages, "Move to Permanent management cluster complete")
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
