package vsphere

import (
	"testing"
)

func TestSetupTemplate(t *testing.T) {
	t.Skip("skipping test, real vCenter needed.")
	cl, err := NewClient("172.60.0.150", "administrator@vsphere.local", "NetApp1!!")
	if err != nil {
		t.Fatalf(err.Error())
	}
	cl.Datacenter, err = cl.GetDatacenter("NetApp-HCI-Datacenter-01")
	if err != nil {
		t.Fatalf(err.Error())
	}
	cl.Network, err = cl.GetNetwork("NetApp HCI VDS 01-HCI_Internal_mNode_Network")
	if err != nil {
		t.Fatalf(err.Error())
	}
	cl.Datastore, err = cl.GetDatastore("NetApp-HCI-Datastore-02")
	if err != nil {
		t.Fatalf(err.Error())
	}
	cl.Folder, err = cl.GetFolder("k8s")
	if err != nil {
		t.Fatalf(err.Error())
	}
	cl.ResourcePool, err = cl.GetResourcePool("*/Resources")
	if err != nil {
		t.Fatalf(err.Error())
	}
	//templateName := "ubuntu-1804-kube-v1.17.3"
	templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"
	_, err = cl.deployOVATemplate(templateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestDeployOVATemplate(t *testing.T) {

	//templateName := "DC0_C0_RP0_VM1"
	templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/DC0_C0_RP0_VM1.ova"

	_, err := sim.conn.deployOVATemplate(templateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}

}
