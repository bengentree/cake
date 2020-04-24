package vsphere

import (
	"fmt"
	"testing"
)

func TestCmd(t *testing.T) {

	tcp, err := newTCPConn("172.60.0.70" + ":" + commandPort)
	if err != nil {
		t.Fatal(err)
	}
	cakeCmd := fmt.Sprintf("%s deploy --local --deployment-type %s > /tmp/cake.out", remoteExecutable, "capv")
	tcp.runAsyncCommand(cakeCmd)
	t.Fail()
}
