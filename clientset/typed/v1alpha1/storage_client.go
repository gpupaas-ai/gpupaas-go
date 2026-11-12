package v1alpha1

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// storageClient implements StorageInterface for project scope (workspace == "") or workspace scope.
type storageClient struct {
	devScopedClient
}

func newStorageClient(restClient *rest.Client, project, workspace string) *storageClient {
	return &storageClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

// Create stores a new volume. The backend upserts by name, so Create first checks that the name is free
// and fails with a 409 when it is not. Provisioning is asynchronous: the returned record is usually still
// in progress (see WaitForStorage).
func (c *storageClient) Create(ctx context.Context, obj *apiv1.Storage, _ gpupaas.CreateOptions) (*apiv1.Storage, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "storage is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("storage %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	return c.upsert(ctx, obj)
}

// Update sends the whole spec again; the backend applies it to the existing volume (upsert by name).
func (c *storageClient) Update(ctx context.Context, obj *apiv1.Storage, _ gpupaas.UpdateOptions) (*apiv1.Storage, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "storage is nil"}
	}
	return c.upsert(ctx, obj)
}

func (c *storageClient) upsert(ctx context.Context, obj *apiv1.Storage) (*apiv1.Storage, error) {
	scoped := *obj // the caller's object is not written to: it may be reused across clients with other scopes
	scoped.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	scoped.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	obj = &scoped
	collection, _ := convert.StoragePaths(c.scope())
	var out convert.DevStorage
	if err := c.rest.Post(ctx, collection(), convert.ToDevStorage(obj, c.project, c.workspace), &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromDevStorage(&out, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *storageClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.Storage, error) {
	_, item := convert.StoragePaths(c.scope())
	var out convert.DevStorage
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromDevStorage(&out, c.workspace), nil
}

func (c *storageClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.StorageList, error) {
	collection, _ := convert.StoragePaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.DevStorageList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromDevStorageList(&out, c.workspace), nil
}

func (c *storageClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	if opts.DryRun {
		return errDryRunUnsupported
	}
	_, item := convert.StoragePaths(c.scope())
	err := c.rest.Delete(ctx, item(name))
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}
