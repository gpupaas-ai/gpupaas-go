package backend

import (
	"context"

	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// ListOptions filters list results.
type ListOptions struct {
	Project   string
	Workspace string
	// SSOUsers filters workspace collaborators when listing (ssoUsers query param).
	SSOUsers *bool
}

// Backend persists and retrieves API objects.
type Backend interface {
	Apply(ctx context.Context, obj runtime.Object) (runtime.Object, error)
	Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error)
	List(ctx context.Context, gvk runtime.GroupVersionKind, opts ListOptions) ([]runtime.Object, error)
	Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error
}
