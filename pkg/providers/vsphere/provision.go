package vsphere

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/netapp/cake/pkg/providers"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
)

type tcp struct {
	Conn *net.Conn
}

// Provision calls the process to create the management cluster
func (v *ProviderVsphere) Provision(spec *providers.Spec) error {
	bootstrapVMIP, err := GetVMIP(v.Resources["bootstrapVM"].(*object.VirtualMachine))
	log.Infof("bootstrap VM IP: %v", bootstrapVMIP)

	/*
		filename, err := os.Executable()
		if err != nil {
			return err
		}
	*/

	// TODO get cake linux binary embedded in and use that to transfer
	// TODO wait until the uploadPort is listening instead of the 30 sec sleep
	time.Sleep(30 * time.Second)
	filename := "bin/cake-linux"
	tcpUpload, err := newTCPConn(bootstrapVMIP + ":" + uploadPort)
	if err != nil {
		return err
	}
	err = tcpUpload.uploadFile(filename)
	if err != nil {
		return err
	}

	// TODO wait until host prereqs are installed and ready
	time.Sleep(30 * time.Second)

	tcp, err := newTCPConn(bootstrapVMIP + ":" + commandPort)
	if err != nil {
		return err
	}
	cakeCmd := fmt.Sprintf("%s deploy --local --deployment-type %s > /tmp/cake.out", remoteExecutable, string(v.EngineType))
	tcp.runAsyncCommand(cakeCmd)

	return err
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

func fileDoesNotExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return true
	}
	return info.IsDir()
}

func (t *tcp) uploadFile(srcFile string) error {
	if fileDoesNotExists(srcFile) {
		return errors.New("file doesnt exist")
	}
	fi, err := os.Open(srcFile)
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
