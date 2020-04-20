package vsphere

import (
	"testing"
)

func TestSetupTemplate(t *testing.T) {
	vs := new(Resource)
	c, err := NewManager("https://172.60.0.150", "administrator@vsphere.local", "NetApp1!!")
	if err != nil {
		t.Fatalf(err.Error())
	}

	datacenters, _ := c.GetDatacenters()
	if err != nil {
		t.Fatalf(err.Error())
	}

	networks, _ := c.GetNetworks(datacenters[0])
	datastores, _ := c.GetDatastores(datacenters[0])
	folders, _ := c.GetFolders()
	resourcepools, _ := c.GetResourcePools(datacenters[0])

	vs.Datacenter = datacenters[0]
	vs.ResourcePool = resourcepools[0]
	vs.Folder = folders[0]
	vs.Datastore = datastores[0]
	vs.Network = networks[0]

	templateName := "ubuntu-1804-kube-v1.17.3"
	templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"

	vs.SessionManager = c
	_, err = vs.DeployOVATemplate(templateName, templateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}

}
