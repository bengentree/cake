package config

import (
	"github.com/netapp/cake/pkg/config/types"
	"github.com/netapp/cake/pkg/engine"
	"github.com/netapp/cake/pkg/provider"
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
	ProviderType types.ProviderType `yaml:"ProviderType" json:"providertype"`
	Provider     provider.Spec      `yaml:"Provider" json:"provider"`
	Engine       engine.MgmtCluster `yaml:"Engine" json:"engine"`
	EngineType   types.EngineType   `yaml:"EngineType" json:"enginetype"`
	Local        bool               `yaml:"Local" json:"local"`
	LogFile      string             `yaml:"LogFile" json:"logfile"`
}
