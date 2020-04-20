package vsphere

import (
	"github.com/vmware/govmomi/object"
)

// VSphere holds information about the vSphere environment being registered
type VSphere struct {
	Infrastructure
	ExternalResources
	SessionManager SessionManager
}

// Infrastructure stores information about the underlying vSphere infrastructure
type Infrastructure struct {
	Datacenter              *object.Datacenter
	Datastore               *object.Datastore
	Folder                  *object.Folder
	BaseFolder              *object.Folder
	TemplateFolder          *object.Folder
	BootstrapFolder         *object.Folder
	ManagementClusterFolder *object.Folder
	WorkloadClusterFolder   *object.Folder
	LoadbalancerFolder      *object.Folder
	ResourcePool            *object.ResourcePool
	ManagementNetwork       object.NetworkReference
	WorkloadNetwork         object.NetworkReference
	StorageNetwork          object.NetworkReference
}

// ExternalResources stores information external to the vSphere environment
type ExternalResources struct {
	ResourceIdentifier       string
	BootstrapName            string
	TemplateOVA              string
	TemplateName             string
	LoadbalancerTemplateOVA  string
	LoadbalancerTemplateName string
}
