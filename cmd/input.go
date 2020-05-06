package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/netapp/cake/pkg/config/types"
)

const (
	labelVCenterURL                   = "vCenter URL"
	labelVCenterUsername              = "vCenter username"
	labelVCenterPassword              = "vCenter password"
	labelVCenterDatacenter            = "vSphere datacenter"
	labelVCenterresourcepool          = "vSphere resourcepool"
	labelVCenterDatastore             = "vSphere datastore"
	labelCAPVRegionName               = "Region name"
	labelCAPVManagementClusterNetwork = "Management Cluster Network"
	labelCAPVWorkloadClusterNetwork   = "Workload Cluster Network"

	labelAddStorageNetwork  = "(Optional) Do you want to add a storage network to your workload cluster nodes?"
	labelCAPVStorageNetwork = "Workload Cluster Storage Network"

	labelElementEnable   = "(Optional) Do you want to setup Element storage for your region? (This requires more information to be collected)"
	labelElementMVIP     = "Element MVIP"
	labelElementSVIP     = "Element SVIP"
	labelElementUser     = "Element user"
	labelElementPassword = "Element password"

	labelObservabilityEnabled         = "(Optional) Do you want to setup the Observability Addon (prometheus, loki, grafana)? (This requires more information to be collected)"
	labelObservabilityArchiveLocation = "Observability Archive location (ex. https://hosted-site.com/wolverine-moni-patch.tgz or baked into the OVA at /root/wolverine-moni-patch.tgz)"

	labelIPAMProvider = "Select IP address management provider"

	labelMNodeIP          = "MNode IP"
	labelMNodePath        = "MNode path"
	labelMNodeVersion     = "MNode version"
	labelMNodeAuthHostURL = "MNode auth host URL"
	labelMNodeAuthSecret  = "MNode secret"
	labelMNodeTLSInsecure = "MNode TLS insecure"

	defaultMNodePath    = "ip"
	defaultMNodeVersion = "v1"

	// // remove, there's no reasonable default value
	// defaultVCenterURL      = "https://myvcenter.internal.local"
	// defaultVCenterUsername = "administrator@vsphere.local"

	// // remove, should be set in default_configuration.go
	// defaultNKSURL   = "https://api.nks.netapp.io/"
	// defaultProvider = provider.HCI
)

// NameAndID represent a tuple of a resource name and resource ID within vsphere
type NameAndID struct {
	Name string
	ID   string
}

func collectNetworkInformation(spec *types.ConfigSpec) {
	/*
		getIPAMProvider(spec)

			switch spec.IPAM.Provider {
			case types.MNodeIPService:
				getMNodeInfo(spec)
			case types.Infoblox:
				// Infoblox input comes from config file only for now, need a better input mechanism when running via interactive CLI
				return
			case types.DHCP:
				return
			default:
				fmt.Println(fmt.Sprintf("IPAM provider %s is not implemented, defaulting to DHCP", spec.IPAM.Provider))
				spec.IPAM.Provider = types.DHCP
			}
	*/
}

