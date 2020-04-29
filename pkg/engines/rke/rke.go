package rke

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/netapp/cake/pkg/cmds"
	"github.com/netapp/cake/pkg/config/events"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engines"
	"github.com/rancher/norman/clientbase"
	rTypes "github.com/rancher/norman/types"
	v3 "github.com/rancher/types/client/management/v3"
	v3public "github.com/rancher/types/client/management/v3public"
	v3project "github.com/rancher/types/client/project/v3"
	log "github.com/sirupsen/logrus"
)

type requiredCmd string

const (
	docker requiredCmd = "docker"
)

// RequiredCommands for capv provisioner
var RequiredCommands = cmds.ProvisionerCommands{Name: "required CAPV bootstrap commands"}

func init() {
	d := cmds.NewCommandLine(nil, string(docker), nil, nil)
	RequiredCommands.AddCommand(d.CommandName, d)
}

// NewMgmtClusterFullConfig creates a new cluster interface with a full config from the client
func NewMgmtClusterFullConfig() *MgmtCluster {
	mc := &MgmtCluster{}
	mc.EventStream = make(chan events.Event)
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	mc.dockerCli = new(dockerCli)
	mc.osCli = new(osCli)
	return mc
}

// MgmtCluster spec for RKE
type MgmtCluster struct {
	EventStream             chan events.Event
	engines.MgmtCluster     `yaml:",inline" mapstructure:",squash"`
	vsphere.ProviderVsphere `yaml:",inline" mapstructure:",squash"`
	token                   string
	clusterURL              string
	rancherClient           *v3.Client
	BootstrapIP             string `yaml:"BootstrapIP"`
	dockerCli               dockerCmds
	osCli                   genericCmds
}

type dockerCmds interface {
	NewEnvClient() (*client.Client, error)
	ContainerCreate(ctx context.Context, cli *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, cli *client.Client, containerID string, options types.ContainerStartOptions) error
}

type dockerCli struct{}

func (dockerCli) NewEnvClient() (*client.Client, error) {
	return client.NewEnvClient()
}

func (dockerCli) ContainerCreate(ctx context.Context, cli *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	return cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
}

func (dockerCli) ContainerStart(ctx context.Context, cli *client.Client, containerID string, options types.ContainerStartOptions) error {
	return cli.ContainerStart(ctx, containerID, options)
}

type genericCmds interface {
	GenericExecute(envs map[string]string, name string, args []string, ctx *context.Context) error
}

type osCli struct{}

func (osCli) GenericExecute(envs map[string]string, name string, args []string, ctx *context.Context) error {
	return cmds.GenericExecute(envs, name, args, ctx)
}

// InstallAddons to HA RKE cluster
func (c MgmtCluster) InstallAddons() error {
	log.Infof("TODO: install addons")
	return nil
}

// RequiredCommands provides validation for required commands
func (c MgmtCluster) RequiredCommands() []string {
	log.Infof("TODO: provide required commands")
	return nil
}

