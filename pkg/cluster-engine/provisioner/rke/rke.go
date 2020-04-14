package rke

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/netapp/capv-bootstrap/pkg/cluster-engine/provisioner"
	"github.com/netapp/capv-bootstrap/pkg/cluster-engine/provisioner/capv"
	"github.com/netapp/capv-bootstrap/pkg/cmds"
	"github.com/prometheus/common/log"
	rancher "github.com/rancher/go-rancher/client"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type requiredCmd string

const (
	kind       requiredCmd = "kind"
	clusterctl requiredCmd = "clusterctl"
	kubectl    requiredCmd = "kubectl"
	docker     requiredCmd = "docker"
	helm       requiredCmd = "helm"
)

// RequiredCommands for capv provisioner
var RequiredCommands = cmds.ProvisionerCommands{Name: "required CAPV bootstrap commands"}

func init() {
	kd := cmds.NewCommandLine(nil, string(kind), nil, nil)
	c := cmds.NewCommandLine(nil, string(clusterctl), nil, nil)
	k := cmds.NewCommandLine(nil, string(kubectl), nil, nil)
	d := cmds.NewCommandLine(nil, string(docker), nil, nil)
	//h := cmds.NewCommandLine(nil, string(helm), nil, nil)

	RequiredCommands.AddCommand(kd.CommandName, kd)
	RequiredCommands.AddCommand(c.CommandName, c)
	RequiredCommands.AddCommand(k.CommandName, k)
	RequiredCommands.AddCommand(d.CommandName, d)
	//RequiredCommands.AddCommand(h.CommandName, h)

}

// NewMgmtCluster creates a new cluster interface
func NewMgmtCluster(controlPlaneMachineCount, workerMachineCount, clustername string) provisioner.Cluster {
	mc := new(MgmtCluster)
	mc.ClusterName = clustername
	mc.ControlPlaneMachineCount = controlPlaneMachineCount
	mc.WorkerMachineCount = workerMachineCount
	mc.events = make(chan interface{})
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	return mc
}

// NewMgmtClusterFullConfig creates a new cluster interface with a full config from the client
func NewMgmtClusterFullConfig(clusterConfig MgmtCluster) provisioner.Cluster {
	mc := new(MgmtCluster)
	mc = &clusterConfig
	mc.events = make(chan interface{})
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}
	return mc
}

// MgmtCluster spec for CAPV
type MgmtCluster struct {
	Datacenter               string `yaml:"Datacenter"`
	Datastore                string `yaml:"Datastore"`
	Folder                   string `yaml:"Folder"`
	LoadBalancerTemplate     string `yaml:"LoadBalancerTemplate"`
	NodeTemplate             string `yaml:"NodeTemplate"`
	ManagementNetwork        string `yaml:"ManagementNetwork"`
	WorkloadNetwork          string `yaml:"WorkloadNetwork"`
	StorageNetwork           string `yaml:"StorageNetwork"`
	ResourcePool             string `yaml:"ResourcePool"`
	VcenterServer            string `yaml:"VcenterServer"`
	VsphereUsername          string `yaml:"VsphereUsername"`
	VspherePassword          string `yaml:"VspherePassword"`
	ClusterName              string `yaml:"ClusterName"`
	CapiSpec                 string `yaml:"CapiSpec"`
	KubernetesVersion        string `yaml:"KubernetesVersion"`
	Namespace                string `yaml:"Namespace"`
	Kubeconfig               string `yaml:"Kubeconfig"`
	SSHAuthorizedKey         string `yaml:"SshAuthorizedKey"`
	ControlPlaneMachineCount string `yaml:"ControlPlaneMachineCount"`
	WorkerMachineCount       string `yaml:"WorkerMachineCount"`
	LogFile                  string `yaml:"LogFile"`
	events                   chan interface{}
	Solidfire                struct {
		Enable   bool   `yaml:"Enable"`
		MVIP     string `yaml:"MVIP"`
		SVIP     string `yaml:"SVIP"`
		User     string `yaml:"User"`
		Password string `yaml:"Password"`
	} `yaml:"Solidfire"`
	Configuration struct {
		Cluster struct {
			KubernetesPodCidr     string `yaml:"KubernetesPodCidr"`
			KubernetesServiceCidr string `yaml:"KubernetesServiceCidr"`
		} `yaml:"Cluster"`
		Observability struct {
			Enabled         bool   `yaml:"Enabled"`
			ArchiveLocation string `yaml:"ArchiveLocation"`
		} `yaml:"Observability"`
	} `yaml:"Configuration"`
	token string
}