func collectVsphereInformation(spec *types.ConfigSpec) error {
	if err := getVCenterURL(spec); err != nil {
		return fmt.Errorf("unable to get vCenter url, %v", err)
	}

	if err := getVCenterUsername(spec); err != nil {
		return fmt.Errorf("unable to get vCenter username, %v", err)
	}

	if err := getVCenterPassword(spec); err != nil {
		return fmt.Errorf("unable to get vCenter password, %v", err)
	}

	client, err := NewGovmomiClient(spec.VCenterUser, spec.VCenterPassword, spec.VCenterURL)
	if err != nil {
		return fmt.Errorf("unable to get vSphere client, %v", err)
	}

	finder := find.NewFinder(client.Client, true)

	datacenters, err := finder.DatacenterList(context.TODO(), "*")
	if err != nil {
		return fmt.Errorf("unable to list datacenters, %v", err)
	}

	if spec.DatacenterID == "" {
		var datacenterlist []NameAndID
		for _, datacenter := range datacenters {
			newNaI := NameAndID{
				Name: datacenter.Name(),
				ID:   datacenter.Reference().Value,
			}
			datacenterlist = append(datacenterlist, newNaI)
		}

		if spec.DatacenterID, err = selectObject(datacenterlist, labelVCenterDatacenter); err != nil {
			return fmt.Errorf("unable to select datacenter from list, %v", err)
		}
	}

	var selectedDatacenter *object.Datacenter
	for _, datacenter := range datacenters {
		if datacenter.Reference().Value == spec.DatacenterID {
			selectedDatacenter = datacenter
			break
		}
	}

	finder.SetDatacenter(selectedDatacenter)

	if spec.ResourcePoolID == "" {
		resourcePools, err := finder.ResourcePoolList(context.TODO(), "*")
		if err != nil {
			return fmt.Errorf("unable to list resource pools, %v", err)
		}

		var resourcepoollist []NameAndID
		for _, resourcepool := range resourcePools {
			newNaI := NameAndID{
				Name: resourcepool.Name(),
				ID:   resourcepool.Reference().Value,
			}
			resourcepoollist = append(resourcepoollist, newNaI)
		}

		if spec.ResourcePoolID, err = selectObject(resourcepoollist, labelVCenterresourcepool); err != nil {
			return fmt.Errorf("unable to select resource pool from list, %v", err)
		}
	}

	if spec.DatastoreID == "" {
		datastores, err := finder.DatastoreList(context.TODO(), "*")
		if err != nil {
			return fmt.Errorf("unable to list datastores, %v", err)
		}

		var datastorelist []NameAndID
		for _, datastore := range datastores {
			newNaI := NameAndID{
				Name: datastore.Name(),
				ID:   datastore.Reference().Value,
			}
			datastorelist = append(datastorelist, newNaI)
		}

		if spec.DatastoreID, err = selectObject(datastorelist, labelVCenterDatastore); err != nil {
			return fmt.Errorf("unable to select datastore from list, %v", err)
		}
	}

	networks, err := finder.NetworkList(context.TODO(), "*")
	if err != nil {
		return fmt.Errorf("unable to list networks, %v", err)
	}

	networklist, err := getValidNetworks(client, networks)
	if err != nil {
		return fmt.Errorf("unable to filter VDS, %v", err)
	}

	if spec.ManagementNetworkID == "" {
		if spec.ManagementNetworkID, err = selectObject(networklist, labelCAPVManagementClusterNetwork); err != nil {
			return fmt.Errorf("unable to select management network, %v", err)
		}
		for _, elem := range networklist {
			if spec.ManagementNetworkID == elem.ID {
				spec.ManagementNetworkName = elem.Name
			}
		}
	}

	if spec.WorkloadNetworkID == "" {
		if spec.WorkloadNetworkID, err = selectObject(networklist, labelCAPVWorkloadClusterNetwork); err != nil {
			return fmt.Errorf("unable to select workload network, %v", err)
		}
		for _, elem := range networklist {
			if spec.WorkloadNetworkID == elem.ID {
				spec.WorkloadNetworkName = elem.Name
			}
		}
	}

	storageNetwork(spec, networklist)

	if spec.RegionName == "" {
		if spec.RegionName, err = getRegionName(); err != nil {
			return fmt.Errorf("unable to get region name, %v", err)
		}
	}

	return nil
}

func storageNetwork(spec *types.ConfigSpec, networkList []NameAndID) {

	// Filter out already selected management and workload networks
	var validStorageNetworks []NameAndID
	for _, network := range networkList {
		if network.ID == spec.ManagementNetworkID || network.ID == spec.WorkloadNetworkID {
			continue
		}
		validStorageNetworks = append(validStorageNetworks, network)
	}

	if addStorageNetwork := getBooleanWithLabel(labelAddStorageNetwork); addStorageNetwork {
		storageNetwork, err := selectObject(validStorageNetworks, labelCAPVStorageNetwork)
		if err != nil {
			log.Fatalf("Unable to select network, %v", err)
		}

		spec.StorageNetworkID = storageNetwork
		for _, elem := range networkList {
			if spec.StorageNetworkID == elem.ID {
				spec.StorageNetworkName = elem.Name
			}
		}
	}
}

