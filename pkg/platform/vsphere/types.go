package vsphere

import (
	"github.com/vmware/govmomi/object"
)

// Resource holds information about the vSphere environment being registered
type Resource struct {
	Infrastructure
	SessionManager SessionManager
}

// Infrastructure stores information about the underlying vSphere infrastructure
type Infrastructure struct {
	Datacenter   *object.Datacenter
	Datastore    *object.Datastore
	Folder       *object.Folder
	ResourcePool *object.ResourcePool
	Network      object.NetworkReference
}
