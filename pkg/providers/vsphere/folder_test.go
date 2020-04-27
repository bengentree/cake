package vsphere

import (
	"fmt"
	"path/filepath"
	"testing"
)

var createFoldersTests = []struct {
	name                  string
	folderPath            string
	expectedInventoryPath string
}{
	{"fully qualified path", "/DC0/vm/cake/vm/newFolder", "/DC0/vm/cake/vm/newFolder"},
	{"relative path", "cake/bootstrap/random", "/DC0/vm/cake/bootstrap/random"},
	{"single folder, fully qualified", "/DC0/vm/new", "/DC0/vm/new"},
	{"single folder, relative", "mgmt", "/DC0/vm/mgmt"},
	{"trailing slash, fully qualified", "/DC0/vm/withslash/", "/DC0/vm/withslash"},
	{"trailing slash, relative", "slash/", "/DC0/vm/slash"},
}

func TestCreateFolders(t *testing.T) {
	for _, tt := range createFoldersTests {
		t.Run(tt.name, func(t *testing.T) {
			lastFolder := filepath.Base(tt.folderPath)
			folder, err := sim.conn.CreateVMFolders(tt.folderPath)
			if err != nil {
				t.Fatal(err)
			}
			if folder[lastFolder].InventoryPath != tt.expectedInventoryPath {
				all, _ := sim.conn.GetAllFolders()
				fmt.Printf("all: %v\n", all)
				t.Fatalf("expected: %v, actual: %v", tt.folderPath, folder[lastFolder].InventoryPath)
			}

			lookup, err := sim.conn.GetFolder(tt.folderPath)
			if err != nil {
				t.Fatal(err)
			}
			if lookup.InventoryPath != folder[lastFolder].InventoryPath {
				all, _ := sim.conn.GetAllFolders()
				fmt.Printf("all: %v\n", all)
				t.Fatalf("expected: %v, actual: %v", folder[lastFolder].InventoryPath, lookup.InventoryPath)
			}
		})
	}
}

func TestDeleteVM(t *testing.T) {

}
