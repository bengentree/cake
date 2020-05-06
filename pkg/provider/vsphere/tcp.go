package vsphere

import (
	"bufio"
	"fmt"
	"errors"
	"github.com/rakyll/statik/fs"
	"io"
	"net"
	"strings"
	"time"
	// for embedded binary
	_ "github.com/netapp/cake/pkg/util/statik"
)

type tcp struct {
	Conn *net.Conn
}

func uploadFilesToBootstrap(bootstrapVMIP, configYAML string) error {
	var err error

	// TODO wait until the uploadPort is listening instead of the 30 sec sleep
	time.Sleep(30 * time.Second)
	tcpUpload, err := newTCPConn(bootstrapVMIP + ":" + uploadPort)
	if err != nil {
		return err
	}
	err = tcpUpload.uploadFile(cakeLinuxBinaryPkgerLocation)
	if err != nil {
		return err
	}

	// upload config file
	tcpUpload, err = newTCPConn(bootstrapVMIP + ":" + uploadConfigPort)
	if err != nil {
		return err
	}
	err = tcpUpload.uploadFileFromString(configYAML)
	if err != nil {
		return err
	}
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