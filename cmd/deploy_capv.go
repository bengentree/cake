package cmd

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var capvCmd = &cobra.Command{
	Use:   "capv",
	Short: "Deploy a K8s CAPv Management Cluster",
	Long:  `capv will create an upstream CAPv management cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		err := deployInit()
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		if localDeploy {
			runEngine("capv")
		} else {
			switch strings.ToLower(providerType) {
			default:
				runProvider("capv", "vsphere")
			}
		}
	},
}

func init() {
	deployCmd.AddCommand(capvCmd)
}