func collectAdditionalConfiguration(spec *types.ConfigSpec) {
	getServiceClusterPodCIDR(spec)

	getServiceClusterServiceCIDR(spec)

	checkDisableCleanup(spec)

	disableAntiAffinity(spec)

	collectObservabilityInformation(spec)

}

func collectObservabilityInformation(spec *types.ConfigSpec) {
	//spec.OptionalConfiguration.Observability.ArchiveLocation = getInputWithLabel(labelObservabilityArchiveLocation)
}

func disableAntiAffinity(spec *types.ConfigSpec) {
	//spec.OptionalConfiguration.DisableHALoadbalancer = true
}

/*
func getIPAMProvider(spec *types.ConfigSpec) {
	if spec.IPAM.Provider != "" {
		return
	}

	IPAMProviders := registration.AvailableIPAMProviders()

	if len(IPAMProviders) == 0 {
		log.Fatal("No IPAM provider found")
	}

	if len(IPAMProviders) == 1 {
		spec.IPAM.Provider = IPAMProviders[0]
		return
	}

	prompt := promptui.Select{
		Label: labelIPAMProvider,
		Items: IPAMProviders,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		log.Fatalf("prompt failed, %v", err)
	}

	spec.IPAM.Provider = IPAMProviders[idx]
}
*/

func getMNodeInfo(spec *types.ConfigSpec) {

	getMNodeIP(spec)

	getMNodePath(spec)

	getMNodeVersion(spec)

	getMNodeAuthHostURL(spec)

	getMNodeAuthSecret(spec)

	getMNodeTLSInsecure(spec)
}

func getMNodeIP(spec *types.ConfigSpec) {
	if spec.IPAM.MNode.IP != "" {
		return
	}
	spec.IPAM.MNode.IP = getInputWithLabel(labelMNodeIP)
}

func getMNodePath(spec *types.ConfigSpec) {
	if spec.IPAM.MNode.Path != "" {
		return
	}
	spec.IPAM.MNode.Path = getInputWithLabelAndDefault(labelMNodePath, defaultMNodePath)
}

func getMNodeVersion(spec *types.ConfigSpec) {
	if spec.IPAM.MNode.Version != "" {
		return
	}
	spec.IPAM.MNode.Version = getInputWithLabelAndDefault(labelMNodeVersion, defaultMNodeVersion)
}

func getMNodeAuthHostURL(spec *types.ConfigSpec) {
	if spec.IPAM.MNode.AuthHostURL != "" {
		return
	}
	spec.IPAM.MNode.AuthHostURL = getInputWithLabel(labelMNodeAuthHostURL)
}

func getMNodeAuthSecret(spec *types.ConfigSpec) {
	if spec.IPAM.MNode.AuthSecret != "" {
		return
	}
	spec.IPAM.MNode.AuthSecret = getInputWithLabel(labelMNodeAuthSecret)
}

func getMNodeTLSInsecure(spec *types.ConfigSpec) {
	spec.IPAM.MNode.TLSInsecure = getBooleanWithLabel(labelMNodeTLSInsecure)
}

func collectElementInformation(spec *types.ConfigSpec) {
	if spec.Solidfire.Enable = setupElementStorage(); !spec.Solidfire.Enable {
		return
	}

	getSolidfireMVIP(spec)

	getSolidfireSVIP(spec)

	getSolidfireUser(spec)

	getSolidfirePassword(spec)
}

func getSolidfireMVIP(spec *types.ConfigSpec) {
	if spec.Solidfire.MVIP != "" {
		return
	}

	spec.Solidfire.MVIP = getInputWithLabel(labelElementMVIP)
}

