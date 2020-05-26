package cmd

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rkeCmd = &cobra.Command{
	Use:   "rke",
	Short: "Deploy a K8s Rancher Management Cluster",
	Long:  `rke deploy will deploy an RKE cluster with Rancher Server`,
	Run: func(cmd *cobra.Command, args []string) {
		err := deployInit()
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		if localDeploy {
			runEngine("rke")
		} else {
			switch strings.ToLower(providerType) {
			default:
				runProvider("rke", "vsphere")
			}
		}
	},
}

func init() {
	deployCmd.AddCommand(rkeCmd)
}
