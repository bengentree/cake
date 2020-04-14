package clusterengine

import (
	"github.com/netapp/capv-bootstrap/pkg/cluster-engine/provisioner/capv"
	"github.com/netapp/capv-bootstrap/pkg/cluster-engine/provisioner/rke"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rkeCmd = &cobra.Command{
	Use:   "rke",
	Short: "Launch Rancher Management Cluster",
	Long:  `Launch Rancher (RKE) Management Cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		runRKEProvisioner(controlPlaneMachineCount, workerMachineCount)
	},
}

func init() {
	rootCmd.AddCommand(rkeCmd)
	responseBody = new(progress)
	responseBody.Messages = []string{}
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

	go serveProgress(C.LogFile)

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
		"ClusterName":              C.ClusterName,
		"ControlPlaneMachineCount": controlPlaneMachineCount,
		"WorkerMachineCount":       workerMachineCount,
		"MissionDuration":          stop.Sub(start).Round(time.Second),
	}).Info("Mission Complete")
	time.Sleep(24 * time.Hour)
}
