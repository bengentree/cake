package vsphere

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/property"
	"strings"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	vim25types "github.com/vmware/govmomi/vim25/types"
)

func getProperties(vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	ctx := context.TODO()
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, fmt.Errorf("unable to get virtual machine properties, %v", err)
	}
	return &props, nil
}

func vmExists(vm *object.VirtualMachine) (bool, error) {
	ctx := context.TODO()
	foundVM, err := find.NewFinder(vm.Client(), true).VirtualMachine(ctx, vm.InventoryPath)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); ok {
			return false, nil
		}
		return false, errors.Wrapf(err, "could not determine if VM %s exists", vm.InventoryPath)
	}
	// Have to verify that this is the same VM, a new instance may have taken its place at the same path
	if vm.Reference() != foundVM.Reference() {
		//log.Debugf("VM managed object reference mismatch for %s: want %v, found %v", vm.InventoryPath, vm.Reference(), foundVM.Reference())
		return false, nil
	}
	return true, nil
}

func getTasksForVM(vm *object.VirtualMachine) ([]vim25types.TaskInfo, error) {

	ctx := context.TODO()

	moRef := vm.Reference()
	taskView, err := view.NewManager(vm.Client()).CreateTaskView(ctx, &moRef)
	if err != nil {
		return nil, errors.Wrap(err, "could not create task view")
	}

	var vmTasks []vim25types.TaskInfo
	err = taskView.Collect(ctx, func(tasks []vim25types.TaskInfo) {
		vmTasks = tasks
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not collect tasks")
	}

	return vmTasks, nil
}

func cancelRunningTasks(client *vim25.Client, taskInfos []vim25types.TaskInfo) error {

	ctx := context.TODO()

	for _, taskInfo := range taskInfos {

		if taskInfo.State != vim25types.TaskInfoStateRunning && taskInfo.State != vim25types.TaskInfoStateQueued {
			// Don't need to cancel task
			//log.Debugf("Ignoring task %s %s on entity %s, state: %s", taskInfo.Key, taskInfo.DescriptionId, taskInfo.EntityName, taskInfo.State)
			continue
		}

		//log.Debugf("Cancelling task %s %s for entity %s, state %s", taskInfo.Key, taskInfo.DescriptionId, taskInfo.EntityName, taskInfo.State)
		err := object.NewTask(client, taskInfo.Task).Cancel(ctx)
		if err != nil {
			return errors.Wrapf(err, "could not cancel task %s %s for entity %s, state %s", taskInfo.Key, taskInfo.DescriptionId, taskInfo.EntityName, taskInfo.State)
		}

	}

	return nil
}

// hasCreationTask returns true if the given task list contains an upload or a clone task that is in progress
func hasCreationTask(taskInfos []vim25types.TaskInfo) bool {
	const uploadingDescriptionID = "ImportVAppLRO"
	const cloningDescriptionID = "VirtualMachine.clone"
	for _, taskInfo := range taskInfos {
		if (taskInfo.State == vim25types.TaskInfoStateRunning || taskInfo.State == vim25types.TaskInfoStateQueued) &&
			(strings.Contains(taskInfo.DescriptionId, uploadingDescriptionID) || strings.Contains(taskInfo.DescriptionId, cloningDescriptionID)) {
			return true
		}
	}
	return false
}

func answerQuestion(client *vim25.Client, vm *object.VirtualMachine, answer string) error {
	ctx := context.TODO()
	var mvm mo.VirtualMachine

	pc := property.DefaultCollector(client)
	err := pc.RetrieveOne(ctx, vm.Reference(), []string{"runtime.question"}, &mvm)
	if err != nil {
		return fmt.Errorf("error getting vmware question: err: %v", err)
	}
	q := mvm.Runtime.Question
	if q == nil {
		return fmt.Errorf("no pending question")
	}

	return vm.Answer(ctx, q.Id, answer)
}

func addCDROMWithISOtoVM(vm *object.VirtualMachine, ctx context.Context, name string, s *Session) error {
	// add cloudinit iso to cdrom
	devices, err := vm.Device(ctx)
	if err != nil {
		return fmt.Errorf("get devices failed, %v", err)
	}
	// ide-200 is the default VirtualIDEController name
	ideCont, err := devices.FindIDEController("ide-200")
	if err != nil {
		return fmt.Errorf("error finding ide controller: err: %v", err)
	}
	cdCreated, err := devices.CreateCdrom(ideCont)
	if err != nil {
		return fmt.Errorf("error creating new cdrom: err: %v", err)
	}
	cdWithISO := devices.InsertIso(cdCreated, fmt.Sprintf("[%s] %s/%s", s.Datastore.Name(), name, seedISOName))
	if err != nil {
		return fmt.Errorf("error editing cdrom device: err: %v", err)
	}
	err = vm.AddDevice(ctx, cdWithISO)
	if err != nil {
		return fmt.Errorf("error adding device to VM: err: %v", err)
	}
	return nil
}
