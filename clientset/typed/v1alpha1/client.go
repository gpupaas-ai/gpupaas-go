package v1alpha1

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// Interface exposes v1alpha1 resource clients.
type Interface interface {
	Projects() ProjectInterface
	Workspaces(project string) WorkspaceInterface
	VirtualMachines(project string) VirtualMachineInterface
	Storages(project string) StorageInterface
	SecurityGroups(project string) SecurityGroupInterface
	SshKeys(project string) SshKeyInterface
	BaremetalMachines(project string) BaremetalMachineInterface
}

// Client implements Interface.
type Client struct {
	rest *rest.Client
}

// New creates a v1alpha1 client.
func New(restClient *rest.Client) *Client {
	return &Client{rest: restClient}
}

func (c *Client) Projects() ProjectInterface {
	return &projectClient{rest: c.rest}
}

func (c *Client) Workspaces(project string) WorkspaceInterface {
	return &workspaceClient{rest: c.rest, project: project}
}

func (c *Client) VirtualMachines(project string) VirtualMachineInterface {
	return newVirtualMachineClient(c.rest, project, "")
}

func (c *Client) Storages(project string) StorageInterface {
	return newStorageClient(c.rest, project, "")
}

func (c *Client) SecurityGroups(project string) SecurityGroupInterface {
	return newSecurityGroupClient(c.rest, project, "")
}

func (c *Client) SshKeys(project string) SshKeyInterface {
	return newSshKeyClient(c.rest, project, "")
}

func (c *Client) BaremetalMachines(project string) BaremetalMachineInterface {
	return newBaremetalMachineClient(c.rest, project)
}

type projectClient struct {
	rest *rest.Client
}

func (c *projectClient) Create(ctx context.Context, obj *apiv1.Project, _ gpupaas.CreateOptions) (*apiv1.Project, error) {
	wire := convert.ToAuthProject(obj)
	var out convert.AuthProject
	if err := c.rest.Post(ctx, convert.AuthProjectsPath, wire, &out); err != nil {
		return nil, err
	}
	return convert.FromAuthProject(&out), nil
}

func (c *projectClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.Project, error) {
	id, err := c.resolveProjectID(ctx, name, "")
	if err != nil {
		return nil, err
	}
	var out convert.AuthProject
	path := fmt.Sprintf(convert.AuthProjectPath, url.PathEscape(id))
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromAuthProject(&out), nil
}

func (c *projectClient) Update(ctx context.Context, obj *apiv1.Project, _ gpupaas.UpdateOptions) (*apiv1.Project, error) {
	id, err := c.resolveProjectID(ctx, obj.Metadata.Name, convert.ProjectIDFromObject(obj))
	if err != nil {
		return nil, err
	}
	wire := convert.ToAuthProject(obj)
	var out convert.AuthProject
	path := fmt.Sprintf(convert.AuthProjectPath, url.PathEscape(id))
	if err := c.rest.Put(ctx, path, wire, &out); err != nil {
		return nil, err
	}
	return convert.FromAuthProject(&out), nil
}

func (c *projectClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	id, err := c.resolveProjectID(ctx, name, "")
	if err != nil {
		return err
	}
	path := fmt.Sprintf(convert.AuthProjectPath, url.PathEscape(id))
	return c.rest.Delete(ctx, path)
}

func (c *projectClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.ProjectList, error) {
	path := convert.AuthProjectsPath
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var page convert.AuthProjectList
	if err := c.rest.Get(ctx, path, &page); err != nil {
		return nil, err
	}
	return convert.FromAuthProjectList(&page), nil
}

// resolveProjectID lists /auth/v1/projects/, matches name, returns auth project id.
func (c *projectClient) resolveProjectID(ctx context.Context, name, knownID string) (string, error) {
	if knownID != "" {
		return knownID, nil
	}
	path := convert.AuthProjectsPath
	for {
		var page convert.AuthProjectList
		if err := c.rest.Get(ctx, path, &page); err != nil {
			return "", err
		}
		if id, ok := convert.FindAuthProjectIDByName(&page, name); ok {
			return id, nil
		}
		path = nextAuthProjectsPath(path, page.Next)
		if path == "" {
			break
		}
	}
	return "", &gpupaas.APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("project %q not found", name),
	}
}

// nextAuthProjectsPath returns the next list path from a paginated response.
func nextAuthProjectsPath(current string, next *string) string {
	if next == nil || strings.TrimSpace(*next) == "" {
		return ""
	}
	u, err := url.Parse(*next)
	if err != nil {
		return ""
	}
	if u.Path == "" {
		return ""
	}
	nextPath := u.Path
	if !strings.HasSuffix(nextPath, "/") && strings.HasSuffix(current, "/") {
		nextPath += "/"
	}
	if u.RawQuery != "" {
		nextPath += "?" + u.RawQuery
	}
	return nextPath
}