func getSolidfireSVIP(spec *types.ConfigSpec) {
	if spec.Solidfire.SVIP != "" {
		return
	}

	spec.Solidfire.SVIP = getInputWithLabel(labelElementSVIP)
}

func getSolidfireUser(spec *types.ConfigSpec) {
	if spec.Solidfire.User != "" {
		return
	}

	spec.Solidfire.User = getInputWithLabel(labelElementUser)
}

func getSolidfirePassword(spec *types.ConfigSpec) {
	if spec.Solidfire.Password != "" {
		return
	}

	spec.Solidfire.Password = getPasswordInputWithLabel(labelElementPassword)
}

func setupElementStorage() bool {
	return getBooleanWithLabel(labelElementEnable)
}

func setupObservability() bool {
	return getBooleanWithLabel(labelObservabilityEnabled)
}

func validateURL(input string) error {
	if _, err := url.ParseRequestURI(input); err != nil {
		return errors.New("invalid URL")
	}

	return nil
}

func validURL(input string) error {
	u, err := url.Parse(input)
	if err != nil {
		return errors.New("invalid URL")
	}

	if u.Scheme == "" || u.Host == "" || u.Path == "" {
		return errors.New("invalid URL")
	}
	return nil
}

func getVCenterURL(spec *types.ConfigSpec) error {
	if cliSettings.vCenterURL != "" {
		spec.VCenterURL = cliSettings.vCenterURL
	} else if envSettings.vCenterURL != "" {
		spec.VCenterURL = envSettings.vCenterURL
	}

	if spec.VCenterURL != "" {
		return nil
	}

	prompt := promptui.Prompt{
		Label:    labelVCenterURL,
		Validate: validateURL,
	}

	var err error
	spec.VCenterURL, err = prompt.Run()
	return err
}

func getVCenterPassword(spec *types.ConfigSpec) error {
	if cliSettings.vCenterPassword != "" {
		spec.VCenterPassword = cliSettings.vCenterPassword
	} else if envSettings.vCenterPassword != "" {
		spec.VCenterPassword = envSettings.vCenterPassword
	}

	if spec.VCenterPassword != "" {
		return nil
	}

	prompt := promptui.Prompt{
		Label: labelVCenterPassword,
		Mask:  '*',
	}

	var err error
	spec.VCenterPassword, err = prompt.Run()
	return err
}

func getVCenterUsername(spec *types.ConfigSpec) error {
	if cliSettings.vCenterUser != "" {
		spec.VCenterUser = cliSettings.vCenterUser
	} else if envSettings.vCenterUser != "" {
		spec.VCenterUser = envSettings.vCenterUser
	}
	if spec.VCenterUser != "" {
		return nil
	}

	prompt := promptui.Prompt{
		Label: labelVCenterUsername,
	}

	var err error
	spec.VCenterUser, err = prompt.Run()
	return err
}

func validateRegionName(r string) error {
	reg, err := regexp.Compile("[^a-zA-Z0-9-_ ]+")
	if err != nil {
		return errors.New("Regex failed")
	}

	checker := reg.FindAllString(r, -1)
	if len(checker) != 0 {
		return fmt.Errorf("Region name contains unacceptable character(s): %s, (acceptable: ' ', '-','_')", strings.Join(checker, ","))
	}

	return nil
}

func getRegionName() (string, error) {
	prompt := promptui.Prompt{
		Label:    labelCAPVRegionName,
		Validate: validateRegionName,
	}

	return prompt.Run()
}

func getServiceClusterPodCIDR(spec *types.ConfigSpec) {
	if cliSettings.managementClusterPodCIDR != "" {
		spec.OptionalConfiguration.Cluster.KubernetesPodCidr = cliSettings.managementClusterPodCIDR
	} else if envSettings.managementClusterPodCIDR != "" {
		spec.OptionalConfiguration.Cluster.KubernetesPodCidr = envSettings.managementClusterPodCIDR
	}
}

