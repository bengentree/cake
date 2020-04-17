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

// SessionManager manages vSphere client sessions
type SessionManager interface {
	GetClient() (*govmomi.Client, error)
	GetDatacenters() ([]*object.Datacenter, error)
	GetNetworks(*object.Datacenter) ([]object.NetworkReference, error)
	GetFolders() ([]*object.Folder, error)
	GetDatastores(*object.Datacenter) ([]*object.Datastore, error)
	GetResourcePools(*object.Datacenter) ([]*object.ResourcePool, error)
	GetVM(dc *object.Datacenter, name string) (*object.VirtualMachine, error)
}

type sessionManager struct {
	client   *govmomi.Client
	server   string
	username string
	password string
}

// NewManager returns a new SessionManager
func NewManager(server string, username string, password string) (SessionManager, error) {

	sm := sessionManager{
		server:   server,
		username: username,
		password: password,
	}

	// Verify connection
	_, err := sm.GetClient()
	if err != nil {
		return nil, fmt.Errorf("unable to verify connection, %v", err)
	}

	return &sm, nil
}

// GetClient returns a govmomi client with an active session
func (m *sessionManager) GetClient() (*govmomi.Client, error) {

	ctx := context.TODO()

	if m.client != nil {
		sessionActive, err := m.client.SessionManager.SessionIsActive(ctx)
		if err == nil && sessionActive {
			log.Debug("Using existing govmomi session")
			return m.client, nil
		}
	}

	log.Debug("Creating new govmomi client")

	nonAuthURL, err := url.Parse(m.server)
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

	authenticatedURL.User = url.UserPassword(m.username, m.password)

	client, err := govmomi.NewClient(ctx, nonAuthURL, true)
	if err != nil {
		return nil, fmt.Errorf("unable to create new vSphere client, %v", err)
	}

	if err = client.Login(ctx, authenticatedURL.User); err != nil {
		return nil, fmt.Errorf("unable to login to vSphere, %v", err)
	}

	m.client = client

	return m.client, nil

}

func (m *sessionManager) GetDatacenters() ([]*object.Datacenter, error) {
	var err error

	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	datacenters, err := finder.DatacenterList(context.TODO(), "*")
	if err != nil {
		return nil, err
	}
	return datacenters, err

}

func (m *sessionManager) GetDatastores(dc *object.Datacenter) ([]*object.Datastore, error) {
	var err error

	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(dc)
	datastores, err := finder.DatastoreList(context.TODO(), "*")
	if err != nil {
		return nil, err
	}
	return datastores, err

}

func (m *sessionManager) GetNetworks(dc *object.Datacenter) ([]object.NetworkReference, error) {
	var err error

	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(dc)
	networks, err := finder.NetworkList(context.TODO(), "*")
	if err != nil {
		return nil, err
	}
	return networks, err

}

func (m *sessionManager) GetFolders() ([]*object.Folder, error) {
	var err error

	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	folders, err := finder.FolderList(context.TODO(), "*")
	if err != nil {
		return nil, err
	}
	return folders, err

}

func (m *sessionManager) GetResourcePools(dc *object.Datacenter) ([]*object.ResourcePool, error) {
	var err error

	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(dc)
	folders, err := finder.ResourcePoolList(context.TODO(), "*")
	if err != nil {
		return nil, err
	}
	return folders, err

}

func (m *sessionManager) GetVM(dc *object.Datacenter, name string) (*object.VirtualMachine, error) {
	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(dc)
	vm, err := finder.VirtualMachine(context.TODO(), name)
	if err != nil {
		return nil, err
	}

	return vm, err

}
