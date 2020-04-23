package vsphere

import (
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/providers"
)

type ConfigFile struct {
	providers.Spec `yaml:",inline" json:",inline" mapstructure:",squash"`
	ProviderVsphere      `yaml:",inline" json:",inline" mapstructure:",squash"`
}

type ProviderVsphere struct {
	vsphere.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
}
