package capv

// Events returns the channel of progress messages
func (m *MgmtCluster) Events() chan interface{} {
	return m.events
}
