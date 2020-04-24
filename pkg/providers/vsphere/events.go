package vsphere

import "github.com/netapp/cake/pkg/config/events"

// Events returns the channel of progress messages
func (v *ProviderVsphere) Events() chan events.Event {
	return v.EventStream
}