func (c MgmtCluster) CreateBootstrap() error {
	var err error

	c.events <- capv.Event{EventType: "progress", Event: "docker run rancher"}
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	imageName := "rancher/rancher"

	//_, err = cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	//if err != nil {
	//	return err
	//}
	dockerPullCmd := exec.Command("docker", "pull", imageName)
	if err := dockerPullCmd.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok && status.ExitStatus() > 0 {
				return err
			}
		} else {
			return err
		}
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

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
			"443/tcp": struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: portBinding,
		RestartPolicy:   container.RestartPolicy{
			Name:              "unless-stopped",
		},
	}, nil, "")
	if err != nil {
		return err
	}

	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	// LOL - This command never completes, thanks Rancher :P
	//code, err := cli.ContainerWait(ctx, resp.ID)
	//if err != nil || code > 0 {
	//	return errors.New(fmt.Sprintf("Error waiting for container, code: %d, err: %s", code, err))
	//}
	// Forgive me, for I have sinned
	time.Sleep(time.Minute * 2)

	//args := []string{
	//	"create",
	//	"cluster",
	//}
	//err = cmds.GenericExecute(nil, string(kind), args, nil)
	//if err != nil {
	//	return err
	//}
	//
	//m.events <- Event{EventType: "progress", Event: "getting and writing bootstrap cluster kubeconfig to disk"}
	//args = []string{
	//	"get",
	//	"kubeconfig",
	//}
	//c := cmds.NewCommandLine(nil, string(kind), args, nil)
	//stdout, stderr, err := c.Program().Execute()
	//if err != nil || string(stderr) != "" {
	//	return fmt.Errorf("err: %v, stderr: %v", err, string(stderr))
	//}
	//
	//err = writeToDisk(m.ClusterName, bootstrapKubeconfig, []byte(stdout), 0644)
	//if err != nil {
	//	return err
	//}
	//
	//// TODO wait for cluster components to be running
	//m.events <- Event{EventType: "progress", Event: "sleeping 20 seconds, need to fix this"}
	//time.Sleep(20 * time.Second)

	return err
}

