package capv

import (
	"github.com/netapp/cake/pkg/config/events"
)

// Events returns the channel of progress messages
func (m MgmtCluster) Events() (chan events.Event) {
	return m.EventStream
}