// CreateBootstrap deploys a rancher container as single node RKE cluster
func (c MgmtCluster) CreateBootstrap() error {
	c.EventStream <- events.Event{EventType: "progress", Event: "docker pull rancher"}
	cli, err := c.dockerCli.NewEnvClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	imageName := "rancher/rancher"

	// This call was not working for some reason... required canonical image format?
	//_, err = cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	//if err != nil {
	//	return err
	//}
	args := []string{
		"pull",
		imageName,
	}
	err = c.osCli.GenericExecute(nil, string(docker), args, nil)
	if err != nil {
		return err
	}

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "80",
	}

	hostBinding2 := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "443",
	}

	containerHTTPPort, err := nat.NewPort("tcp", "80")
	if err != nil {
		return err
	}

	containerHTTPSPort, err := nat.NewPort("tcp", "443")
	if err != nil {
		return err
	}

	portBinding := nat.PortMap{containerHTTPPort: []nat.PortBinding{hostBinding}, containerHTTPSPort: []nat.PortBinding{hostBinding2}}

	resp, err := c.dockerCli.ContainerCreate(ctx, cli, &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			"80/tcp":  struct{}{},
			"443/tcp": struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}, nil, "")
	if err != nil {
		return err
	}

	c.EventStream <- events.Event{EventType: "progress", Event: "docker run rancher"}
	if err = c.dockerCli.ContainerStart(ctx, cli, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

// InstallControlPlane configures a single node RKE cluster
func (c *MgmtCluster) InstallControlPlane() error {
	// TODO: Remove TLS hack
	// Get "https://localhost/": x509: certificate signed by unknown authority
	dt := http.DefaultTransport
	switch dt.(type) {
	case *http.Transport:
		if dt.(*http.Transport).TLSClientConfig == nil {
			dt.(*http.Transport).TLSClientConfig = &tls.Config{}
		}
		dt.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
	}

	c.EventStream <- events.Event{EventType: "progress", Event: "wait for rancher API"}
	err := waitForRancherAPI()
	if err != nil {
		return err
	}

	c.EventStream <- events.Event{EventType: "progress", Event: "configure standalone rancher"}

	// Roughly the sequence followed for single node rancher server config:
	// https://forums.rancher.com/t/automating-rancher-2-x-installation-and-configuration/11454/2
	//# Login tokenResp good for 1 minute
	//LOGINTOKEN=`curl -k -s 'https://127.0.0.1/v3-public/localProviders/local?action=login' -H 'content-type: application/json' --data-binary '{"username":"admin","password":"admin","ttl":60000}' | jq -r .tokenResp`
	//
	//# Change password
	//curl -k -s 'https://127.0.0.1/v3/users?action=changepassword' -H 'Content-Type: application/json' -H "Authorization: Bearer $LOGINTOKEN" --data-binary '{"currentPassword":"admin","newPassword":"something better"}'
	//
	//# Create API key good forever
	//APIKEY=`curl -k -s 'https://127.0.0.1/v3/token' -H 'Content-Type: application/json' -H "Authorization: Bearer $LOGINTOKEN" --data-binary '{"type":"tokenResp","description":"for scripts and stuff"}' | jq -r .tokenResp`
	//echo "API Key: ${APIKEY}"
	//
	//# Set server-url
	//curl -k -s 'https://127.0.0.1/v3/settings/server-url' -H 'Content-Type: application/json' -H "Authorization: Bearer $APIKEY" -X PUT --data-binary '{"name":"server-url","value":"https://your-rancher.com/"}'

	body := v3public.BasicLogin{
		Password:  "admin",
		TTLMillis: 0,
		Username:  "admin",
	}
	jsonBody, err := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://localhost/v3-public/localProviders/local?action=login", bytes.NewBuffer(jsonBody))
	req.Header.Add("x-api-csrf", "d1b2b5ebf8")
	resp, _ := http.DefaultClient.Do(req)
	log.Infof("Enabled local login")
	log.Debugf("Enabled local login: %v+", resp)

	var tokenResp v3public.Token
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return errors.New("Unable to decode tokenResp: " + err.Error())
	}
	c.token = tokenResp.Token
	log.Infof("Using token: %s", c.token)

	c.rancherClient, err = v3.NewClient(&clientbase.ClientOpts{
		URL:      "https://localhost/v3",
		TokenKey: c.token,
	})
	if err != nil {
		return errors.New("Unable to create rancher client: " + err.Error())
	}

	log.Infof("Successfully created Rancher client")
	keys := make([]string, len(c.rancherClient.Types))
	for k := range c.rancherClient.Types {
		keys = append(keys, k)
	}
	log.Debugf("Schema Types: %v", keys)

	setting, err := c.rancherClient.Setting.ByID("server-url")
	if err != nil {
		return errors.New("Unable to get server-url setting: " + err.Error())
	}
	log.Debugf("Server URL setting : %v+", setting)

	setting, err = c.rancherClient.Setting.Update(setting, map[string]string{"name": "server-url", "value": "https://" + c.BootstrapIP})
	if err != nil {
		return errors.New("Unable to update server-url setting: " + err.Error())
	}
	log.Infof("Server URL updated : %s", setting.Value)

	return nil
}

