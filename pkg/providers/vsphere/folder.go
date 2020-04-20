package vsphere

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

// CreateVMFolder creates all folders in a path like "one/two/three"
func (r *Resource) CreateVMFolder(folderPath string) ([]*object.Folder, error) {
	folders := strings.Split(folderPath, "/")
	var desiredFolders []*object.Folder

	base, err := r.createVMFolderRootLevel(folders[0])
	if err != nil {
		return nil, err
	}
	desiredFolders = append(desiredFolders, base)
	for f := 1; f < len(folders); f++ {
		nested, err := r.createVMFolderNestedLevel(desiredFolders[f-1], folders[f])
		if err != nil {
			return nil, err
		}
		desiredFolders = append(desiredFolders, nested)
	}

	return desiredFolders, nil
}

// createVMFolderRootLevel creates a VM folder at the root level
func (r *Resource) createVMFolderRootLevel(folderName string) (*object.Folder, error) {
	d := time.Now().Add(2 * time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	client, err := r.SessionManager.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(r.Datacenter)
	iPath := r.Datacenter.InventoryPath + "/vm/" + folderName
	desiredFolder, err := finder.Folder(ctx, iPath)
	if err != nil {
		rootFolder := r.Datacenter.InventoryPath + "/vm"
		folder, err := finder.Folder(ctx, rootFolder)
		if err != nil {
			return nil, fmt.Errorf("unable to find folder, %v", err)
		}
		desiredFolder, err = folder.CreateFolder(ctx, folderName)
		if err != nil {
			return nil, fmt.Errorf("unable to create folder, %v", err)
		}
		if desiredFolder.InventoryPath == "" {
			desiredFolder.SetInventoryPath(iPath)
		}
	}

	return desiredFolder, nil
}

// createVMFolderNestedLevel creates a VM folder inside of a root level folder
func (r *Resource) createVMFolderNestedLevel(rootFolder *object.Folder, folderName string) (*object.Folder, error) {
	d := time.Now().Add(2 * time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	client, err := r.SessionManager.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(r.Datacenter)
	desiredFolder := new(object.Folder)
	n := fmt.Sprintf("%s/%s", rootFolder.InventoryPath, folderName)
	desiredFolder, err = finder.Folder(ctx, n)
	if err != nil {
		desiredFolder, err = rootFolder.CreateFolder(ctx, folderName)
		if err != nil {
			return nil, fmt.Errorf("unable to create folder, %v", err)
		}
	}
	if desiredFolder.InventoryPath == "" && rootFolder.InventoryPath != "" {
		desiredFolder.SetInventoryPath(rootFolder.InventoryPath + "/" + folderName)
	}

	return desiredFolder, nil
}

// DeleteVMFolder removes a folder from vcenter
func (r *Resource) DeleteVMFolder(folder *object.Folder) (*object.Task, error) {
	d := time.Now().Add(2 * time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	var task *object.Task
	client, err := r.SessionManager.GetClient()
	if err != nil {
		return nil, err
	}
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(r.Datacenter)
	found, err := finder.Folder(ctx, folder.InventoryPath)
	if err == nil {
		task, err = found.Destroy(ctx)
		if err != nil {
			return nil, err
		}
	}

	return task, nil
}
