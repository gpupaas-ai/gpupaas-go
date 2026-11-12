package v1alpha1

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// bareMetalClient implements BareMetalInterface for project scope (workspace == "") or workspace scope.
type bareMetalClient struct {
	devScopedClient
}

func newBareMetalClient(restClient *rest.Client, project, workspace string) *bareMetalClient {
	return &bareMetalClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *bareMetalClient) paths() (collection func() string, item func(string) string, action func(string, string) string) {
	return convert.BareMetalPaths(c.scope())
}

// apply posts the instance to the collection; the backend creates it or updates the one of that name.
func (c *bareMetalClient) apply(ctx context.Context, obj *apiv1.BareMetal) (*apiv1.BareMetal, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "bare metal is nil"}
	}
	scoped := *obj // the caller's object is not written to: it may be reused across clients with other scopes
	scoped.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	scoped.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	obj = &scoped
	wire := convert.ToBareMetal(obj, c.project, c.workspace)
	collection, _, _ := c.paths()
	var out convert.BareMetalObject
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromBareMetal(&out, c.project, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

// Create stores a new instance. The backend upserts by name, so Create first checks that the name
// is free and fails with a 409 when it is not.
func (c *bareMetalClient) Create(ctx context.Context, obj *apiv1.BareMetal, _ gpupaas.CreateOptions) (*apiv1.BareMetal, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "bare metal is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("bare metal %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	return c.apply(ctx, obj)
}

// Update applies a changed spec to an existing instance.
func (c *bareMetalClient) Update(ctx context.Context, obj *apiv1.BareMetal, _ gpupaas.UpdateOptions) (*apiv1.BareMetal, error) {
	return c.apply(ctx, obj)
}

func (c *bareMetalClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.BareMetal, error) {
	_, item, _ := c.paths()
	var out convert.BareMetalObject
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromBareMetal(&out, c.project, c.workspace), nil
}

func (c *bareMetalClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.BareMetalList, error) {
	collection, _, _ := c.paths()
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.BareMetalObjectList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromBareMetalList(&out, c.project, c.workspace), nil
}

func (c *bareMetalClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	if opts.DryRun {
		return errDryRunUnsupported
	}
	_, item, _ := c.paths()
	err := c.rest.Delete(ctx, item(name))
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *bareMetalClient) Action(ctx context.Context, name, verb string, payload map[string]any) (*apiv1.BareMetal, error) {
	if verb == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "action is required"}
	}
	if payload == nil {
		payload = map[string]any{}
	}
	_, _, action := c.paths()
	if err := c.rest.Post(ctx, action(name, verb), payload, nil); err != nil {
		return nil, err
	}
	return c.Get(ctx, name, gpupaas.GetOptions{})
}

// ---------------------------------------------------------------------------
// BmProfile (read-only)
// ---------------------------------------------------------------------------

type bmProfileClient struct {
	rest    *rest.Client
	project string
}

func (c *bmProfileClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.BmProfile, error) {
	_, item := convert.BmProfilePaths(c.project)
	var out convert.BmProfileObject
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromBmProfile(&out, c.project), nil
}

func (c *bmProfileClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.BmProfileList, error) {
	collection, _ := convert.BmProfilePaths(c.project)
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.BmProfileObjectList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromBmProfileList(&out, c.project), nil
}
