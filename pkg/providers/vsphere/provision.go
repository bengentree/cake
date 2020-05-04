package vsphere

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/rakyll/statik/fs"
	log "github.com/sirupsen/logrus"

	// for embedded binary
	_ "github.com/netapp/cake/pkg/statik"
	"gopkg.in/yaml.v3"
)

type tcp struct {
	Conn *net.Conn
}

// Provision calls the process to create the management cluster for CAPV
func (v *MgmtBootstrapCAPV) Provision() error {
	bootstrapVMIP, err := GetVMIP(v.TrackedResources.VMs[bootstrapVMName])
	if err != nil {
		return err
	}
	log.Infof("bootstrap VM IP: %v", bootstrapVMIP)

	configYAML, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	tcp, err := createConnectionToBootstrap(bootstrapVMIP, string(configYAML))
	if err != nil {
		return err
	}

	cakeCmd := fmt.Sprintf(runLocalCakeCmd, remoteExecutable, string(v.EngineType))
	tcp.runAsyncCommand(cakeCmd)

	return err
}

// Provision calls the process to create the management cluster for RKE
func (v *MgmtBootstrapRKE) Provision() error {
	var bootstrapVMIP string
	if v.Nodes == nil {
		v.Nodes = map[string]string{}
	}
	for name, vm := range v.TrackedResources.VMs {
		vmIP, err := GetVMIP(vm)
		if err != nil {
			return err
		}
		if name == fmt.Sprintf("%s1", rkeControlNodePrefix) {
			bootstrapVMIP = vmIP
			v.BootstrapIP = vmIP
		}
		v.Nodes[name] = vmIP
		// TODO switch log message to eents on the eventstream chan
		log.WithFields(log.Fields{
			"nodeName": name,
			"nodeIP":   vmIP,
		}).Info("vm IP received")
	}

	configYAML, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	tcp, err := createConnectionToBootstrap(bootstrapVMIP, string(configYAML))
	if err != nil {
		return err
	}
	cakeCmd := fmt.Sprintf(
		"CAKE_BOOTSTRAPIP=%s %s",
		v.BootstrapIP,
		fmt.Sprintf(
			runLocalCakeCmd,
			remoteExecutable,
			string(v.EngineType),
		),
	)
	log.Infof(cakeCmd)
	tcp.runAsyncCommand(cakeCmd)

	return nil
}

func createConnectionToBootstrap(bootstrapVMIP, configYAML string) (tcp, error) {
	var result tcp
	var err error

	// TODO wait until the uploadPort is listening instead of the 30 sec sleep
	time.Sleep(30 * time.Second)
	tcpUpload, err := newTCPConn(bootstrapVMIP + ":" + uploadPort)
	if err != nil {
		return result, err
	}
	err = tcpUpload.uploadFile(cakeLinuxBinaryPkgerLocation)
	if err != nil {
		return result, err
	}

	// upload config file
	tcpUpload, err = newTCPConn(bootstrapVMIP + ":" + uploadConfigPort)
	if err != nil {
		return result, err
	}
	err = tcpUpload.uploadFileFromString(configYAML)
	if err != nil {
		return result, err
	}
	// TODO wait until host prereqs are installed and ready
	time.Sleep(30 * time.Second)
	result, err = newTCPConn(bootstrapVMIP + ":" + commandPort)
	return result, err
}

func newTCPConn(serverAddr string) (tcp, error) {
	t := tcp{}
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return t, err
	}
	t.Conn = &conn
	return t, nil
}

func (t *tcp) runAsyncCommand(cmd string) {
	fmt.Fprintf(*t.Conn, cmd+" & disown\n")
}

func (t *tcp) runSyncCommand(cmd string) string {
	var result string
	fmt.Fprintf(*t.Conn, cmd+"\n")
	message, _ := bufio.NewReader(*t.Conn).ReadString('\n')
	result = strings.TrimSpace(message)
	return result
}

func (t *tcp) uploadFile(srcFile string) error {
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	fi, err := statikFS.Open(srcFile)
	if err != nil {
		return err
	}
	defer fi.Close()

	sizeWritten, err := io.Copy(*t.Conn, fi)
	if err != nil {
		return err
	}

	fs, err := fi.Stat()
	if err != nil {
		return err
	}
	sizeOriginal := fs.Size()

	if sizeOriginal != sizeWritten {
		return errors.New("problem with transfer")
	}

	return nil
}

func (t *tcp) uploadFileFromString(fileContents string) error {
	fi := strings.NewReader(fileContents)
	_, err := io.Copy(*t.Conn, fi)
	if err != nil {
		return err
	}
	return nil
}
