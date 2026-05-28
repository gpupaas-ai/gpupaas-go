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
	DryRun         bool
	IgnoreNotFound bool
	// SSOUser sets ssoUsers query param when removing workspace collaborators.
	SSOUser *bool
}

// GetOptions configures get operations.
type GetOptions struct{}

// ListOptions configures list operations.
type ListOptions struct {
	Limit         string
	Continue      string
	LabelSelector string
	// SSOUsers filters workspace collaborators to SSO users when true (ssoUsers query param).
	SSOUsers *bool
}

// ActionOptions configures virtual machine action requests (start, stop, etc.).
type ActionOptions struct {
	Variables []map[string]string
	Envs      []map[string]string
}
