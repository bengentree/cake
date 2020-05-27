package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/netapp/cake/pkg/config/cluster"
	vsphereConfig "github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
)

// Session holds govmomi connection details
type Session struct {
	Conn         *govmomi.Client
	Datacenter   *object.Datacenter
	Datastore    *object.Datastore
	Folder       *object.Folder
	ResourcePool *object.ResourcePool
	Network      object.NetworkReference
}

// TrackedResources are vmware objects created during the bootstrap process
type TrackedResources struct {
	Folders map[string]*object.Folder
	VMs     map[string]*object.VirtualMachine
}

// GeneratedKey is the key pair generated for the run
type GeneratedKey struct {
	PrivateKey string
	PublicKey  string
}

// MgmtBootstrap spec for CAPV
type MgmtBootstrap struct {
	provider.Spec                 `yaml:",inline" json:",inline" mapstructure:",squash"`
	vsphereConfig.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
	Session                       *Session         `yaml:"-" json:"-" mapstructure:"-"`
	TrackedResources              TrackedResources `yaml:"-" json:"-" mapstructure:"-"`
	Prerequisites                 string           `yaml:"-" json:"-" mapstructure:"-"`
}

// MgmtBootstrapCAPV is the spec for bootstrapping a CAPV management cluster
type MgmtBootstrapCAPV struct {
	MgmtBootstrap      `yaml:",inline" json:",inline" mapstructure:",squash"`
	cluster.CAPIConfig `yaml:",inline" json:",inline" mapstructure:",squash"`
}

// MgmtBootstrapRKE is the spec for bootstrapping a RKE management cluster
type MgmtBootstrapRKE struct {
	MgmtBootstrap `yaml:",inline" json:",inline" mapstructure:",squash"`
	BootstrapIP   string            `yaml:"BootstrapIP" json:"bootstrapIP"`
	Nodes         map[string]string `yaml:"Nodes" json:"nodes"`
	RKEConfigPath string            `yaml:"RKEConfigPath"`
	Hostname      string            `yaml:"Hostname" json:"hostname"`
	GeneratedKey  GeneratedKey      `yaml:"-" json:"-" mapstructure:"-"`
}

// Client setups connection to remote vCenter
func (v *MgmtBootstrap) Client() error {
	c, err := NewClient(v.URL, v.Username, v.Password)
	if err != nil {
		return err
	}
	c.Datacenter, err = c.GetDatacenter(v.Datacenter)
	if err != nil {
		return err
	}
	c.Network, err = c.GetNetwork(v.ManagementNetwork)
	if err != nil {
		return err
	}
	c.Datastore, err = c.GetDatastore(v.Datastore)
	if err != nil {
		return err
	}
	c.ResourcePool, err = c.GetResourcePool(v.ResourcePool)
	if err != nil {
		return err
	}
	v.Session = c
	v.TrackedResources.Folders = make(map[string]*object.Folder)
	v.TrackedResources.VMs = make(map[string]*object.VirtualMachine)

	return nil
}

