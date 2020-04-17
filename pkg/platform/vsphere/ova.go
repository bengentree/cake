package vsphere

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// DeployOVATemplate uploads ova and makes it a template
func (r *Resource) DeployOVATemplate(templateName, templatePath string) (*object.VirtualMachine, error) {
	ctx := context.TODO()

	vSphereClient, err := r.SessionManager.GetClient()
	if err != nil {
		return nil, fmt.Errorf("unable to get vSphere client, %v", err)
	}

	finder := find.NewFinder(vSphereClient.Client, true)

	finder.SetDatacenter(r.Datacenter)

	foundTemplate, err := finder.VirtualMachine(ctx, templateName)
	if err == nil {
		return foundTemplate, nil
	}

	networks := []types.OvfNetworkMapping{
		{
			Name:    "nic0",
			Network: r.Network.Reference(),
		},
	}

	cisp := types.OvfCreateImportSpecParams{
		DiskProvisioning:   "thin",
		EntityName:         templateName,
		IpAllocationPolicy: "dhcpPolicy",
		IpProtocol:         "IPv4",
		OvfManagerCommonParams: types.OvfManagerCommonParams{
			DeploymentOption: "",
			Locale:           "US"},
		PropertyMapping: nil,
		// We need to give it a network spec, even though we don't need/want networks since we overwrite them at clone time.
		// govmomi complains that the network spec is missing otherwise (can't create the import spec).
		NetworkMapping: networks,
	}

	vm, err := createVirtualMachine(ctx, cisp, templatePath, r)
	if err != nil {
		return nil, fmt.Errorf("unable to create virtual machine, %v", err)
	}

	// Remove NICs from virtual machine before marking it as template

	if err := removeNICs(ctx, vm); err != nil {
		return nil, fmt.Errorf("unable to remove NICs from template, %v", err)
	}

	if err := vm.MarkAsTemplate(ctx); err != nil {
		return nil, fmt.Errorf("unable to mark virtual machine as a template, %v", err)
	}

	return vm, nil
}

func createVirtualMachine(ctx context.Context, cisp types.OvfCreateImportSpecParams, ovaPath string, vSphere *Resource) (*object.VirtualMachine, error) {
	vSphereClient, err := vSphere.SessionManager.GetClient()
	if err != nil {
		return nil, fmt.Errorf("unable to get vSphere client, %v", err)
	}

	ovaClient, err := newOVA(vSphereClient, ovaPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create ova client, %v", err)
	}

	spec, err := ovaClient.getImportSpec(ctx, ovaPath, vSphere.ResourcePool, vSphere.Datastore, cisp)
	if err != nil {
		return nil, fmt.Errorf("unable to create import spec for template (%s), %v", ovaPath, err)
	}
	if spec.Error != nil {
		return nil, fmt.Errorf("unable to create import spec for template, %v", spec.Error)
	}
	switch s := spec.ImportSpec.(type) {
	case *types.VirtualMachineImportSpec:
		if s.ConfigSpec.VAppConfig.GetVmConfigSpec().OvfSection != nil {
			s.ConfigSpec.VAppConfig.GetVmConfigSpec().OvfSection = nil
		}
	}

	lease, err := vSphere.ResourcePool.ImportVApp(ctx, spec.ImportSpec, vSphere.Folder, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to import the template, %v", err)
	}

	info, err := lease.Wait(ctx, spec.FileItem)
	if err != nil {
		return nil, fmt.Errorf("unable to import the template, %v", err)
	}

	u := lease.StartUpdater(ctx, info)
	defer u.Done()

	for _, i := range info.Items {
		err = ovaClient.upload(ctx, lease, i, ovaPath)
		if err != nil {
			return nil, fmt.Errorf("unable to import the template, %v", err)
		}
	}

	err = lease.Complete(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to import the template, %v", err)
	}

	moref := &info.Entity

	vm := object.NewVirtualMachine(vSphereClient.Client, *moref)

	return vm, nil
}

func openLocal(path string) (io.ReadCloser, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening local file, %w", err)
	}

	s, err := f.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("error stat on local file, %w", err)
	}

	return f, s.Size(), nil
}

func isRemotePath(path string) bool {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return true
	}
	return false
}

type tapeArchiveEntry struct {
	io.Reader
	f io.Closer
}

func (t *tapeArchiveEntry) Close() error {
	return t.f.Close()
}

type ova interface {
	upload(ctx context.Context, lease *nfc.Lease, item nfc.FileItem, ovaPath string) error
	getImportSpec(ctx context.Context, ovaPath string, resourcePool mo.Reference, datastore mo.Reference, cisp types.OvfCreateImportSpecParams) (*types.OvfCreateImportSpecResult, error)
}