func (c *MgmtCluster) InstallCAPV() error {
	c.events <- capv.Event{EventType: "progress", Event: "configure standalone rancher"}

	// Remove hack
	// Get "https://localhost/": x509: certificate signed by unknown authority
	dt := http.DefaultTransport
	switch dt.(type) {
	case *http.Transport:
		if dt.(*http.Transport).TLSClientConfig == nil {
			dt.(*http.Transport).TLSClientConfig = &tls.Config{}
		}
		dt.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
	}

	// https://forums.rancher.com/t/automating-rancher-2-x-installation-and-configuration/11454/2
	//# Login token good for 1 minute
	//LOGINTOKEN=`curl -k -s 'https://127.0.0.1/v3-public/localProviders/local?action=login' -H 'content-type: application/json' --data-binary '{"username":"admin","password":"admin","ttl":60000}' | jq -r .token`
	//
	//# Change password
	//curl -k -s 'https://127.0.0.1/v3/users?action=changepassword' -H 'Content-Type: application/json' -H "Authorization: Bearer $LOGINTOKEN" --data-binary '{"currentPassword":"admin","newPassword":"something better"}'
	//
	//# Create API key good forever
	//APIKEY=`curl -k -s 'https://127.0.0.1/v3/token' -H 'Content-Type: application/json' -H "Authorization: Bearer $LOGINTOKEN" --data-binary '{"type":"token","description":"for scripts and stuff"}' | jq -r .token`
	//echo "API Key: ${APIKEY}"
	//
	//# Set server-url
	//curl -k -s 'https://127.0.0.1/v3/settings/server-url' -H 'Content-Type: application/json' -H "Authorization: Bearer $APIKEY" -X PUT --data-binary '{"name":"server-url","value":"https://your-rancher.com/"}'

	//
	body, _ := json.Marshal(map[string]interface{}{
		"username": "admin",
		"password": "admin",
		"ttl": 0,
	})
	req, err := http.NewRequest("POST", "https://localhost/v3-public/localProviders/local?action=login", bytes.NewBuffer(body))
	req.Header.Add("x-api-csrf", "d1b2b5ebf8")
	resp, err := http.DefaultClient.Do(req)
	log.Infof("Enabled local login: %v+", resp)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	s := strings.Split(result["token"], ":")
	user := s[0]
	token := s[1]
	c.token = result["token"]
	log.Infof("Using token %s user: %s token: %s", result["token"], user, token)

	//body, err = json.Marshal(map[string]string{
	//	"currentPassword": "admin",
	//	"newPassword": "solidfire",
	//})
	//if err != nil {
	//	return err
	//}
	//req, err = http.NewRequest("POST", "https://localhost/v3/users?action=changepassword", bytes.NewBuffer(body))
	//if err != nil {
	//	return err
	//}
	//
	//req.Header.Add("x-api-csrf", "d1b2b5ebf8")
	//resp, err = http.DefaultClient.Do(req)
	//if err != nil {
	//	return err
	//}

	opts := &rancher.ClientOpts{
		Url: "https://localhost",
		AccessKey: user,
		SecretKey: token,
	}

	cli, err := rancher.NewRancherClient(opts)
	if err != nil {
		return errors.New("Unable to create rancher client: " + err.Error())
	}

	log.Infof("Successfully created Rancher client")
	keys := make([]string, len(cli.Types))
	for k := range cli.Types {
		keys = append(keys, k)
	}
	log.Infof("Schema Types: %v", keys)

	// https://github.com/cloudnautique/rbs-sandbox/blob/b1c3236490d16ba82ea4e1b5849bdcb1c913c292/rancher/rancherserver.go
	//if opts.AccessKey == "" || opts.SecretKey == "" {
	//	apiKey, err := cli.ApiKey.Create(&rancher.ApiKey{
	//		AccountId: "1a1",
	//	})
	//	if err != nil {
	//		return err
	//	}
	//	//fileToWrite, err := os.Create("tmp")
	//	//if err != nil {
	//	//	logrus.Fatalf("Could not write out keys: %s", err)
	//	//}
	//	//
	//	//encoder := candiedyaml.NewEncoder(fileToWrite)
	//	//err = encoder.Encode(keyDataOut)
	//	//if err != nil {
	//	//	logrus.Fatalf("Failed to encode keys: %s", err)
	//	//}
	//
	//	cli.Opts.AccessKey = apiKey.PublicValue
	//	cli.Opts.SecretKey = apiKey.SecretValue
	//}


	log.Infof("Using Access Key: %s", cli.Opts.AccessKey)


	body, _ = json.Marshal(map[string]interface{}{
		"name": "server-url",
		"value": "https://172.60.5.53",
	})
	req, err = http.NewRequest("PUT", "https://127.0.0.1/v3/settings/server-url", bytes.NewBuffer(body))
	req.Header.Add("x-api-csrf", "d1b2b5ebf8")
	req.Header.Add("Authorization", "Bearer "+ result["token"])
	resp, err = http.DefaultClient.Do(req)
	log.Infof("Changed server URL: %v+", resp)

	// Ayaye
	//setting, err := cli.Setting.ById("server-url")
	//if err != nil {
	//	return errors.New("Unable to get server-url setting: " + err.Error())
	//}
	//
	//log.Infof("Server URL setting : %s", setting)
	//
	//setting, err = cli.Setting.Update(setting, map[string]string{"name":"server-url","value":"https://localhost"})
	//if err != nil {
	//	return errors.New("Unable to update server-url setting: " + err.Error())
	//}
	//
	//log.Infof("Server URL updated : %s", setting)

	//out, err := cli.Setting.List(rancher.NewListOpts())
	//if err != nil {
	//	return err
	//}
	//log.Infof("Settings: %v+", out)

	return nil
}

