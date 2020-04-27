package vsphere

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"testing"

	"github.com/vmware/govmomi/simulator"
)

var sim struct {
	conn   *Session
	server *simulator.Server
}

const rootPEM = `
-----BEGIN CERTIFICATE-----
MIIEBDCCAuygAwIBAgIDAjppMA0GCSqGSIb3DQEBBQUAMEIxCzAJBgNVBAYTAlVT
MRYwFAYDVQQKEw1HZW9UcnVzdCBJbmMuMRswGQYDVQQDExJHZW9UcnVzdCBHbG9i
YWwgQ0EwHhcNMTMwNDA1MTUxNTU1WhcNMTUwNDA0MTUxNTU1WjBJMQswCQYDVQQG
EwJVUzETMBEGA1UEChMKR29vZ2xlIEluYzElMCMGA1UEAxMcR29vZ2xlIEludGVy
bmV0IEF1dGhvcml0eSBHMjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AJwqBHdc2FCROgajguDYUEi8iT/xGXAaiEZ+4I/F8YnOIe5a/mENtzJEiaB0C1NP
VaTOgmKV7utZX8bhBYASxF6UP7xbSDj0U/ck5vuR6RXEz/RTDfRK/J9U3n2+oGtv
h8DQUB8oMANA2ghzUWx//zo8pzcGjr1LEQTrfSTe5vn8MXH7lNVg8y5Kr0LSy+rE
ahqyzFPdFUuLH8gZYR/Nnag+YyuENWllhMgZxUYi+FOVvuOAShDGKuy6lyARxzmZ
EASg8GF6lSWMTlJ14rbtCMoU/M4iarNOz0YDl5cDfsCx3nuvRTPPuj5xt970JSXC
DTWJnZ37DhF5iR43xa+OcmkCAwEAAaOB+zCB+DAfBgNVHSMEGDAWgBTAephojYn7
qwVkDBF9qn1luMrMTjAdBgNVHQ4EFgQUSt0GFhu89mi1dvWBtrtiGrpagS8wEgYD
VR0TAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAQYwOgYDVR0fBDMwMTAvoC2g
K4YpaHR0cDovL2NybC5nZW90cnVzdC5jb20vY3Jscy9ndGdsb2JhbC5jcmwwPQYI
KwYBBQUHAQEEMTAvMC0GCCsGAQUFBzABhiFodHRwOi8vZ3RnbG9iYWwtb2NzcC5n
ZW90cnVzdC5jb20wFwYDVR0gBBAwDjAMBgorBgEEAdZ5AgUBMA0GCSqGSIb3DQEB
BQUAA4IBAQA21waAESetKhSbOHezI6B1WLuxfoNCunLaHtiONgaX4PCVOzf9G0JY
/iLIa704XtE7JW4S615ndkZAkNoUyHgN7ZVm2o6Gb4ChulYylYbc3GrKBIxbf/a/
zG+FA1jDaFETzf3I93k9mTXwVqO94FntT0QJo544evZG0R0SnU++0ED8Vf4GXjza
HFa9llF7b1cq26KqltyMdMKVvvBulRP/F/A8rLIQjcxz++iPAsbw+zOzlTvjwsto
WHPbqCRiOwY1nQ2pM714A5AuTHhdUDqB1O6gyHA43LL5Z/qHQF1hwFGPa4NrzQU6
yuGnBXj8ytqU0CwIPX4WecigUCAkVDNx
-----END CERTIFICATE-----`

func setupSimulator() error {
	model := simulator.VPX()
	err := model.Create()
	if err != nil {
		return fmt.Errorf("unable to create simulator model, %v", err)
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		return fmt.Errorf("failed to parse root certificate")
	}

	model.Service.TLS = &tls.Config{
		RootCAs: roots,
	}
	server := model.Service.NewServer()
	username := server.URL.User.Username()
	password, _ := server.URL.User.Password()
	url := "https://" + server.URL.Host
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
