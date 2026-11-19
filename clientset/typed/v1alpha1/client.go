package v1alpha1

import (
	"net/url"
	"strings"
	"sync"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// Interface exposes v1alpha1 resource clients.
type Interface interface {
	Workspaces(project string) WorkspaceInterface
	SshKeys(project string) SshKeyInterface
	SecurityGroups(project string) SecurityGroupInterface
	Storages(project string) StorageInterface
	MKSClusters(project string) MKSClusterInterface
	MKSClusterProfiles(project string) MKSClusterProfileInterface
	MKSNodes(project string) MKSNodeInterface
	MKSMatchmaker(project string) MKSMatchmakerInterface
	BareMetals(project string) BareMetalInterface
	BmProfiles(project string) BmProfileInterface
	VMInstances(project string) VMInstanceInterface
	VMProfiles(project string) VMProfileInterface
	ComputeInstances(project string) ComputeInstanceInterface
}

// Client implements Interface.
type Client struct {
	rest *rest.Client
}

// New creates a v1alpha1 client.
func New(restClient *rest.Client) *Client {
	return &Client{rest: restClient}
}

func (c *Client) Workspaces(project string) WorkspaceInterface {
	return &workspaceClient{rest: c.rest, project: project}
}

func (c *Client) SshKeys(project string) SshKeyInterface {
	return newSshKeyClient(c.rest, project, "")
}

func (c *Client) SecurityGroups(project string) SecurityGroupInterface {
	return newSecurityGroupClient(c.rest, project, "")
}

func (c *Client) Storages(project string) StorageInterface {
	return newStorageClient(c.rest, project, "")
}

func (c *Client) MKSClusters(project string) MKSClusterInterface {
	return newMKSClusterClient(c.rest, project, "")
}

func (c *Client) MKSClusterProfiles(project string) MKSClusterProfileInterface {
	return &mksClusterProfileClient{rest: c.rest, project: project}
}

func (c *Client) MKSNodes(project string) MKSNodeInterface {
	return &mksNodeClient{rest: c.rest, project: project}
}

func (c *Client) MKSMatchmaker(project string) MKSMatchmakerInterface {
	return &mksMatchmakerClient{rest: c.rest, project: project}
}

func (c *Client) BareMetals(project string) BareMetalInterface {
	return newBareMetalClient(c.rest, project, "")
}

func (c *Client) BmProfiles(project string) BmProfileInterface {
	return &bmProfileClient{rest: c.rest, project: project}
}

func (c *Client) VMInstances(project string) VMInstanceInterface {
	return newVMInstanceClient(c.rest, project, "")
}

func (c *Client) VMProfiles(project string) VMProfileInterface {
	return &vmProfileClient{rest: c.rest, project: project}
}

func (c *Client) ComputeInstances(project string) ComputeInstanceInterface {
	return newComputeInstanceClient(c.rest, project, "")
}

// workspaceClient hands out the clients of the resources that live in one workspace of a project.
type workspaceClient struct {
	rest    *rest.Client
	project string
}

func (c *workspaceClient) MKSClusters(workspace string) MKSClusterInterface {
	return newMKSClusterClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) BareMetals(workspace string) BareMetalInterface {
	return newBareMetalClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) VMInstances(workspace string) VMInstanceInterface {
	return newVMInstanceClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) ComputeInstances(workspace string) ComputeInstanceInterface {
	return newComputeInstanceClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) SshKeys(workspace string) SshKeyInterface {
	return newSshKeyClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) SecurityGroups(workspace string) SecurityGroupInterface {
	return newSecurityGroupClient(c.rest, c.project, workspace)
}

func (c *workspaceClient) Storages(workspace string) StorageInterface {
	return newStorageClient(c.rest, c.project, workspace)
}

func listQuery(opts gpupaas.ListOptions) string {
	values := url.Values{}
	if opts.Limit != "" {
		values.Set("limit", opts.Limit)
	}
	if opts.Continue != "" {
		values.Set("offset", opts.Continue)
	}
	if opts.ServiceType != "" {
		values.Set("serviceType", opts.ServiceType)
	}
	return values.Encode()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// errDryRunUnsupported is returned by Delete when DeleteOptions.DryRun is set: the API has no dry-run mode, and
// carrying on would delete the object for real.
var errDryRunUnsupported = &gpupaas.APIError{StatusCode: 400, Message: "dry run is not supported for delete: the request was not sent"}

// createLocks serializes the check-then-create of one object name within a process. The backend upserts by name, so
// without it two concurrent Create calls for one name both see "not found" and the second silently overwrites the
// first. The lock does not cover another process: only the backend can refuse that (MKS clusters ask it to).
var createLocks sync.Map

func lockCreate(kind, project, workspace, name string) func() {
	key := strings.Join([]string{kind, project, workspace, name}, "\x00")
	m, _ := createLocks.LoadOrStore(key, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
