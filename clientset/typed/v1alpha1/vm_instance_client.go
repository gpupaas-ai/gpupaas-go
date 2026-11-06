package v1alpha1

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// vmInstanceClient implements VMInstanceInterface for project scope (workspace == "") or workspace scope.
type vmInstanceClient struct {
	devScopedClient
}

func newVMInstanceClient(restClient *rest.Client, project, workspace string) *vmInstanceClient {
	return &vmInstanceClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *vmInstanceClient) paths() (collection func() string, item func(string) string, action func(string, string) string) {
	return convert.VMInstancePaths(c.scope())
}

// apply posts the instance to the collection; the backend creates it or updates the one of that name.
func (c *vmInstanceClient) apply(ctx context.Context, obj *apiv1.VMInstance) (*apiv1.VMInstance, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "virtual machine is nil"}
	}
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	wire := convert.ToVMInstance(obj, c.project, c.workspace)
	collection, _, _ := c.paths()
	var out convert.VMInstanceObject
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromVMInstance(&out, c.project, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

// Create stores a new instance. The backend upserts by name, so Create first checks that the name
// is free and fails with a 409 when it is not.
func (c *vmInstanceClient) Create(ctx context.Context, obj *apiv1.VMInstance, _ gpupaas.CreateOptions) (*apiv1.VMInstance, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "virtual machine is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("virtual machine %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	return c.apply(ctx, obj)
}

// Update applies a changed spec to an existing instance.
func (c *vmInstanceClient) Update(ctx context.Context, obj *apiv1.VMInstance, _ gpupaas.UpdateOptions) (*apiv1.VMInstance, error) {
	return c.apply(ctx, obj)
}

func (c *vmInstanceClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.VMInstance, error) {
	_, item, _ := c.paths()
	var out convert.VMInstanceObject
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromVMInstance(&out, c.project, c.workspace), nil
}

func (c *vmInstanceClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.VMInstanceList, error) {
	collection, _, _ := c.paths()
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.VMInstanceObjectList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromVMInstanceList(&out, c.project, c.workspace), nil
}

func (c *vmInstanceClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	_, item, _ := c.paths()
	err := c.rest.Delete(ctx, item(name))
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *vmInstanceClient) Action(ctx context.Context, name, verb string, payload map[string]any) (*apiv1.VMInstance, error) {
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
// VMProfile (read-only)
// ---------------------------------------------------------------------------

type vmProfileClient struct {
	rest    *rest.Client
	project string
}

func (c *vmProfileClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.VMProfile, error) {
	_, item := convert.VMProfilePaths(c.project)
	var out convert.VMProfileObject
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromVMProfile(&out, c.project), nil
}

func (c *vmProfileClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.VMProfileList, error) {
	collection, _ := convert.VMProfilePaths(c.project)
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.VMProfileObjectList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromVMProfileList(&out, c.project), nil
}
