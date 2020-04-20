/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/gookit/color"
	"github.com/netapp/cake/pkg/config/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	red    = color.New(color.FgRed)
	blue   = color.New(color.FgBlue)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
)

var configFile string

// genconfigCmd represents the genconfig command
var genconfigCmd = &cobra.Command{
	Use:   "genconfig",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		runEasyConfig()
	},
}

func init() {
	rootCmd.AddCommand(genconfigCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// genconfigCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// genconfigCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runEasyConfig() {
	var spec = &types.ConfigSpec{}
	configure(spec)
	writeConfig(spec)
}

func cakeBaseDirPath() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s/.cake", os.Getenv("USERPROFILE"))
	}

	return fmt.Sprintf("%s/.cake", os.Getenv("HOME"))
}

func writeConfig(spec *types.ConfigSpec) {
	var configOut []byte
	var err error

	writeSpec := *spec

	if configOut, err = yaml.Marshal(writeSpec); err != nil {
		log.Fatalln(err)
	}

	configFile := getConfigFile(spec.RegionName)

	err = writeFile(configFile, configOut, 0644)
	if err != nil {
		log.Println(fmt.Sprintf("Unable to save region config for later use, %s", err.Error()))
		return
	}
}

func writeFile(configFile string, contents []byte, permissionCode os.FileMode) error {
	if permissionCode == 0 {
		permissionCode = 0644
	}

	if err := ioutil.WriteFile(configFile, contents, permissionCode); err != nil {
		return fmt.Errorf("unable to write config file, %v", err)
	}

	return nil
}

func getConfigFile(regionName string) string {
	if configFile != "" {
		return configFile
	}

	if err := createConfigDirectory(regionName); err != nil {
		log.Fatalf("Unable to create directory, %v", err)
	}

	basePath := cakeBaseDirPath()

	return fmt.Sprintf("%s/%s/config.yaml", basePath, regionName)
}

func createConfigDirectory(directoryName string) error {
	basePath := cakeBaseDirPath()
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		err = os.Mkdir(basePath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	fullPath := fmt.Sprintf("%s/%s", basePath, directoryName)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		err = os.Mkdir(fullPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func configure(spec *types.ConfigSpec) {
	fmt.Println(fmt.Sprintf("%s DHCP will be replaced in future versions with internal IP services or a third party IPAM provider\n", yellow.Render("Note:")))

	// fail fast if we can't connect to specified vSphere
	if err := collectVsphereInformation(spec); err != nil {
		log.Fatalln(err)
	}

	collectNetworkInformation(spec)
	collectAdditionalConfiguration(spec)
	writeConfig(spec)
}