type workspaceClient struct {
	rest    *rest.Client
	project string
}

func (c *workspaceClient) collectionPath() string {
	return fmt.Sprintf(convert.PaaSWorkspacesPath, url.PathEscape(c.project))
}

func (c *workspaceClient) itemPath(name string) string {
	return fmt.Sprintf(convert.PaaSWorkspacePath, url.PathEscape(c.project), url.PathEscape(name))
}

func (c *workspaceClient) Create(ctx context.Context, obj *apiv1.Workspace, _ gpupaas.CreateOptions) (*apiv1.Workspace, error) {
	return c.apply(ctx, obj)
}

func (c *workspaceClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.Workspace, error) {
	var out convert.PaaSWorkspace
	if err := c.rest.Get(ctx, c.itemPath(name), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspace(&out), nil
}

func (c *workspaceClient) Update(ctx context.Context, obj *apiv1.Workspace, _ gpupaas.UpdateOptions) (*apiv1.Workspace, error) {
	return c.apply(ctx, obj)
}

func (c *workspaceClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	return c.rest.Delete(ctx, c.itemPath(name))
}

func (c *workspaceClient) List(ctx context.Context, _ gpupaas.ListOptions) (*apiv1.WorkspaceList, error) {
	var out convert.PaaSWorkspaceList
	if err := c.rest.Get(ctx, c.collectionPath(), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspaceList(&out), nil
}

func (c *workspaceClient) Collaborators(workspace string) WorkspaceCollaboratorInterface {
	return &workspaceCollaboratorClient{
		rest:      c.rest,
		project:   c.project,
		workspace: workspace,
	}
}

func (c *workspaceClient) VirtualMachines(workspace string) VirtualMachineInterface {
	return newVirtualMachineClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) Storages(workspace string) StorageInterface {
	return newStorageClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) SecurityGroups(workspace string) SecurityGroupInterface {
	return newSecurityGroupClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) SshKeys(workspace string) SshKeyInterface {
	return newSshKeyClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) apply(ctx context.Context, obj *apiv1.Workspace) (*apiv1.Workspace, error) {
	wire := convert.ToPaaSWorkspace(stripStatusWorkspace(obj), c.project)
	var out convert.PaaSWorkspace
	if err := c.rest.Post(ctx, c.collectionPath(), wire, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspace(&out), nil
}

func stripStatusWorkspace(obj *apiv1.Workspace) *apiv1.Workspace {
	cp := *obj
	cp.Status = apiv1.WorkspaceStatus{}
	return &cp
}

func listQuery(opts gpupaas.ListOptions) string {
	values := url.Values{}
	if opts.Limit != "" {
		values.Set("limit", opts.Limit)
	}
	if opts.Continue != "" {
		values.Set("offset", opts.Continue)
	}
	return values.Encode()
}

type workspaceCollaboratorClient struct {
	rest      *rest.Client
	project   string
	workspace string
}

func (c *workspaceCollaboratorClient) collaboratorsPath() string {
	return fmt.Sprintf(convert.PaaSWorkspaceCollaboratorsPath, url.PathEscape(c.project), url.PathEscape(c.workspace))
}

func (c *workspaceCollaboratorClient) assignPath() string {
	return fmt.Sprintf(convert.PaaSWorkspaceAssignCollabPath, url.PathEscape(c.project), url.PathEscape(c.workspace))
}

func (c *workspaceCollaboratorClient) unassignPath() string {
	return fmt.Sprintf(convert.PaaSWorkspaceUnassignCollabPath, url.PathEscape(c.project), url.PathEscape(c.workspace))
}

func (c *workspaceCollaboratorClient) Create(ctx context.Context, obj *apiv1.WorkspaceCollaborator, _ gpupaas.CreateOptions) (*apiv1.WorkspaceCollaborator, error) {
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)

	if obj.AssignExisting() {
		wire := convert.ToPaaSWorkspaceAddCollaborators(obj, c.project, c.workspace)
		var out convert.PaaSWorkspaceAddCollaborators
		path := withSSOUsersQuery(c.assignPath(), obj.Spec.IsSSOUser)
		if err := c.rest.Post(ctx, path, wire, &out); err != nil {
			return nil, err
		}
		return &apiv1.WorkspaceCollaborator{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindWorkspaceCollaborator},
			Metadata: apiv1.ObjectMeta{
				Name:      obj.CollaboratorUsername(),
				Project:   obj.Metadata.Project,
				Workspace: obj.Metadata.Workspace,
			},
			Spec: apiv1.WorkspaceCollaboratorSpec{
				Username:  obj.CollaboratorUsername(),
				Role:      obj.Spec.ResolvedRole(),
				IsSSOUser: obj.Spec.IsSSOUser,
			},
			Status: apiv1.WorkspaceCollaboratorStatus{
				Phase: out.Status.ConditionStatus,
				Role:  obj.Spec.ResolvedRole(),
			},
		}, nil
	}

	wire := convert.ToPaaSWorkspaceCollaborator(obj, c.project, c.workspace)
	var out convert.PaaSWorkspaceCollaborator
	if err := c.rest.Post(ctx, c.collaboratorsPath(), wire, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspaceCollaborator(&out, c.workspace), nil
}

func (c *workspaceCollaboratorClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.WorkspaceCollaborator, error) {
	list, err := c.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		item := &list.Items[i]
		if item.Metadata.Name == name || item.Spec.Username == name || item.Spec.Email == name {
			return item, nil
		}
	}
	return nil, &gpupaas.APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("workspace collaborator %q not found", name),
	}
}

