package vsphere

import (
	"testing"
)

func TestSetupTemplate(t *testing.T) {
	vs := new(VSphere)
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
	vs.TemplateFolder = folders[0]
	vs.Datastore = datastores[0]
	vs.ManagementNetwork = networks[0]

	vs.TemplateName = "ubuntu-1804-kube-v1.17.3"
	vs.TemplateOVA = "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"
	vs.LoadbalancerTemplateName = "capv-haproxy-v0.6.0-rc.2"
	vs.LoadbalancerTemplateOVA = "https://storage.googleapis.com/capv-images/extra/haproxy/release/v0.6.0-rc.2/capv-haproxy-v0.6.0-rc.2.ova"

	vs.SessionManager = c
	_, err = DeployOVATemplate(vs, vs.TemplateName, vs.TemplateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}

}
