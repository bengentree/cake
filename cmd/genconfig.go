package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dustinkirkland/golang-petname"
	"github.com/gookit/color"
	"github.com/netapp/cake/pkg/config/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	red         = color.New(color.FgRed)
	blue        = color.New(color.FgBlue)
	green       = color.New(color.FgGreen)
	yellow      = color.New(color.FgYellow)
	clusterSpec string
)

// genconfigCmd represents the genconfig command
var genconfigCmd = &cobra.Command{
	Use:   "genconfig",
	Short: "Create a cluster-spec via command line prompts",
	Long: `genconfig provides an option to interactively create a cluster-spec file.  By default the 
	spec file is placed in ~/.cake/<cluster-name>/spec.yml.  If no name is provides an auto generated
	name will be used.
  For example:
	    "cake genconfig -n my-demo"
  Will take interactive input from the user and create the file "~/.cake/my-demo/spe.yml"`,
	Run: func(cmd *cobra.Command, args []string) {
		runEasyConfig()
	},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	rootCmd.AddCommand(genconfigCmd)
}

func initSpecFile() {
	if clusterName == "" {
		clusterName = petname.Generate(2, "-")
		log.Infof("generated a cluster name:  %s\n", clusterName)
	}
	initSpecDir()

}
func fileExists(fn string) bool {
	_, err := os.Stat(fn)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func initSpecDir() {
	specPath = filepath.Join(cakeBaseDirPath(), clusterName)
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		log.Infof("creating config directory: %s\n", specPath)
		os.MkdirAll(specPath, 0700)
	}

}

func runEasyConfig() {

	clusterSpec = fmt.Sprintf("%s/spec.yaml", specPath)
	if fileExists(clusterSpec) {
		log.Fatal("A cluster spec file already exists for the cluster-name specified, please use another name or delete the existing spec.yml file")
	}
	log.Infof("creating spec file based on user input: %s\n", clusterSpec)
	var spec = &types.ConfigSpec{}
	spec.ClusterName = clusterName
	configure(spec)
	writeSpec(spec)
}

func cakeBaseDirPath() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s/%s", os.Getenv("USERPROFILE"), defaultSpecDir)
	}

	return fmt.Sprintf("%s/%s", os.Getenv("HOME"), defaultSpecDir)
}

func writeSpec(spec *types.ConfigSpec) {
	var configOut []byte
	var err error

	writeSpec := *spec

	if configOut, err = yaml.Marshal(writeSpec); err != nil {
		log.Fatalln(err)
	}

	err = writeFile(clusterSpec, configOut, 0644)
	if err != nil {
		log.Println(fmt.Sprintf("Unable to save cluster spec file (%s), %s", clusterSpec, err.Error()))
		return
	}
}

func writeFile(specFile string, contents []byte, permissionCode os.FileMode) error {
	if permissionCode == 0 {
		permissionCode = 0644
	}

	if err := ioutil.WriteFile(specFile, contents, permissionCode); err != nil {
		return fmt.Errorf("unable to write config file, %v", err)
	}

	return nil
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
	// fail fast if we can't connect to specified vSphere
	if err := collectVsphereInformation(spec); err != nil {
		log.Fatalln(err)
	}

	collectNetworkInformation(spec)
	collectAdditionalConfiguration(spec)
	writeSpec(spec)
}