func getServiceClusterServiceCIDR(spec *types.ConfigSpec) {
	if cliSettings.managementClusterCIDR != "" {
		spec.OptionalConfiguration.Cluster.KubernetesServiceCidr = cliSettings.managementClusterCIDR
	} else if envSettings.managementClusterCIDR != "" {
		spec.OptionalConfiguration.Cluster.KubernetesServiceCidr = envSettings.managementClusterCIDR
	}
}

func checkDisableCleanup(spec *types.ConfigSpec) {

	if cliSettings.disableCleanup {
		spec.OptionalConfiguration.DisableCleanup = cliSettings.disableCleanup
	} else if envSettings.disableCleanup {
		spec.OptionalConfiguration.DisableCleanup = envSettings.disableCleanup
	}
}

func selectObject(allObjects []NameAndID, label string) (string, error) {
	if len(allObjects) == 1 {
		return allObjects[0].ID, nil
	}

	var nameList []string
	for _, obj := range allObjects {
		nameList = append(nameList, fmt.Sprintf("%s (ID: %s)", obj.Name, obj.ID))
	}

	prompt := promptui.Select{
		Label: label,
		Items: nameList,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return allObjects[idx].ID, nil
}

func getInputWithLabel(label string) string {
	prompt := promptui.Prompt{
		Label: label,
	}

	result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed, %v", err)
	}

	return result
}

func getInputWithLabelAndDefault(label, defaultValue string) string {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
	}

	result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed, %v", err)
	}

	return result
}

// Returns true if selected answer is 'Yes'
func getBooleanWithLabel(label string) bool {
	prompt := promptui.Select{
		Label: label,
		Items: []string{"No", "Yes"},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed, %v", err)
	}

	return idx == 1
}

func getPasswordInputWithLabel(label string) string {
	prompt := promptui.Prompt{
		Label: label,
		Mask:  '*',
	}

	result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed, %v", err)
	}

	return result
}

func NewGovmomiClient(govusername string, govpassword string, govurl string) (*govmomi.Client, error) {
	ctx := context.TODO()

	nonAuthUrl, err := url.Parse(govurl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse vCenter url, %v", err)
	}
	if !strings.HasSuffix(nonAuthUrl.Path, "sdk") {
		nonAuthUrl.Path = nonAuthUrl.Path + "sdk"
	}
	authenticatedUrl, err := url.Parse(nonAuthUrl.String())
	if err != nil {
		return nil, fmt.Errorf("unable to parse vCenter url, %v", err)
	}
	authenticatedUrl.User = url.UserPassword(govusername, govpassword)

	client, err := govmomi.NewClient(ctx, nonAuthUrl, true) // insecure = true
	if err != nil {
		return nil, fmt.Errorf("unable to get vSphere client, %v", err)
	}

	if err = client.Login(ctx, authenticatedUrl.User); err != nil {
		return nil, fmt.Errorf("unable to login to vCenter, %v", err)
	}

	return client, nil
}

func getValidNetworks(client *govmomi.Client, foundNetworks []object.NetworkReference) ([]NameAndID, error) {
	var networks []NameAndID
	for _, networkRef := range foundNetworks {
		common := object.NewCommon(client.Client, networkRef.Reference())

		switch common.Reference().Type {
		case "DistributedVirtualPortgroup":
			var moNetwork mo.Network
			if err := common.Properties(context.TODO(), common.Reference(), nil, &moNetwork); err != nil {
				return networks, fmt.Errorf("unable to get properties of networks, %v", err)
			}

			newNaI := NameAndID{
				Name: moNetwork.Name,
				ID:   common.Reference().Value,
			}
			networks = append(networks, newNaI)
		case "Network":
			var moNetwork mo.Network
			if err := common.Properties(context.TODO(), common.Reference(), nil, &moNetwork); err != nil {
				return networks, fmt.Errorf("unable to get properties of networks, %v", err)
			}

			newNaI := NameAndID{
				Name: moNetwork.Name,
				ID:   common.Reference().Value,
			}
			networks = append(networks, newNaI)
		}

	}
	return networks, nil
}