type handler struct {
	client *govmomi.Client
}

// newOVA returns a new ova client
func newOVA(client *govmomi.Client, basePath string) (ova, error) {
	_, err := url.Parse(basePath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing url %s, %w", basePath, err)
	}

	return &handler{
		client: client,
	}, nil
}

func (h *handler) getImportSpec(ctx context.Context, ovaPath string, resourcePool mo.Reference, datastore mo.Reference, cisp types.OvfCreateImportSpecParams) (*types.OvfCreateImportSpecResult, error) {
	m := ovf.NewManager(h.client.Client)

	o, err := h.readOvf("*.ovf", ovaPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read OVF file from %s, %v", ovaPath, err)
	}

	return m.CreateImportSpec(ctx, string(o), resourcePool, datastore, cisp)
}

func (h *handler) upload(ctx context.Context, lease *nfc.Lease, item nfc.FileItem, ovaPath string) error {
	file := item.Path

	f, size, err := h.openOva(file, ovaPath)
	if err != nil {
		return fmt.Errorf("unable to open OVA, %v", err)
	}
	defer f.Close()

	opts := soap.Upload{
		ContentLength: size,
	}

	return lease.Upload(ctx, item, f, opts)
}

func (h *handler) readOvf(name string, ovaPath string) ([]byte, error) {
	tarReader, _, err := h.openOva(name, ovaPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open OVA file %s, %v", ovaPath, err)
	}
	defer tarReader.Close()

	return ioutil.ReadAll(tarReader)
}

func (h *handler) openOva(name string, ovaPath string) (io.ReadCloser, int64, error) {
	f, _, err := h.openFile(ovaPath)
	if err != nil {
		return nil, 0, err
	}

	tarReader := tar.NewReader(f)

	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		matched, err := path.Match(name, path.Base(h.Name))
		if err != nil {
			return nil, 0, err
		}

		if matched {
			return &tapeArchiveEntry{tarReader, f}, h.Size, nil
		}
	}

	_ = f.Close()

	return nil, 0, os.ErrNotExist
}

func (h *handler) openFile(path string) (io.ReadCloser, int64, error) {
	if isRemotePath(path) {
		return h.openRemote(path)
	}
	return openLocal(path)
}

func (h *handler) openRemote(link string) (io.ReadCloser, int64, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, 0, fmt.Errorf("Error parsing url %s, %w", link, err)
	}

	return h.client.Client.Download(context.TODO(), u, &soap.DefaultDownload)

}

func removeNICs(ctx context.Context, vm *object.VirtualMachine) error {

	log.Debugf("Removing NICs from VM %s (%s)", vm.InventoryPath, vm.Reference())

	vmProps, err := getProperties(vm)
	if err != nil {
		return pkgerrors.Wrap(err, "unable to get virtual machine properties")
	}

	virtualDeviceList := object.VirtualDeviceList(vmProps.Config.Hardware.Device)
	nicDevices := virtualDeviceList.SelectByType((*types.VirtualEthernetCard)(nil))

	if len(nicDevices) == 0 {
		// Nothing to do
		log.Debugf("No NIC devices to remove on VM %s (%s)", vm.InventoryPath, vm.Reference())
		return nil
	}

	var deviceConfigSpecs []types.BaseVirtualDeviceConfigSpec
	for _, dev := range nicDevices {
		bvEthCard, ok := dev.(types.BaseVirtualEthernetCard)
		if !ok {
			return fmt.Errorf("device is not a base virtual ethernet card")
		}
		ethCard := bvEthCard.GetVirtualEthernetCard()
		spec := &types.VirtualDeviceConfigSpec{}
		spec.Operation = types.VirtualDeviceConfigSpecOperationRemove
		spec.Device = ethCard
		deviceConfigSpecs = append(deviceConfigSpecs, spec)
	}

	vmConfigSpec := types.VirtualMachineConfigSpec{}
	vmConfigSpec.DeviceChange = deviceConfigSpecs

	task, err := vm.Reconfigure(ctx, vmConfigSpec)
	if err != nil {
		return pkgerrors.Wrapf(err, "could not reconfigure vm %s (%s)", vm.InventoryPath, vm.Reference())
	}

	if err := task.Wait(ctx); err != nil {
		return pkgerrors.Wrapf(err, "failed waiting on vm reconfigure task for %s (%s)", vm.InventoryPath, vm.Reference())
	}

	return nil
}


