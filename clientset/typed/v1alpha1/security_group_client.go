package v1alpha1

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// securityGroupClient implements SecurityGroupInterface for project scope (workspace == "") or
// workspace scope.
type securityGroupClient struct {
	devScopedClient
}

func newSecurityGroupClient(restClient *rest.Client, project, workspace string) *securityGroupClient {
	return &securityGroupClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

// Create stores a new group. The backend upserts by name, so Create first checks that the name is
// free and fails with a 409 when it is not.
func (c *securityGroupClient) Create(ctx context.Context, obj *apiv1.SecurityGroup, _ gpupaas.CreateOptions) (*apiv1.SecurityGroup, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "security group is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("security group %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	collection, _ := convert.SecurityGroupPaths(c.scope())
	var out convert.DevSecurityGroup
	if err := c.rest.Post(ctx, collection(), convert.ToDevSecurityGroup(obj, c.project, c.workspace), &out); err != nil {
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

func (c *securityGroupClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	_, item := convert.SecurityGroupPaths(c.scope())
	err := c.rest.Delete(ctx, item(name))
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}
