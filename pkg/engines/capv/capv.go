package capv

import (
	"github.com/netapp/cake/pkg/config/cluster"
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engines"
)

// MgmtCluster spec for CAPV
type MgmtCluster struct {
	engines.MgmtCluster     `yaml:",inline" json:",inline" mapstructure:",squash"`
	vsphere.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
	cluster.CAPIConfig      `yaml:",inline" json:",inline" mapstructure:",squash"`
}
