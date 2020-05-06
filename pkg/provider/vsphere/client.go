package vsphere

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
)

// NewClient returns a new vsphere Session
func NewClient(server string, username string, password string) (*Session, error) {
	sm := new(Session)
	ctx := context.TODO()
	log.Debug("Creating new govmomi client")
	if !strings.HasPrefix(server, "https://") && !strings.HasPrefix(server, "http://") {
		server = "https://" + server
	}
	nonAuthURL, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("unable to parse vCenter url, %v", err)
	}
	if !strings.HasSuffix(nonAuthURL.Path, "sdk") {
		nonAuthURL.Path = nonAuthURL.Path + "sdk"
	}
	authenticatedURL, err := url.Parse(nonAuthURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to parse vCenter url, %v", err)
	}
	authenticatedURL.User = url.UserPassword(username, password)
	client, err := govmomi.NewClient(ctx, nonAuthURL, true)
	if err != nil {
		return nil, fmt.Errorf("unable to create new vSphere client, %v", err)
	}
	if err = client.Login(ctx, authenticatedURL.User); err != nil {
		return nil, fmt.Errorf("unable to login to vSphere, %v", err)
	}
	sm.Conn = client

	return sm, nil
}

// GetDatacenter returns the govmomi object for a datacenter
func (s *Session) GetDatacenter(name string) (*object.Datacenter, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	datacenter, err := finder.Datacenter(context.TODO(), name)
	if err != nil {
		return nil, err
	}
	return datacenter, err
}

// GetDatastore returns the govmomi object for a datastore
func (s *Session) GetDatastore(name string) (*object.Datastore, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	datastore, err := finder.Datastore(context.TODO(), name)
	if err != nil {
		return nil, err
	}
	return datastore, err
}

// GetNetwork returns the govmomi object for a network
func (s *Session) GetNetwork(name string) (object.NetworkReference, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)

	network, err := finder.Network(context.TODO(), name)
	if err != nil {
		return nil, err
	}
	return network, err
}

// GetResourcePool returns the govmomi object for a resource pool
func (s *Session) GetResourcePool(name string) (*object.ResourcePool, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	resourcePool, err := finder.ResourcePool(context.TODO(), name)
	if err != nil {
		return nil, err
	}
	return resourcePool, err
}

// GetVM returns the govmomi object for a virtual machine
func (s *Session) GetVM(name string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	vm, err := finder.VirtualMachine(context.TODO(), name)
	if err != nil {
		return nil, err
	}
	return vm, err
}
