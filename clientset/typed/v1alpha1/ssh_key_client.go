package v1alpha1

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

type devScopedClient struct {
	rest      *rest.Client
	project   string
	workspace string
}

func (c *devScopedClient) scope() convert.DevScope {
	return convert.DevScope{Project: c.project, Workspace: c.workspace}
}

// sshKeyClient implements SshKeyInterface for project scope (workspace == "") or workspace scope.
type sshKeyClient struct {
	devScopedClient
}

func newSshKeyClient(restClient *rest.Client, project, workspace string) *sshKeyClient {
	return &sshKeyClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

// Create stores a new key. The backend upserts by name, so Create first checks that the name is
// free and fails with a 409 when it is not.
func (c *sshKeyClient) Create(ctx context.Context, obj *apiv1.SshKey, _ gpupaas.CreateOptions) (*apiv1.SshKey, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "ssh key is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("ssh key %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	scoped := *obj // the caller's object is not written to: it may be reused across clients with other scopes
	scoped.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	scoped.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	obj = &scoped
	collection, _ := convert.SshKeyPaths(c.scope())
	var out convert.DevSshKey
	if err := c.rest.Post(ctx, collection(), convert.ToDevSshKey(obj, c.project, c.workspace), &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromDevSshKey(&out, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *sshKeyClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.SshKey, error) {
	_, item := convert.SshKeyPaths(c.scope())
	var out convert.DevSshKey
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromDevSshKey(&out, c.workspace), nil
}

func (c *sshKeyClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.SshKeyList, error) {
	collection, _ := convert.SshKeyPaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.DevSshKeyList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromDevSshKeyList(&out, c.workspace), nil
}

func (c *sshKeyClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	if opts.DryRun {
		return errDryRunUnsupported
	}
	_, item := convert.SshKeyPaths(c.scope())
	err := c.rest.Delete(ctx, item(name))
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}
