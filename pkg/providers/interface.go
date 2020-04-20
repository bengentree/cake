package providers

// Bootstrap is the interface for creating a bootstrap vm and running cluster provisioning
type Bootstrap interface {
	// Prepare setups up any needed infrastructure
	Prepare() error
	// Provision runs the management cluster creation steps
	Provision() error
	// Progress watches the cluster creation for progress
	Progress() error
	// Finalize saves any deliverables and removes any created bootstrap infrastructure
	Finalize() error
	// Events are status messages from the implementation
	Events() chan interface{}
}
