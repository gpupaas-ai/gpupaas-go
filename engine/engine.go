package engine

import (
	"context"
	"fmt"

	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
	"github.com/gpupaas-ai/gpupaas-go/validation"
)

// Engine orchestrates validation and backend operations.
type Engine struct {
	backend backend.Backend
	scheme  *runtime.Scheme
}

// New creates an Engine.
func New(b backend.Backend, scheme *runtime.Scheme) *Engine {
	return &Engine{backend: b, scheme: scheme}
}

// Apply validates and upserts an object.
func (e *Engine) Apply(ctx context.Context, obj runtime.Object, opts validation.Options) (runtime.Object, error) {
	if err := validation.ValidateObject(obj, opts); err != nil {
		return nil, err
	}
	return e.backend.Apply(ctx, obj)
}

// Get retrieves an object by scope and name.
func (e *Engine) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error) {
	if validation.RequiresProject(gvk.Kind) && project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return e.backend.Get(ctx, gvk, project, workspace, name)
}

// List returns objects matching gvk and list options.
func (e *Engine) List(ctx context.Context, gvk runtime.GroupVersionKind, opts backend.ListOptions) ([]runtime.Object, error) {
	if validation.RequiresProject(gvk.Kind) && opts.Project == "" {
		return nil, fmt.Errorf("project is required")
	}
	return e.backend.List(ctx, gvk, opts)
}

// Delete removes an object by scope and name.
func (e *Engine) Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if validation.RequiresProject(gvk.Kind) && project == "" {
		return fmt.Errorf("project is required")
	}
	return e.backend.Delete(ctx, gvk, project, workspace, name)
}
