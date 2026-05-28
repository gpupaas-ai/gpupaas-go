package v1alpha1

import (
	"context"

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

type storageClient struct {
	devScopedClient
}

func newStorageClient(restClient *rest.Client, project, workspace string) *storageClient {
	return &storageClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *storageClient) Create(ctx context.Context, obj *apiv1.Storage, _ gpupaas.CreateOptions) (*apiv1.Storage, error) {
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	wire := convert.ToDevStorage(stripStatusStorage(obj), c.project, c.workspace)
	collection, _ := convert.StoragePaths(c.scope())
	var out convert.DevStorage
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
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

func (c *storageClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item := convert.StoragePaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

func stripStatusStorage(obj *apiv1.Storage) *apiv1.Storage {
	cp := *obj
	cp.Status = apiv1.StorageStatus{}
	return &cp
}

type securityGroupClient struct {
	devScopedClient
}

func newSecurityGroupClient(restClient *rest.Client, project, workspace string) *securityGroupClient {
	return &securityGroupClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *securityGroupClient) Create(ctx context.Context, obj *apiv1.SecurityGroup, _ gpupaas.CreateOptions) (*apiv1.SecurityGroup, error) {
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	wire := convert.ToDevSecurityGroup(stripStatusSecurityGroup(obj), c.project, c.workspace)
	collection, _ := convert.SecurityGroupPaths(c.scope())
	var out convert.DevSecurityGroup
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromDevSecurityGroup(&out, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *securityGroupClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.SecurityGroup, error) {
	_, item := convert.SecurityGroupPaths(c.scope())
	var out convert.DevSecurityGroup
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromDevSecurityGroup(&out, c.workspace), nil
}

func (c *securityGroupClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.SecurityGroupList, error) {
	collection, _ := convert.SecurityGroupPaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.DevSecurityGroupList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromDevSecurityGroupList(&out, c.workspace), nil
}

func (c *securityGroupClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item := convert.SecurityGroupPaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

func stripStatusSecurityGroup(obj *apiv1.SecurityGroup) *apiv1.SecurityGroup {
	cp := *obj
	cp.Status = apiv1.SecurityGroupStatus{}
	return &cp
}

type sshKeyClient struct {
	devScopedClient
}

func newSshKeyClient(restClient *rest.Client, project, workspace string) *sshKeyClient {
	return &sshKeyClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *sshKeyClient) Create(ctx context.Context, obj *apiv1.SshKey, _ gpupaas.CreateOptions) (*apiv1.SshKey, error) {
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	wire := convert.ToDevSshKey(stripStatusSshKey(obj), c.project, c.workspace)
	collection, _ := convert.SshKeyPaths(c.scope())
	var out convert.DevSshKey
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
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

func (c *sshKeyClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item := convert.SshKeyPaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

func stripStatusSshKey(obj *apiv1.SshKey) *apiv1.SshKey {
	cp := *obj
	cp.Status = apiv1.SshKeyStatus{}
	return &cp
}
