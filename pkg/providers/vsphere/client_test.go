package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/vmware/govmomi/simulator"
)

var sim struct {
	conn   *Session
	server *simulator.Server
}

func setupSimulator() error {
	model := simulator.VPX()
	err := model.Create()
	if err != nil {
		return fmt.Errorf("unable to create simulator model, %v", err)
	}
	server := model.Service.NewServer()
	username := server.URL.User.Username()
	password, _ := server.URL.User.Password()
	url := "http://" + server.URL.Host
	sim.server = server
	conn, err := NewClient(url, username, password)
	if err != nil {
		return err
	}
	conn.Datacenter, _ = conn.GetDatacenter("/DC0")
	sim.conn = conn
	return nil
}

func shutdown() {
	sim.server.Close()
}

func TestMain(m *testing.M) {
	setupSimulator()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestGetDatacenter(t *testing.T) {
	name := "/DC0"
	obj, err := sim.conn.GetDatacenter(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if obj.InventoryPath != name {
		t.Fatalf("expected: %v, actual: %v", name, obj.InventoryPath)
	}
}

func TestGetDatastore(t *testing.T) {
	name := "/DC0/datastore/LocalDS_0"
	obj, err := sim.conn.GetDatastore(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if obj.InventoryPath != name {
		t.Fatalf("expected: %v, actual: %v", name, obj.InventoryPath)
	}
}

func TestGetNetwork(t *testing.T) {
	name := "/DC0/network/VM Network"
	obj, err := sim.conn.GetNetwork(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if obj.GetInventoryPath() != name {
		t.Fatalf("expected: %v, actual: %v", name, obj.GetInventoryPath())
	}
}

func TestGetFolder(t *testing.T) {
	name := "/DC0/vm"
	obj, err := sim.conn.GetFolder(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if obj.InventoryPath != name {
		t.Fatalf("expected: %v, actual: %v", name, obj.InventoryPath)
	}
}

func TestGetResourcePool(t *testing.T) {
	name := "/DC0/host/DC0_H0/Resources"
	obj, err := sim.conn.GetResourcePool(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if obj.InventoryPath != name {
		t.Fatalf("expected: %v, actual: %v", name, obj.InventoryPath)
	}
}

func TestGetVM(t *testing.T) {
	name := "DC0_H0_VM1"
	obj, err := sim.conn.GetVM(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if obj.InventoryPath != "/DC0/vm/DC0_H0_VM1" {
		t.Fatalf("expected: %v, actual: %v", name, obj.InventoryPath)
	}
}
