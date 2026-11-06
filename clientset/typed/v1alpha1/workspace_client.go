package v1alpha1

import (
	"context"
	"fmt"
	"net/url"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func (c *workspaceClient) collectionPath() string {
	return fmt.Sprintf(convert.PaaSWorkspacesPath, url.PathEscape(c.project))
}

func (c *workspaceClient) itemPath(name string) string {
	return fmt.Sprintf(convert.PaaSWorkspacePath, url.PathEscape(c.project), url.PathEscape(name))
}

// apply posts the workspace to the collection; the backend creates it or updates the one of that name.
func (c *workspaceClient) apply(ctx context.Context, obj *apiv1.Workspace) (*apiv1.Workspace, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "workspace is nil"}
	}
	var out convert.PaaSWorkspace
	if err := c.rest.Post(ctx, c.collectionPath(), convert.ToPaaSWorkspace(obj, c.project), &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromPaaSWorkspace(&out), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

// Create stores a new workspace. The backend upserts by name, so Create first checks that the name
// is free and fails with a 409 when it is not.
func (c *workspaceClient) Create(ctx context.Context, obj *apiv1.Workspace, _ gpupaas.CreateOptions) (*apiv1.Workspace, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "workspace is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("workspace %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	return c.apply(ctx, obj)
}

// Update applies changed metadata or spec to an existing workspace.
func (c *workspaceClient) Update(ctx context.Context, obj *apiv1.Workspace, _ gpupaas.UpdateOptions) (*apiv1.Workspace, error) {
	return c.apply(ctx, obj)
}

func (c *workspaceClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.Workspace, error) {
	var out convert.PaaSWorkspace
	if err := c.rest.Get(ctx, c.itemPath(name), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspace(&out), nil
}

func (c *workspaceClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.WorkspaceList, error) {
	path := c.collectionPath()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.PaaSWorkspaceList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspaceList(&out), nil
}

func (c *workspaceClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	err := c.rest.Delete(ctx, c.itemPath(name))
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}
