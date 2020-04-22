package capv

import (
	"github.com/netapp/cake/pkg/config"
	"github.com/netapp/cake/pkg/engines"
)

// Events returns the channel of progress messages
func (m MgmtCluster) Events(spec *engines.Spec) chan config.Event {
	return spec.EventStream
}