func (c *workspaceCollaboratorClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	wire := convert.ToPaaSWorkspaceDeleteCollaborators(c.project, c.workspace, name)
	var out convert.PaaSWorkspaceDeleteCollaborators
	path := c.unassignPath()
	if opts.SSOUser != nil {
		path = withSSOUsersQuery(path, *opts.SSOUser)
	}
	return c.rest.Post(ctx, path, wire, &out)
}

func (c *workspaceCollaboratorClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.WorkspaceCollaboratorList, error) {
	path := c.collaboratorsPath()
	if opts.SSOUsers != nil {
		path = withSSOUsersQuery(path, *opts.SSOUsers)
	}
	if q := listQuery(opts); q != "" {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		path += sep + q
	}
	var out convert.PaaSWorkspaceCollaboratorList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSWorkspaceCollaboratorList(&out, c.workspace), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func withSSOUsersQuery(path string, ssoUsers bool) string {
	if !ssoUsers {
		return path
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + "ssoUsers=true"
}

type virtualMachineClient struct {
	rest      *rest.Client
	project   string
	workspace string
}

func newVirtualMachineClient(restClient *rest.Client, project, workspace string) *virtualMachineClient {
	return &virtualMachineClient{rest: restClient, project: project, workspace: workspace}
}

func (c *virtualMachineClient) scope() convert.VMScope {
	return convert.VMScope{Project: c.project, Workspace: c.workspace}
}

func (c *virtualMachineClient) Create(ctx context.Context, obj *apiv1.VirtualMachine, _ gpupaas.CreateOptions) (*apiv1.VirtualMachine, error) {
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	obj.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	wire := convert.ToDevVirtualMachine(stripStatusVirtualMachine(obj), c.project, c.workspace)
	collection, _, _, _ := convert.VMPaths(c.scope())
	var out convert.DevVirtualMachine
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromDevVirtualMachine(&out, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *virtualMachineClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.VirtualMachine, error) {
	_, item, _, _ := convert.VMPaths(c.scope())
	var out convert.DevVirtualMachine
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromDevVirtualMachine(&out, c.workspace), nil
}

func (c *virtualMachineClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.VirtualMachineList, error) {
	collection, _, _, _ := convert.VMPaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.DevVirtualMachineList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromDevVirtualMachineList(&out, c.workspace), nil
}

func (c *virtualMachineClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item, _, _ := convert.VMPaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

func (c *virtualMachineClient) GetStatus(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.VirtualMachine, error) {
	_, _, status, _ := convert.VMPaths(c.scope())
	var out convert.DevVirtualMachine
	if err := c.rest.Get(ctx, status(name), &out); err != nil {
		return nil, err
	}
	return convert.FromDevVirtualMachine(&out, c.workspace), nil
}

func (c *virtualMachineClient) Start(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error) {
	return c.executeAction(ctx, name, "start", opts)
}

func (c *virtualMachineClient) Stop(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error) {
	return c.executeAction(ctx, name, "stop", opts)
}

func (c *virtualMachineClient) Reboot(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error) {
	return c.executeAction(ctx, name, "reboot", opts)
}

func (c *virtualMachineClient) Action(ctx context.Context, name, action string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error) {
	return c.executeAction(ctx, name, action, opts)
}

func (c *virtualMachineClient) executeAction(ctx context.Context, name, action string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error) {
	_, _, _, actionPath := convert.VMPaths(c.scope())
	payload := convert.DevVirtualMachineActionPayload{
		Variables: opts.Variables,
		Envs:      opts.Envs,
	}
	var out convert.DevVirtualMachine
	err := c.rest.Post(ctx, actionPath(name, action), payload, &out)
	if err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" || out.Status.Status != "" {
		return convert.FromDevVirtualMachine(&out, c.workspace), nil
	}
	return c.Get(ctx, name, gpupaas.GetOptions{})
}

func stripStatusVirtualMachine(obj *apiv1.VirtualMachine) *apiv1.VirtualMachine {
	cp := *obj
	cp.Status = apiv1.VirtualMachineStatus{}
	return &cp
}
