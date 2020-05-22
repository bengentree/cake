package vsphere

import (
	"encoding/json"
	"fmt"
	"github.com/netapp/cake/pkg/config/cluster"
	vsphereConfig "github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/progress"
	"github.com/netapp/cake/pkg/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"time"
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
	var progressMessages []string
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
			v.EventStream.Publish(&progress.StatusEvent{
				Type:  "progress",
				Msg:   respStruct.Messages[x],
				Level: "info",
			})
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
		Msg:   fmt.Sprintf("all files from the cluster deployment can be found here: %s/", downloadDir),
		Level: "info",
	})
	return err
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
