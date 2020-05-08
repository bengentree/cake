package cmd

import (
	"os"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type settings struct {
	config                   string
	vCenterURL               string
	vCenterUser              string
	vCenterPassword          string
	managementClusterPodCIDR string
	managementClusterCIDR    string
	disableCleanup           bool
	disablePreflight         bool
	logLevel                 string
}

var (
	cliSettings settings
	envSettings settings
	cfgFile     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cake",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("error executing command: %v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cake.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigType("yaml")
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Errorf("error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cake" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cake")
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("cake")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Errorf("failed to read in viper config: %v\n", err)
		os.Exit(1)
	}
}
