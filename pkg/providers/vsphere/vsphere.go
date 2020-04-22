package vsphere

import (
	"github.com/netapp/cake/pkg/providers"
	"github.com/netapp/cake/pkg/config"
)

type ConfigFile struct {
	providers.Spec `yaml:",inline" json:",inline" mapstructure:",squash"`
	ProviderVsphere      `yaml:",inline" json:",inline" mapstructure:",squash"`
}

type ProviderVsphere struct {
	config.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
}