func (c MgmtCluster) CreatePermanent() error {
	c.events <- capv.Event{EventType: "progress", Event: "configure RKE management cluster"}

	// POST https://localhost/v3/cloudcredential
	b := []byte(`{
		"type": "cloudCredential",
		"vmwarevspherecredentialConfig": {
			"password": "NetApp1!!",
			"username": "administrator@vsphere.local",
			"vcenter": "172.60.0.151",
			"vcenterPort": "443",
			"type": "vmwarevspherecredentialconfig"
		},
		"name": "rke-bootstrap"
	}`)
	log.Infof("byte body: %q", b)
	resp, err := c.makeHTTPRequest("POST", "https://localhost/v3/cloudcredential", b)
	if err != nil {
		return err
	}
	log.Infof("Created vsphere cloud cred: %v+", resp)
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	cloudCredID := result["id"].(string)
	log.Infof("Cloud cred ID: %v+", cloudCredID)


	// POST https://localhost/v3/nodetemplate
	b = []byte(`{
		"useInternalIpAddress": true,
		"type": "nodeTemplate",
		"engineInstallURL": "https://releases.rancher.com/install-docker/19.03.sh",
		"engineRegistryMirror": [],
		"vmwarevsphereConfig": {
			"boot2dockerUrl": "https://releases.rancher.com/os/latest/rancheros-vmware.iso",
			"cloneFrom": "",
			"cloudConfig": "",
			"cloudinit": "",
			"contentLibrary": "",
			"cpuCount": "2",
			"creationType": "legacy",
			"datacenter": "/NetApp-HCI-Datacenter-01",
			"datastore": "/NetApp-HCI-Datacenter-01/datastore/NetApp-HCI-Datastore-01",
			"datastoreCluster": "",
			"diskSize": "20000",
			"folder": "/NetApp-HCI-Datacenter-01/vm/rancher",
			"hostsystem": "",
			"memorySize": "2048",
			"password": "",
			"pool": "/NetApp-HCI-Datacenter-01/host/NetApp-HCI-Cluster-01/Resources/rancher",
			"sshPassword": "tcuser",
			"sshPort": "22",
			"sshUser": "docker",
			"sshUserGroup": "staff",
			"username": "",
			"vappIpallocationpolicy": "",
			"vappIpprotocol": "",
			"vappTransport": "",
			"vcenter": "",
			"vcenterPort": "443",
			"type": "vmwarevsphereConfig",
			"useDataStoreCluster": false,
			"network": ["/NetApp-HCI-Datacenter-01/network/VM_Network"],
			"tag": [],
			"customAttribute": [],
			"cfgparam": ["disk.enableUUID=TRUE"],
			"vappProperty": []
		},
		"namespaceId": "fixme",
		"cloudCredentialId": "cattle-global-data:cc-sqqg9",
		"labels": {}
	}`)
	reqJSON := make(map[string]interface{})
	json.Unmarshal(b, &reqJSON)
	reqJSON["cloudCredentialId"] = cloudCredID

	resp, err = c.makeHTTPRequest("POST", "https://localhost/v3/nodetemplate", reqJSON)
	if err != nil {
		return err
	}
	log.Infof("Created node template: %v+", resp)
	respJSON := make(map[string]interface{})
	json.NewDecoder(resp.Body).Decode(&respJSON)
	nodeTemplateID := respJSON["id"].(string)
	log.Infof("Node template ID: %v+", nodeTemplateID)


	// POST https://localhost/v3/cluster?_replace=true
	b = []byte(`{
		"dockerRootDir": "/var/lib/docker",
		"enableClusterAlerting": false,
		"enableClusterMonitoring": false,
		"enableNetworkPolicy": false,
		"windowsPreferedCluster": false,
		"type": "cluster",
		"name": "rke-management",
		"rancherKubernetesEngineConfig": {
		"addonJobTimeout": 30,
			"ignoreDockerVersion": true,
			"sshAgentAuth": false,
			"type": "rancherKubernetesEngineConfig",
			"kubernetesVersion": "v1.17.4-rancher1-3",
			"authentication": {
			"strategy": "x509",
				"type": "authnConfig"
		},
		"dns": {
			"type": "dnsConfig",
				"nodelocal": {
				"type": "nodelocal",
					"ip_address": "",
					"node_selector": null,
					"update_strategy": {}
			}
		},
		"network": {
			"mtu": 0,
				"plugin": "canal",
				"type": "networkConfig",
				"options": {
				"flannel_backend_type": "vxlan"
			}
		},
		"ingress": {
			"provider": "nginx",
				"type": "ingressConfig"
		},
		"monitoring": {
			"provider": "metrics-server",
				"replicas": 1,
				"type": "monitoringConfig"
		},
		"services": {
			"type": "rkeConfigServices",
				"kubeApi": {
				"alwaysPullImages": false,
					"podSecurityPolicy": false,
					"serviceNodePortRange": "30000-32767",
					"type": "kubeAPIService"
			},
			"etcd": {
				"creation": "12h",
					"extraArgs": {
					"heartbeat-interval": 500,
						"election-timeout": 5000
				},
				"gid": 0,
					"retention": "72h",
					"snapshot": false,
					"uid": 0,
					"type": "etcdService",
					"backupConfig": {
					"enabled": true,
						"intervalHours": 12,
						"retention": 6,
						"safeTimestamp": false,
						"type": "backupConfig"
				}
			}
		},
		"upgradeStrategy": {
			"maxUnavailableControlplane": "1",
				"maxUnavailableWorker": "10%%",
				"drain": "false",
				"nodeDrainInput": {
				"deleteLocalData": "false",
					"force": false,
					"gracePeriod": -1,
					"ignoreDaemonSets": true,
					"timeout": 120,
					"type": "nodeDrainInput"
			},
			"maxUnavailableUnit": "percentage"
		}
	},
		"localClusterAuthEndpoint": {
		"enabled": true,
			"type": "localClusterAuthEndpoint"
	},
		"labels": {},
		"scheduledClusterScan": {
		"enabled": false,
			"scheduleConfig": null,
			"scanConfig": null
		}
	}`)
	resp, err = c.makeHTTPRequest("POST", "https://localhost/v3/cluster?_replace=true", b)
	if err != nil {
		return err
	}
	log.Infof("Created cluster: %v+", resp)
	result = make(map[string]interface{})
	json.NewDecoder(resp.Body).Decode(&result)
	clusterID := result["id"].(string)
	log.Infof("Cluster ID: %v+", clusterID)

	// POST - https://localhost/v3/nodepool
	/*
	{
		"controlPlane": true,
		"deleteNotReadyAfterSecs": 0,
		"etcd": false,
		"quantity": 1,
		"worker": false,
		"type": "nodePool",
		"clusterId": "c-qtsbz",
		"nodeTemplateId": "cattle-global-nt:nt-v5v22",
		"hostnamePrefix": "rke-ctrl"
	}

	{"controlPlane":false,"deleteNotReadyAfterSecs":0,"etcd":false,"quantity":2,"worker":true,"type":"nodePool","nodeTemplateId":"cattle-global-nt:nt-v5v22","clusterId":"c-qtsbz","hostnamePrefix":"rke-worker"}

	{"controlPlane":false,"deleteNotReadyAfterSecs":0,"etcd":true,"quantity":1,"worker":false,"type":"nodePool","nodeTemplateId":"cattle-global-nt:nt-v5v22","clusterId":"c-qtsbz","hostnamePrefix":"rke-etcd"}
	 */
	return c.createNodePools(clusterID, nodeTemplateID)
}