// Progress monitors the of the management cluster bootstrapping process
func (v *MgmtBootstrap) Progress() error {
	var err error
	var completedSuccessfully bool
	var respStruct progress.Status
	var progressMessages []progress.StatusEvent
	var msgLen int

	for {
		resp, err := http.Get("http://" + v.BootstrapperIP + ":8081/progress")
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		responseData, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(responseData, &respStruct)
		currentProgressMessages := respStruct.Messages
		msgLen = len(progressMessages)
		for x := msgLen; x < len(currentProgressMessages); x++ {
			v.EventStream.Publish(&respStruct.Messages[x])
			progressMessages = append(progressMessages, respStruct.Messages[x])
		}
		if respStruct.Complete {
			completedSuccessfully = respStruct.CompletedSuccessfully
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !completedSuccessfully {
		err = fmt.Errorf("didnt complete successfully")
	}

	return err
}

// Finalize handles saving deliverables and cleaning up the bootstrap VM
func (v *MgmtBootstrap) Finalize() error {
	var err error

	// remove seed.iso from all VMs
	var g errgroup.Group
	for _, elem := range v.TrackedResources.VMs {
		vm := elem
		g.Go(func() error {
			return deleteSeedISO(v, vm)
		})
	}
	if err := g.Wait(); err != nil {
		v.EventStream.Publish(&progress.StatusEvent{
			Type:  "progress",
			Msg:   fmt.Sprintf("error deleting seed iso: err: %v", err),
			Level: "info",
		})
	}

	url := fmt.Sprintf("http://%s:8081", v.BootstrapperIP)
	downloadDir := v.LogDir
	// save log file to disk
	progress.DownloadTxtFile(fmt.Sprintf("%s%s", url, progress.URILogs), path.Join(downloadDir, v.ClusterName+".log"))

	r, err := http.Get(fmt.Sprintf("%s%s", url, progress.URIDeliverable))
	if err != nil {
		return err
	}
	resp, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	var deliverables []progress.DeliverableInfo
	json.Unmarshal(resp, &deliverables)
	for _, elem := range deliverables {
		name := fmt.Sprintf("%s%s", filepath.Base(elem.Url), elem.FileExt)
		err := progress.DownloadTxtFile(fmt.Sprintf("%s%s", url, elem.Url), path.Join(downloadDir, name))
		if err != nil {
			return err
		}
	}

	v.EventStream.Publish(&progress.StatusEvent{
		Type:  "progress",
		Msg:   fmt.Sprintf("Cake deployment files saved to directory: %s/", downloadDir),
		Level: "info",
	})
	return err
}

func deleteSeedISO(v *MgmtBootstrap, elem *object.VirtualMachine) error {
	var err error
	fm := v.Session.Datastore.NewFileManager(v.Session.Datacenter, true)
	remove := fm.DeleteFile
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		dev, _ := elem.Device(ctx)
		cdDev, err := dev.FindCdrom("")
		if err != nil {
			v.EventStream.Publish(&progress.StatusEvent{
				Type:  "progress",
				Msg:   fmt.Sprintf("error finding cdrom on %v: err: %v", elem.Name(), err),
				Level: "info",
			})
		}
		elem.EditDevice(ctx, dev.EjectIso(cdDev))
		err = remove(ctx, elem.Name()+"/"+seedISOName)
		if err != nil {
			v.EventStream.Publish(&progress.StatusEvent{
				Type:  "progress",
				Msg:   fmt.Sprintf("error removing %s from %v: err: %v", seedISOName, elem.Name(), err),
				Level: "info",
			})
		}
	}(&wg)

	for i := 1; i <= 10; i++ {
		err = answerQuestion(v.Session.Conn.Client, elem, "0")
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
		if i == 10 {
			cancel()
			return fmt.Errorf("error timed out answering cdrom eject question from %v: err: %v", elem.Name(), err)
		}
	}
	wg.Wait()
	return nil
}

// Events returns the channel of progress messages
func (v *MgmtBootstrap) Events() progress.Events {
	return v.EventStream
}

func (tr *TrackedResources) addTrackedFolder(resources map[string]*object.Folder) {
	for key, value := range resources {
		tr.Folders[key] = value
	}
}

func (tr *TrackedResources) addTrackedVM(resources map[string]*object.VirtualMachine) {
	for key, value := range resources {
		tr.VMs[key] = value
	}
}

func (v *MgmtBootstrap) createFolders() error {
	desiredFolders := []string{
		fmt.Sprintf("%s/%s", baseFolder, templatesFolder),
		fmt.Sprintf("%s/%s", baseFolder, bootstrapFolder),
	}

	for _, f := range desiredFolders {
		tempFolder, err := v.Session.CreateVMFolders(f)
		if err != nil {
			return err
		}
		v.TrackedResources.addTrackedFolder(tempFolder)
	}

	if v.Folder != "" {
		fromConfig, err := v.Session.CreateVMFolders(v.Folder)
		if err != nil {
			return err
		}
		v.TrackedResources.addTrackedFolder(fromConfig)
		v.Folder = fromConfig[filepath.Base(v.Folder)].InventoryPath
		v.Session.Folder = fromConfig[filepath.Base(v.Folder)]
	} else {
		tempFolder, err := v.Session.CreateVMFolders(fmt.Sprintf("%s/%s", baseFolder, v.ClusterName))
		if err != nil {
			return err
		}
		v.TrackedResources.addTrackedFolder(tempFolder)
		v.Folder = tempFolder[v.ClusterName].InventoryPath
		v.Session.Folder = tempFolder[v.ClusterName]
	}
	return nil
}
