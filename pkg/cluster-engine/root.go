package clusterengine

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logLevel                        string
	cfgFile                         string
	controlPlaneMachineCount        int
	workerMachineCount              int
	controlPlaneMachineCountDefault = 1
	workerMachineCountDefault       = 2
	logLevelDefault                 = "info"
	appName                         = "cluster-engine"
)

var rootCmd = &cobra.Command{
	Use:   "cluster-engine",
	Short: "Launch Kubernetes clusters using upstream projects",
	Long:  `Launch Kubernetes clusters using upstream projects like CAPV and RKE`,
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/.%s.yaml)", appName))
	rootCmd.PersistentFlags().StringVarP(&logLevel, "loglevel", "l", logLevelDefault, "specify a log level: debug, info, warning, error")
	rootCmd.PersistentFlags().IntVarP(&controlPlaneMachineCount, "controllers", "c", controlPlaneMachineCountDefault, "the number of control plane nodes to provision")
	rootCmd.PersistentFlags().IntVarP(&workerMachineCount, "workers", "w", workerMachineCountDefault, "the number of worker nodes to provision")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logInit()
		return nil
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName("." + appName)
	}

	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err == nil {
		log.Infof("Using config file: %v", viper.ConfigFileUsed())
	} else {
		log.Fatalf("error reading config file: %v\n", err.Error())
	}
}

func logInit() {
	log.SetOutput(os.Stdout)
	switch logLevel {
	case "info":
		log.SetLevel(log.InfoLevel)
		log.SetFormatter(&log.TextFormatter{
			DisableTimestamp: true,
		})
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "debug":
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

// Execute cluster-engine
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
