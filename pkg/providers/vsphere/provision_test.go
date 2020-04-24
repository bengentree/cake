package vsphere

import (
	"fmt"
	"testing"
)

func TestCmd(t *testing.T) {

	//tcp, err := newTCPConn("172.60.0.87" + ":" + commandPort)
	tcp, err := newTCPConn("172.60.0.87" + ":" + "4444")
	if err != nil {
		t.Fatal(err)
	}
	cakeCmd := fmt.Sprintf("%s deploy --local --deployment-type capv --config /root/.cake.yaml > /tmp/cake.out", remoteExecutable)
	cakeCmd = "echo ${HOME} > /tmp/env.log"
	tcp.runAsyncCommand(cakeCmd)
	t.Fail()
}

func TestUpload(t *testing.T) {
	filename := "../../../bin/cake-linux"
	tcpUpload, err := newTCPConn("172.60.0.77" + ":" + uploadPort)
	if err != nil {
		t.Fatal(err)
	}
	err = tcpUpload.uploadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
}
