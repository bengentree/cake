package config

import (
	"github.com/netapp/cake/pkg/config/types"
	"github.com/netapp/cake/pkg/engines"
	"github.com/netapp/cake/pkg/providers"
)

// Supported Provider and Engine Types
const (
	VsphereProvider = types.ProviderType("VSPHERE")
	KVMProvider     = types.ProviderType("KVM")
	EngineRKE       = types.EngineType("RKE")
	EngineCAPI      = types.EngineType("CAPV")
)

// Spec holds information needed to provision a K8s management cluster
type Spec struct {
	ProviderType types.ProviderType  `yaml:"ProviderType" json:"providertype"`
	Provider     providers.Spec      `yaml:"Provider" json:"provider"`
	Engine       engines.MgmtCluster `yaml:"Engine" json:"engine"`
	EngineType   types.EngineType    `yaml:"EngineType" json:"enginetype"`
	Local        bool                `yaml:"Local" json:"local"`
	LogFile      string              `yaml:"LogFile" json:"logfile"`
}
