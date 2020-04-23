package capv

import (
	"github.com/netapp/cake/pkg/config/vsphere"
	"github.com/netapp/cake/pkg/engines"
)

/*
// NewMgmtCluster creates a new cluster interface with a full config from the client
func NewMgmtCluster(clusterConfig MgmtCluster) engines.Cluster {
	mc := new(MgmtCluster)
	mc = &clusterConfig
	mc.EventStream = make(chan config.Event)
	if mc.LogFile != "" {
		cmds.FileLogLocation = mc.LogFile
		os.Truncate(mc.LogFile, 0)
	}

	return mc
}
*/

type ConfigFile struct {
	//engines.Spec `yaml:",inline" json:",inline" mapstructure:",squash"`
	MgmtCluster  `yaml:",inline" json:",inline" mapstructure:",squash"`
}

// MgmtCluster spec for CAPV
type MgmtCluster struct {
	engines.MgmtCluster
	vsphere.ProviderVsphere `yaml:",inline" json:",inline" mapstructure:",squash"`
}

// SpecConvert makes the unmarshalled provider map a struct
func (m MgmtCluster) SpecConvert() error {
	//var result MgmtCluster
	//err := mapstructure.Decode(spec.Provider, &result)
	//if err != nil {
	//	return err
	//}
	//spec.Provider = result
	//return errS
	return nil
}
