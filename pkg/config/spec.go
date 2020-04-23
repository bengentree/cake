package config

import (
	"github.com/netapp/cake/pkg/engines"
	"github.com/netapp/cake/pkg/providers"
)

// ProviderType for available providers
type ProviderType string

// EngineType for available engines
type EngineType string

// Supported Provider and Engine Types
const (
	VsphereProvider = ProviderType("VSPHERE")
	KVMProvider     = ProviderType("KVM")
	EngineRKE       = EngineType("RKE")
	EngineCAPI      = EngineType("CAPI")
)

// Spec holds information needed to provision a K8s management cluster
type Spec struct {
	ProviderType ProviderType        `yaml:"ProviderType" json:"providertype"`
	Provider     providers.Spec      `yaml:"Provider" json:"provider"`
	Engine       engines.MgmtCluster `yaml:"Engine" json:"engine"`
	EngineType   EngineType          `yaml:"EngineType" json:"enginetype"`
	Local        bool                `yaml:"Local" json:"local"`
	LogFile      string              `yaml:"LogFile" json:"logfile"`
}