// vsphereCloudCredential extends the rancher v3 model to include VMware properties
type vsphereCloudCredential struct {
	*v3.CloudCredential
	VMwareVsphereCredentialConfig vsphereCredentialConfig `json:"vmwarevspherecredentialConfig,omitempty" yaml:"vmwarevspherecredentialConfig,omitempty"`
}

// vsphereCredentialConfig are vSphere specific credential config properties
type vsphereCredentialConfig struct {
	Password    string `json:"password,omitempty" yaml:"password,omitempty"`
	Username    string `json:"username,omitempty" yaml:"username,omitempty"`
	Vcenter     string `json:"vcenter,omitempty" yaml:"vcenter,omitempty"`
	VcenterPort string `json:"vcenterPort,omitempty" yaml:"vcenterPort,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
}

// newVsphereCloudCredential constructor
func newVsphereCloudCredential(vcenter, username, password string) *vsphereCloudCredential {
	return &vsphereCloudCredential{
		CloudCredential: &v3.CloudCredential{
			Name: "rke-bootstrap",
		},
		VMwareVsphereCredentialConfig: vsphereCredentialConfig{
			Password:    password,
			Username:    username,
			Vcenter:     vcenter,
			VcenterPort: "443",
			Type:        "vmwarevspherecredentialconfig",
		},
	}
}

// vsphereNodeTemplate extends rancher v3 NodeTemplate model to include vSphere properties
type vsphereNodeTemplate struct {
	*v3.NodeTemplate
	NamespaceID         string              `json:"namespaceId,omitempty" yaml:"namespaceId,omitempty"`
	VmwareVsphereConfig vmwareVsphereConfig `json:"vmwarevsphereConfig,omitempty" yaml:"vmwarevsphereConfig,omitempty"`
}

// vmwareVsphereConfig vSphere specific NodeTemplate properties
type vmwareVsphereConfig struct {
	Boot2DockerURL   string `json:"boot2dockerUrl,omitempty" yaml:"boot2dockerurl,omitempty"`
	CloneFrom        string `json:"cloneFrom,omitempty" yaml:"cloneFrom,omitempty"`
	CloudConfig      string `json:"cloudConfig,omitempty" yaml:"cloudConfig,omitempty"`
	CloudInit        string `json:"cloudInit,omitempty" yaml:"cloudInit,omitempty"`
	ContentLibrary   string `json:"contentLibrary,omitempty" yaml:"contentLibrary,omitempty"`
	CreationType     string `json:"creationType,omitempty" yaml:"creationType,omitempty"`
	CPUCount         string `json:"cpuCount,omitempty" yaml:"cpuCount,omitempty"`
	Datacenter       string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	Datastore        string `json:"datastore,omitempty" yaml:"datastore,omitempty"`
	DatastoreCluster string `json:"datastoreCluster,omitempty" yaml:"datastoreCluster,omitempty"`
	DiskSize         string `json:"diskSize,omitempty" yaml:"diskSize,omitempty"`
	Folder           string `json:"folder,omitempty" yaml:"folder,omitempty"`
	Hostsystem       string `json:"hostsystem,omitempty" yaml:"region,omitempty"`
	MemorySize       string `json:"memorySize,omitempty" yaml:"memorySize,omitempty"`
	SSHPassword      string `json:"sshPassword,omitempty" yaml:"sshPassword,omitempty"`
	SSHPort          string `json:"sshPort,omitempty" yaml:"sshPort,omitempty"`
	SSHUser          string `json:"sshUser,omitempty" yaml:"sshUser,omitempty"`
	SSHUserGroup     string `json:"sshUserGroup,omitempty" yaml:"sshUserGroup,omitempty"`
	Pool             string `json:"pool,omitempty" yaml:"pool,omitempty"`
	*vsphereCredentialConfig
	VappIPAllocationPolicy string   `json:"vappIpallocationpolicy,omitempty" yaml:"vappIpallocationpolicy,omitempty"`
	VappIPProtocol         string   `json:"vappIpprotocol,omitempty" yaml:"vappIpprotocol,omitempty"`
	VappTransport          string   `json:"vappTransport,omitempty" yaml:"vappTransport,omitempty"`
	UseDataStoreCluster    bool     `json:"useDataStoreCluster,omitempty" yaml:"useDataStoreCluster,omitempty"`
	Network                []string `json:"network,omitempty" yaml:"network,omitempty"`
	CFGParam               []string `json:"cfgparam,omitempty" yaml:"cfgparam,omitempty"`
	Tag                    []string `json:"tag,omitempty" yaml:"tag,omitempty"`
	CustomAttribute        []string `json:"customAttribute,omitempty" yaml:"customAttribute,omitempty"`
	VappProperty           []string `json:"vappProperty,omitempty" yaml:"vappProperty,omitempty"`
}

// newVsphereNodeTemplate constructor
func newVsphereNodeTemplate(ccID, datacenter, datastore, folder, pool string, networks []string) *vsphereNodeTemplate {
	return &vsphereNodeTemplate{
		NodeTemplate: &v3.NodeTemplate{
			CloudCredentialID:    ccID,
			EngineInstallURL:     "https://releases.rancher.com/install-docker/19.03.sh",
			EngineRegistryMirror: make([]string, 0),
			UseInternalIPAddress: true,
			Labels:               make(map[string]string),
		},
		NamespaceID: "fixme",
		VmwareVsphereConfig: vmwareVsphereConfig{
			Boot2DockerURL:   "https://releases.rancher.com/os/latest/rancheros-vmware.iso",
			CloneFrom:        "",
			CloudConfig:      "",
			CloudInit:        "",
			ContentLibrary:   "",
			CPUCount:         "2",
			CreationType:     "legacy",
			Datacenter:       datacenter,
			Datastore:        datastore,
			DatastoreCluster: "",
			DiskSize:         "20000",
			Folder:           folder,
			Hostsystem:       "",
			MemorySize:       "2048",
			SSHPassword:      "tcuser",
			SSHPort:          "22",
			SSHUser:          "docker",
			SSHUserGroup:     "staff",
			Pool:             pool,
			vsphereCredentialConfig: &vsphereCredentialConfig{
				Password:    "",
				Username:    "",
				Vcenter:     "",
				VcenterPort: "443",
				Type:        "vmwarevsphereConfig",
			},
			VappIPAllocationPolicy: "",
			VappIPProtocol:         "",
			VappTransport:          "",
			UseDataStoreCluster:    false,
			Network:                networks,
			Tag:                    make([]string, 0),
			CustomAttribute:        make([]string, 0),
			CFGParam:               []string{"disk.enableUUID=TRUE"},
			VappProperty:           make([]string, 0),
		},
	}
}

// CreatePermanent deploys HA RKE cluster to vSphere
func (c *MgmtCluster) CreatePermanent() error {
	c.EventStream <- events.Event{EventType: "progress", Event: "configure RKE management cluster"}
	// POST https://localhost/v3/cloudcredential
	body := newVsphereCloudCredential(c.URL, c.Username, c.Password)
	resp, err := c.makeHTTPRequest("POST", "https://localhost/v3/cloudcredential", body)
	if err != nil {
		return err
	}
	log.Info("Created vsphere cloud cred")
	var credResp v3.CloudCredential
	err = json.NewDecoder(resp.Body).Decode(&credResp)
	if err != nil {
		return errors.New("unable to decode cloud cred response: " + err.Error())
	}
	cloudCredID := credResp.ID
	log.Infof("Cloud cred ID: %v", cloudCredID)

	nodeTemplate := newVsphereNodeTemplate(cloudCredID, c.Datacenter, c.Datastore, c.Folder, c.ResourcePool, []string{c.ManagementNetwork})
	resp, err = c.makeHTTPRequest("POST", "https://localhost/v3/nodetemplate", nodeTemplate)
	if err != nil {
		return err
	}
	log.Debugf("Created node template: %v+", resp)
	var nodeTemplateResp v3.NodeTemplate
	err = json.NewDecoder(resp.Body).Decode(&nodeTemplateResp)
	if err != nil {
		return err
	}
	nodeTemplateID := nodeTemplateResp.ID
	log.Infof("Node template ID: %v+", nodeTemplateID)

	clusterReq := &v3.Cluster{
		DockerRootDir:           "/var/lib/docker",
		EnableClusterAlerting:   false,
		EnableClusterMonitoring: false,
		EnableNetworkPolicy:     nil,
		WindowsPreferedCluster:  false,
		Name:                    c.ClusterName,
		RancherKubernetesEngineConfig: &v3.RancherKubernetesEngineConfig{
			AddonJobTimeout:     30,
			Version:             c.KubernetesVersion,
			IgnoreDockerVersion: true,
			SSHAgentAuth:        false,
			Authentication: &v3.AuthnConfig{
				Strategy: "x509",
			},
			DNS: &v3.DNSConfig{}, // This may be an issue nodelocal?
			Network: &v3.NetworkConfig{
				Options: map[string]string{
					"flannel_backend_type": "vxlan",
				},
				Plugin: "canal",
			},
			Ingress: &v3.IngressConfig{
				Provider: "nginx",
			},
			Monitoring: &v3.MonitoringConfig{
				Provider: "metrics-server",
			},
			Services: &v3.RKEConfigServices{
				KubeAPI: &v3.KubeAPIService{
					AlwaysPullImages:     false,
					PodSecurityPolicy:    false,
					ServiceNodePortRange: "30000-32767",
				},
				Etcd: &v3.ETCDService{
					Creation: "12h",
					ExtraArgs: map[string]string{
						"heartbeat-interval": "500",
						"election-timeout":   "5000",
					},
					GID:       0,
					Retention: "72h",
					Snapshot:  &[]bool{false}[0],
					UID:       0,
					BackupConfig: &v3.BackupConfig{
						Enabled:       &[]bool{true}[0],
						IntervalHours: 12,
						Retention:     6,
						SafeTimestamp: false,
					},
				},
			},
			// Missing UpgradeStrategy
		},
		LocalClusterAuthEndpoint: &v3.LocalClusterAuthEndpoint{
			Enabled: true,
		},
		// Missing ScheduledClusterScan
	}
	clusterResp, err := c.rancherClient.Cluster.Create(clusterReq)
	resp, err = c.makeHTTPRequest("POST", "https://localhost/v3/cluster?_replace=true", clusterResp)
	if err != nil {
		return err
	}
	log.Infof("Created cluster")
	clusterID := clusterResp.ID
	c.clusterURL = clusterResp.Links["self"]
	log.Infof("Cluster ID: %v+", clusterID)

	err = c.createNodePools(clusterID, nodeTemplateID)
	if err != nil {
		return err
	}

	c.EventStream <- events.Event{EventType: "progress", Event: "waiting 15 minutes for RKE cluster to be ready"}
	err = c.waitForCondition(c.clusterURL, "type", "Ready", 15)
	if err != nil {
		return err
	}

	var g errgroup.Group
	nodeCollectionResp, err := c.rancherClient.Node.List(&rTypes.ListOpts{})
	if err != nil {
		return err
	}
	for _, node := range nodeCollectionResp.Data {
		g.Go(func() error {
			return c.waitForCondition(node.Links["self"], "type", "Ready", 5)
		})
	}

	return g.Wait()
}

// PivotControlPlane deploys rancher server via helm chart to HA RKE cluster
func (c MgmtCluster) PivotControlPlane() error {
	c.EventStream <- events.Event{EventType: "progress", Event: "install production rancher server"}

	catalogReq := &v3.Catalog{
		Branch:   "master",
		Kind:     "helm",
		Name:     "rancher-latest",
		URL:      "https://releases.rancher.com/server-charts/latest",
		Username: "",
		Password: "",
	}
	catalogResp, err := c.rancherClient.Catalog.Create(catalogReq)
	if err != nil {
		return err
	}
	log.Info("Added rancher helm chart")

	err = c.waitForCondition(catalogResp.Links["self"], "type", "Refreshed", 2)

	// I don't know if setting the default project ID is necessary. The UI did it so I added it here as well
	var defaultProj v3.Project
	projCollectionResp, err := c.rancherClient.Project.List(&rTypes.ListOpts{})
	if err != nil {
		return err
	}
	log.Info("Got all projects")
	for _, proj := range projCollectionResp.Data {
		if proj.Name == "Default" {
			defaultProj = proj
		}
	}
	log.Infof("Got default project ID: %s", defaultProj.ID)

	projSplit := strings.Split(defaultProj.ID, ":")
	pID := projSplit[1]

	resp, err := c.makeHTTPRequest("GET", fmt.Sprintf("%s/namespaces/default", c.clusterURL), nil)
	if err != nil {
		return err
	}
	log.Infof("Got default namespace")
	result := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	labels := result["labels"].(map[string]interface{})
	labels["field.cattle.io/projectId"] = pID
	result["projectId"] = defaultProj.ID
	resp, err = c.makeHTTPRequest("PUT", fmt.Sprintf("%s/namespaces/default", c.clusterURL), result)
	if err != nil {
		return err
	}
	log.Infof("Updated default namespace")

	appName := "rancher"
	appReq := &v3project.App{
		Prune:   false,
		Timeout: 300,
		Wait:    false,
		Name:    appName,
		Answers: map[string]string{
			"tls": "external",
		},
		TargetNamespace: "default",
		ExternalID:      "catalog://?catalog=rancher-latest&template=rancher&version=2.4.2",
		ProjectID:       defaultProj.ID,
		ValuesYaml:      "",
	}
	resp, err = c.makeHTTPRequest("POST", fmt.Sprintf("%s/app", defaultProj.Links["self"]), appReq)
	if err != nil {
		return err
	}
	log.Infof("Deployed rancher server via helm")

	var appResp v3project.App
	err = json.NewDecoder(resp.Body).Decode(&appResp)
	rancherAppURL := appResp.Links["self"]
	log.Infof("Rancher app URL: %s", rancherAppURL)

	c.EventStream <- events.Event{EventType: "progress", Event: "waiting 5 minutes for rancher server to be ready"}
	err = c.waitForCondition(rancherAppURL, "type", "Deployed", 5)
	if err != nil {
		return err
	}

	resp, err = c.makeHTTPRequest("GET", defaultProj.Links["workloads"], nil)
	if err != nil {
		return err
	}
	var workloadCollectionResp v3project.WorkloadCollection
	err = json.NewDecoder(resp.Body).Decode(&workloadCollectionResp)
	if err != nil {
		return err
	}
	var rWorkload v3project.Workload
	for _, w := range workloadCollectionResp.Data {
		if w.Name == appName {
			rWorkload = w
			break
		}
	}
	log.Infof("Rancher app workload ID: %s", rWorkload.ID)

	if err = waitForAvailable(func() []v3project.DeploymentCondition {
		resp, _ := c.makeHTTPRequest("GET", rWorkload.Links["self"], nil)
		var w v3project.Workload
		_ = json.NewDecoder(resp.Body).Decode(&w)
		return w.DeploymentStatus.Conditions
	}); err != nil {
		return err
	}

	resp, _ = c.makeHTTPRequest("GET", rWorkload.Links["self"], nil)
	var w v3project.Workload
	err = json.NewDecoder(resp.Body).Decode(&w)
	if err != nil {
		return err
	}

	var rancherAddr string
	for _, e := range w.PublicEndpoints {
		if e.Protocol == "HTTPS" {
			if len(e.Addresses) == 0 {
				return errors.New("unable to find public HTTPS rancher URL")
			}
			rancherAddr = e.Addresses[0]
		}
	}

	log.Infof("Rancher is available: https://%s", rancherAddr)
	return nil
}

// Events returns the channel of progress messages
func (c MgmtCluster) Events() chan events.Event {
	return c.EventStream
}

func (c MgmtCluster) waitForCondition(resourceURL, key, val string, timeoutInMins int) error {
	timeout := time.After(time.Duration(timeoutInMins) * time.Minute)
	tick := time.Tick(30 * time.Second)
	cReceived := make(map[string]struct{})
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout after %d minutes waiting for %s with condition %s=%s", timeoutInMins, resourceURL, key, val)
		case <-tick:
			resp, _ := c.makeHTTPRequest("GET", resourceURL, nil)
			if resp != nil {
				result := make(map[string]interface{})
				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					log.Warnf(err.Error())
				}
				if conditions, ok := result["conditions"].([]interface{}); ok {
					for _, c := range conditions {
						cMap := c.(map[string]interface{})
						condition := cMap[key].(string)
						_, ok := cReceived[condition]
						if !ok {
							log.Infof("Received a new condition: %s", condition)
							cReceived[condition] = struct{}{}
						}
						if condition == val {
							return nil
						}
					}
				}
			}
			log.Info("Waiting for resource...")
		}
	}
}

func waitForAvailable(cFunc func() []v3project.DeploymentCondition) error {
	timeout := time.After(5 * time.Minute)
	tick := time.Tick(30 * time.Second)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout after 5 minutes waiting for available condition")
		case <-tick:
			conditions := cFunc()
			for _, c := range conditions {
				if c.Type == "Available" {
					return nil
				}
			}
			log.Info("Waiting for available...")
		}
	}
}

func (c MgmtCluster) createNodePools(clusterID, nodeTemplateID string) error {
	mgmtCount := int64(c.ControlPlaneCount)
	workerCount := int64(c.WorkerCount)
	/*
		mgmtCount, err := strconv.ParseInt(c.ControlPlaneCount, 10, 64)
		if err != nil {
			log.Warnf("Unable to parse ControlPlaneMachineCount, defaulting to 1: %s", err)
			mgmtCount = 1
		}
		workerCount, err := strconv.ParseInt(c.WorkerMachineCount, 10, 64)
		if err != nil {
			log.Warnf("Unable to parse WorkerMachineCount, defaulting to 2: %s", err)
			workerCount = 2
		}
	*/
	nodePools := []struct {
		prefix string
		count  int64
		ctrl   bool
		worker bool
		etcd   bool
	}{
		{"rke-ctrl", mgmtCount, true, false, true},
		{"rke-worker", workerCount, false, true, true},
		//{"rke-etcd", 1, false, false, true},
	}
	for _, np := range nodePools {
		nodePoolReq := &v3.NodePool{
			ControlPlane:            np.ctrl,
			DeleteNotReadyAfterSecs: 0,
			Etcd:                    np.etcd,
			Quantity:                np.count,
			Worker:                  np.worker,
			ClusterID:               clusterID,
			NodeTemplateID:          nodeTemplateID,
			HostnamePrefix:          np.prefix,
		}
		nodePoolResp, err := c.rancherClient.NodePool.Create(nodePoolReq)
		if err != nil {
			return err
		}
		log.Info("Created node pool: ", nodePoolResp.HostnamePrefix)
	}
	return nil
}

func (c MgmtCluster) makeHTTPRequest(method, url string, payload interface{}) (*http.Response, error) {
	var req *http.Request
	if payload != nil {
		body, ok := payload.([]byte)
		if !ok {
			body, _ = json.Marshal(payload)
		}
		req, _ = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	req.Header.Add("x-api-csrf", "d1b2b5ebf8")
	req.Header.Add("Authorization", "Bearer "+c.token)
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("HTTP request: %q", dump)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, err
	}

	dump, err = httputil.DumpResponse(resp, true)
	if err != nil {
		return resp, err
	}

	log.Debugf("HTTP response: %q", dump)
	return resp, err
}

func waitForRancherAPI() error {
	timeout := time.After(time.Minute * 2)
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New("timeout after 2 minutes waiting for rancher API")
		case <-tick:
			resp, err := http.DefaultClient.Get("https://localhost")
			if err != nil {
				log.Debugf("Ignoring error getting URL: %s", err)
			}
			if resp != nil {
				if resp.StatusCode != http.StatusOK {
					log.Debugf("Ignoring unsuccessful HTTP response from Rancher API: %v+", resp)
				} else {
					log.Info("Rancher API is responding")
					// TODO: Figure out if this is still required
					time.Sleep(time.Second * 5)
					return nil
				}
			}
		}
	}
}
