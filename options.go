package gpupaas

// CreateOptions configures create operations.
type CreateOptions struct {
	DryRun bool
}

// UpdateOptions configures update operations.
type UpdateOptions struct {
	DryRun bool
}

// DeleteOptions configures delete operations.
type DeleteOptions struct {
	// DryRun is not supported by Delete: setting it returns an error and nothing is deleted.
	DryRun         bool
	IgnoreNotFound bool
	// Force removes the record even when its infrastructure cannot be torn down. Only MKS clusters
	// support it; the other kinds ignore it (DELETE ...?force=true): use it only for a cluster that failed before anything
	// was provisioned, or when the infrastructure is already gone.
	Force bool
}

// GetOptions configures get operations.
type GetOptions struct{}

// ListOptions configures list operations.
type ListOptions struct {
	Limit         string
	Continue      string
	LabelSelector string
	// ServiceType narrows a storage list to one family: "high-speed-storage" or "object-storage". The
	// controller accepts the filter although its API document does not list it. Other kinds ignore it.
	ServiceType string
}
