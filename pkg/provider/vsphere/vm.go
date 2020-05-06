package vsphere

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/netapp/cake/pkg/provider/vsphere/cloudinit"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/sync/errgroup"
)

type cloneSpec struct {
	template   *object.VirtualMachine
	name       string
	bootScript string
	publicKey  []string
	osUser     string
}

// CloneTemplates clones multiple VMs asynchronously
func (s *Session) CloneTemplates(clonesSpec ...cloneSpec) (map[string]*object.VirtualMachine, error) {
	numVMs := len(clonesSpec)
	result := make(map[string]*object.VirtualMachine, numVMs)

	var g errgroup.Group

	batch := 3

	for i := 0; i < numVMs; i += batch {
		j := i + batch
		if j > numVMs {
			j = numVMs
		}

		for _, vm := range clonesSpec[i:j] {
			vm := vm
			g.Go(func() error {
				r, err := s.CloneTemplate(vm.template, vm.name, vm.bootScript, vm.publicKey, vm.osUser)
				if err != nil {
					return err
				}
				result[vm.name] = r
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return result, err
		}
	}

	return result, nil

}

// CloneTemplate creates a VM from a template
func (s *Session) CloneTemplate(template *object.VirtualMachine, name string, bootScript string, publicKeys []string, osUser string) (*object.VirtualMachine, error) {

	// give whole clone process a 10 minute timeout
	d := time.Now().Add(10 * time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	cloudinitUserDataConfig, err := cloudinit.GenerateUserData(bootScript, publicKeys, osUser)
	if err != nil {
		return nil, fmt.Errorf("unable to generate user data, %v", err)
	}

	spec := types.VirtualMachineCloneSpec{}
	spec.Config = &types.VirtualMachineConfigSpec{}
	spec.Config.ExtraConfig = cloudinitUserDataConfig
	/*
		TODO make cpu and memory configurable
		spec.Config.NumCPUs = int32
		spec.Config.MemoryMB = int64
	*/

	spec.Location.Datastore = types.NewReference(s.Datastore.Reference())
	spec.Location.Pool = types.NewReference(s.ResourcePool.Reference())
	spec.PowerOn = false // Do not turn machine on until after metadata reconfiguration
	spec.Location.DiskMoveType = string(types.VirtualMachineRelocateDiskMoveOptionsMoveAllDiskBackingsAndConsolidate)

	vmProps, err := getProperties(template)
	if err != nil {
		return nil, fmt.Errorf("unable to get virtual machine properties, %v", err)
	}

	l := object.VirtualDeviceList(vmProps.Config.Hardware.Device)

	deviceSpecs := []types.BaseVirtualDeviceConfigSpec{}

	nics := l.SelectByType((*types.VirtualEthernetCard)(nil))

	// Remove any existing nics on the source vm
	for _, dev := range nics {
		nic := dev.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
		nicspec := &types.VirtualDeviceConfigSpec{}
		nicspec.Operation = types.VirtualDeviceConfigSpecOperationRemove
		nicspec.Device = nic
		deviceSpecs = append(deviceSpecs, nicspec)
	}

	nic := types.VirtualVmxnet3{}
	nic.Backing, err = s.Network.EthernetCardBackingInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get information on NIC, %v", err)
	}
	nicspec := &types.VirtualDeviceConfigSpec{}
	nicspec.Operation = types.VirtualDeviceConfigSpecOperationAdd
	nicspec.Device = &nic
	deviceSpecs = append(deviceSpecs, nicspec)

	spec.Config.DeviceChange = deviceSpecs

	log.Debugf("cloning %s with spec: %+v", name, spec)
	task, err := template.Clone(ctx, s.Folder, name, spec)
	if err != nil {
		return nil, fmt.Errorf("unable to clone template, %v", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("clone task failed, %v", err)
	}

	vm, err := s.GetVM(name)
	if err != nil {
		return nil, fmt.Errorf("unable to find virtual machine, %v", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("reconfigure task failed, %v", err)
	}

	log.Debugf("powering on %s", name)
	task, err = vm.PowerOn(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to power on VM, %v", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("power on task failed, %v", err)
	}

	return vm, nil
}

// DeleteVM deletes a VM
func DeleteVM(vm *object.VirtualMachine) error {
	ctx := context.TODO()

	// Verify that the VM exists
	exists, err := vmExists(vm)
	if err != nil {
		return err
	}
	if !exists {
		log.Debugf("VM %s not found, will not delete", vm.InventoryPath)
		return nil
	}

	// Check for tasks
	vmTasks, err := getTasksForVM(vm)
	if err != nil {
		return fmt.Errorf("could not get vm tasks, %v", err)
	}

	// Cancel running tasks, if any
	if len(vmTasks) > 0 {
		log.Debugf("Found %d tasks for VM %s", len(vmTasks), vm.InventoryPath)
		err = cancelRunningTasks(vm.Client(), vmTasks)
		if err != nil {
			return fmt.Errorf("could not cancel tasks for vm %s, %v", vm.InventoryPath, err)
		}
		// If the VM was uploading/cloning, and we just cancelled the task, the VM will go away
		if hasCreationTask(vmTasks) {
			// Have to wait for the VM to disappear before continuing, best effort only
			// Note that there does not seem to be an API to wait for the cancel task to finish and VM to disappear
			maxTries := 10
			for tryCount := 0; tryCount < maxTries; tryCount++ {
				log.Debugf("Checking if VM %s exists after cancelling creation task", vm.InventoryPath)
				exists, err = vmExists(vm)
				if err != nil {
					log.Errorf("Could not check if VM %s exists", vm.InventoryPath)
				}
				if err == nil && !exists {
					// VM has gone away
					log.Debugf("VM %s deleted after cancelling creation task", vm.InventoryPath)
					return nil
				}
				time.Sleep(2 * time.Second)
			}
			log.Debugf("Wait for VM %s to be deleted after cancelling creation task timed out", vm.InventoryPath)
		}
	}

	// Double check that VM is there
	exists, err = vmExists(vm)
	if err != nil {
		return err
	}
	if !exists {
		log.Debugf("VM %s not found, will not delete", vm.InventoryPath)
		return nil
	}

	powerState, err := vm.PowerState(ctx)
	if err != nil {
		return fmt.Errorf("unable to determine virtual machine power state, %v", err)
	}

	if powerState != types.VirtualMachinePowerStatePoweredOff {
		log.Debugf("Powering off VM %s", vm.InventoryPath)
		task, err := vm.PowerOff(ctx)
		if err != nil {
			return fmt.Errorf("unable to power off virtual machine %s, %v", vm.InventoryPath, err)
		}

		if err := task.Wait(ctx); err != nil {
			return fmt.Errorf("power off task for vm %s failed, %v", vm.InventoryPath, err)
		}
	}

	log.Debugf("Deleting VM %s", vm.InventoryPath)
	task, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("unable to destroy virtual machine %s, %v", vm.InventoryPath, err)
	}

	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("destroy task for vm %s failed, %v", vm.InventoryPath, err)
	}

	return nil
}

// GetVMIP returns the first IPv4 IP on the first NIC
func GetVMIP(vm *object.VirtualMachine) (string, error) {
	const (
		timeout = 10 * time.Minute
		nic     = "ethernet-0"
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Debugf("Waiting for vm to receive ip on interface %s, timeout %s", nic, timeout)
	macToIPMap, err := vm.WaitForNetIP(ctx, true, nic)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timed out after %s while waiting for ip", timeout)
		}
		return "", fmt.Errorf("failed waiting for IP, %v", err)
	}

	for _, ips := range macToIPMap {
		for _, ip := range ips {
			return ip, nil
		}
	}

	return "", errors.New("could not find IP address of VM Network NIC")
}
