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

	/*
		all, err := cl.CreateVMFolder("/NetApp-HCI-Datacenter-01/vm/cake/two/three")
		if err != nil {
			t.Fatalf(err.Error())
		}
		fmt.Println(all["three"].InventoryPath)
		t.Fail()


			dcf, err := cl.Datacenter.Folders(context.TODO())
			if err != nil {
				t.Fatalf(err.Error())
			}

			fobj, err := dcf.VmFolder.CreateFolder(context.TODO(), "one/two/three")
			if err != nil {
				t.Fatalf(err.Error())
			}
			fmt.Println(fobj.InventoryPath)
			fmt.Println(fobj)



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
	*/

	//cl.ResourcePool, err = cl.GetResourcePool("*/Resources")
	//if err != nil {
	//	t.Fatalf(err.Error())
	//}

	//templateName := "ubuntu-1804-kube-v1.17.3"
	/*
		templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"
		_, err = cl.deployOVATemplate(templateOVA)
		if err != nil {
			t.Fatalf(err.Error())
		}
	*/
}

func TestDeployOVATemplate(t *testing.T) {
	var err error
	sim.conn.Network, err = sim.conn.GetNetwork("/DC0/network/VM Network")
	if err != nil {
		t.Fatal(err)
	}
	sim.conn.ResourcePool, err = sim.conn.GetResourcePool("/DC0/host/DC0_H0/Resources")
	if err != nil {
		t.Fatal(err)
	}
	sim.conn.Datastore, err = sim.conn.GetDatastore("/DC0/datastore/LocalDS_0")
	if err != nil {
		t.Fatal(err)
	}

	//templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/DC0_C0_RP0_VM1.ova"
	//templateOVA = "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"
	templateOVA := "https://communities.vmware.com/servlet/JiveServlet/downloadBody/21621-102-3-28798/Tiny Linux VM.ova"

	_, err = sim.conn.deployOVATemplate(templateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}

}
