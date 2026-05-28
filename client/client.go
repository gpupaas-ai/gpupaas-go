package client

import (
	"context"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/backend/memory"
	"github.com/gpupaas-ai/gpupaas-go/backend/remote"
	"github.com/gpupaas-ai/gpupaas-go/engine"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
	"github.com/gpupaas-ai/gpupaas-go/validation"
)

// Options configures Client construction.
type Options struct {
	Config    gpupaas.Config
	UseMemory bool
	// Verbose logs HTTP requests and responses when using the remote backend.
	Verbose bool
}

// Client is the primary SDK entry point.
type Client struct {
	Engine *engine.Engine
	Scheme *runtime.Scheme
}

// New creates a Client with the selected backend.
func New(opts Options) *Client {
	scheme := v1alpha1.DefaultScheme()
	var b backend.Backend
	if opts.UseMemory {
		b = memory.New()
	} else {
		cfg := opts.Config
		if cfg.Endpoint == "" && cfg.APIKey == "" && cfg.Token == "" {
			cfg = gpupaas.ConfigFromEnv()
		}
		cfg.Verbose = cfg.Verbose || opts.Verbose
		cfg.Normalize()
		b = remote.New(cfg, scheme)
	}
	return &Client{
		Engine: engine.New(b, scheme),
		Scheme: scheme,
	}
}

// Apply validates scope defaults and creates or updates obj.
func (c *Client) Apply(ctx context.Context, obj runtime.Object, project, workspace string) (runtime.Object, error) {
	return c.Engine.Apply(ctx, obj, validation.Options{
		DefaultProject:   project,
		DefaultWorkspace: workspace,
	})
}

// Get retrieves a single object.
func (c *Client) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error) {
	return c.Engine.Get(ctx, gvk, project, workspace, name)
}

// List returns objects for gvk within project (and optional workspace filter).
func (c *Client) List(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace string) ([]runtime.Object, error) {
	return c.Engine.List(ctx, gvk, backend.ListOptions{Project: project, Workspace: workspace})
}

// Delete removes an object by name and scope.
func (c *Client) Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error {
	return c.Engine.Delete(ctx, gvk, project, workspace, name)
}
