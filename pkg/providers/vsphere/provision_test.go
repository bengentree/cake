package vsphere

import (
	"testing"
)

func TestCmd(t *testing.T) {
	t.Skip("skipping test, needs a real connection.")

	//tcp, err := newTCPConn("172.60.0.85" + ":" + commandPort)
	tcp, err := newTCPConn("172.60.0.85" + ":" + "50001")
	if err != nil {
		t.Fatal(err)
	}
	//cakeCmd := fmt.Sprintf("%s deploy --local --deployment-type capv --config /root/.cake.yaml > /tmp/cake.out", remoteExecutable)
	cakeCmd := "getent passwd root > /tmp/env.log"
	tcp.runAsyncCommand(cakeCmd)
	t.Fail()
}

func TestUpload(t *testing.T) {
	t.Skip("skipping test, needs a real connection.")
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