func (c MgmtCluster) CAPvPivot() error {
	return nil
}

// Events returns the channel of progress messages
func (c *MgmtCluster) Events() chan interface{} {
	return c.events
}

func (c MgmtCluster) createNodePools(clusterID, nodeTemplateID string) error {
	nodePools := []struct {
		prefix string
		count int
		ctrl bool
		worker bool
		etcd bool
	}{
		{"rke-ctrl", 1, true, false, false},
		{"rke-worker", 2, false, true, false},
		{"rke-etcd", 1, false, false, true},
	}
	for _, np := range nodePools {
		req := createNodePoolReq(clusterID, nodeTemplateID, np.prefix, np.count, np.ctrl, np.worker, np.etcd)
		resp, err := c.makeHTTPRequest("POST", "https://localhost/v3/nodepool", req)
		if err != nil {
			return err
		}
		log.Infof("Created node pool: %v+", resp)
	}
	return nil
}

func (c MgmtCluster) makeHTTPRequest(method, url string, payload interface{}) (*http.Response, error) {
	body, ok := payload.([]byte)
	if !ok {
		body, _ = json.Marshal(payload)
	}
	log.Infof("body: %q", body)
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Add("x-api-csrf", "d1b2b5ebf8")
	req.Header.Add("Authorization", "Bearer "+ c.token)
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("HTTP request: %q", dump)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, err
	}

	dump, err = httputil.DumpResponse(resp, true)
	if err != nil {
		return resp, err
	}

	log.Infof("HTTP response: %q", dump)
	return resp, err
}

func createNodePoolReq(clusterID, nodeTemplateID, prefix string, cnt int, ctrl, worker, etcd bool) map[string]interface{} {
	b := []byte(`{
		"controlPlane": true,
		"deleteNotReadyAfterSecs": 0,
		"etcd": false,
		"quantity": 1,
		"worker": false,
		"type": "nodePool",
		"clusterId": "c-qtsbz",
		"nodeTemplateId": "cattle-global-nt:nt-v5v22",
		"hostnamePrefix": "rke-ctrl"
	}`)
	result := make(map[string]interface{})
	json.Unmarshal(b, &result)
	result["clusterId"] = clusterID
	result["nodeTemplateId"] = nodeTemplateID
	result["hostnamePrefix"] = prefix
	result["quantity"] = cnt
	result["worker"] = worker
	result["controlPlane"] = ctrl
	result["etcd"] = etcd
	return result
}
