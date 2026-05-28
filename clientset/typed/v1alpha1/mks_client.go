package v1alpha1

import (
	"context"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

// ---------------------------------------------------------------------------
// MKSCluster client (project-scoped)
// ---------------------------------------------------------------------------

type mksClusterClient struct {
	rest    *rest.Client
	project string
}

func newMKSClusterClient(restClient *rest.Client, project string) *mksClusterClient {
	return &mksClusterClient{rest: restClient, project: project}
}

func (c *mksClusterClient) scope() convert.PaaSMKSScope {
	return convert.PaaSMKSScope{Project: c.project}
}

func (c *mksClusterClient) Create(ctx context.Context, obj *apiv1.MKSCluster, _ gpupaas.CreateOptions) (*apiv1.MKSCluster, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "mks cluster is nil"}
	}
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	wire := convert.ToPaaSMKSCluster(stripStatusMKSCluster(obj), c.project)
	collection, _, _ := convert.MKSClusterPaths(c.scope())
	var out convert.PaaSMKSCluster
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromPaaSMKSCluster(&out), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *mksClusterClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.MKSCluster, error) {
	_, item, _ := convert.MKSClusterPaths(c.scope())
	var out convert.PaaSMKSCluster
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSCluster(&out), nil
}

func (c *mksClusterClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSClusterList, error) {
	collection, _, _ := convert.MKSClusterPaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.PaaSMKSClusterList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSClusterList(&out), nil
}

func (c *mksClusterClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item, _ := convert.MKSClusterPaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

func (c *mksClusterClient) Upgrade(ctx context.Context, name string, req *apiv1.MKSUpgradeRequest, _ gpupaas.ActionOptions) (*apiv1.MKSCluster, error) {
	if req == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "upgrade request is nil"}
	}
	payload := convert.ToPaaSMKSUpgradeRequest(req)
	return c.postClusterAction(ctx, name, "upgrade", payload)
}

func (c *mksClusterClient) ScaleNodeGroup(ctx context.Context, name string, req *apiv1.MKSScaleNodeGroupRequest, _ gpupaas.ActionOptions) (*apiv1.MKSCluster, error) {
	if req == nil || req.NodeGroupName == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "nodeGroupName is required for ScaleNodeGroup"}
	}
	payload := convert.ToPaaSMKSScaleNodeGroupRequest(req)
	return c.postClusterAction(ctx, name, "scaleNodeGroup", payload)
}

func (c *mksClusterClient) AddNodeGroup(ctx context.Context, name string, nodeGroup *apiv1.MKSNodeGroup, _ gpupaas.ActionOptions) (*apiv1.MKSCluster, error) {
	if nodeGroup == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "nodeGroup is required for AddNodeGroup"}
	}
	payload := convert.ToPaaSMKSAddNodeGroupRequest(nodeGroup)
	return c.postClusterAction(ctx, name, "addNodeGroup", payload)
}

func (c *mksClusterClient) RemoveNodeGroup(ctx context.Context, name, nodeGroupName string, _ gpupaas.ActionOptions) (*apiv1.MKSCluster, error) {
	if nodeGroupName == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "nodeGroupName is required for RemoveNodeGroup"}
	}
	payload := &convert.PaaSMKSRemoveNodeGroupRequest{NodeGroupName: nodeGroupName}
	return c.postClusterAction(ctx, name, "removeNodeGroup", payload)
}

func (c *mksClusterClient) postClusterAction(ctx context.Context, name, action string, payload interface{}) (*apiv1.MKSCluster, error) {
	_, _, subroute := convert.MKSClusterPaths(c.scope())
	var out convert.PaaSMKSCluster
	if err := c.rest.Post(ctx, subroute(name, action), payload, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromPaaSMKSCluster(&out), nil
	}
	return c.Get(ctx, name, gpupaas.GetOptions{})
}

func (c *mksClusterClient) Nodes(clusterName string) MKSNodeInterface {
	return newMKSNodeClient(c.rest, c.project, clusterName)
}

func (c *mksClusterClient) WorkerNodeGroups(clusterName string) MKSWorkerNodeGroupInterface {
	return newMKSWorkerNodeGroupClient(c.rest, c.project, clusterName)
}

func (c *mksClusterClient) AuditEvents(clusterName string) MKSAuditEventInterface {
	return newMKSAuditEventClient(c.rest, c.project, clusterName)
}

func stripStatusMKSCluster(obj *apiv1.MKSCluster) *apiv1.MKSCluster {
	cp := *obj
	cp.Status = apiv1.MKSClusterStatus{}
	return &cp
}

// ---------------------------------------------------------------------------
// MKSNode client (cluster-scoped)
// ---------------------------------------------------------------------------

type mksNodeClient struct {
	rest    *rest.Client
	project string
	cluster string
}

func newMKSNodeClient(restClient *rest.Client, project, cluster string) *mksNodeClient {
	return &mksNodeClient{rest: restClient, project: project, cluster: cluster}
}

func (c *mksNodeClient) scope() convert.PaaSMKSClusterScope {
	return convert.PaaSMKSClusterScope{Project: c.project, Cluster: c.cluster}
}

func (c *mksNodeClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.MKSNode, error) {
	_, item, _ := convert.MKSNodePaths(c.scope())
	var out convert.PaaSMKSNode
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSNode(&out), nil
}

func (c *mksNodeClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSNodeList, error) {
	collection, _, _ := convert.MKSNodePaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.PaaSMKSNodeList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSNodeList(&out), nil
}

func (c *mksNodeClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item, _ := convert.MKSNodePaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

func (c *mksNodeClient) Drain(ctx context.Context, name string, req *apiv1.MKSDrainRequest, _ gpupaas.ActionOptions) (*apiv1.MKSNode, error) {
	payload := convert.ToPaaSMKSDrainRequest(req)
	return c.postNodeAction(ctx, name, "drain", payload)
}

func (c *mksNodeClient) Cordon(ctx context.Context, name string, _ gpupaas.ActionOptions) (*apiv1.MKSNode, error) {
	return c.postNodeAction(ctx, name, "cordon", nil)
}

func (c *mksNodeClient) Uncordon(ctx context.Context, name string, _ gpupaas.ActionOptions) (*apiv1.MKSNode, error) {
	return c.postNodeAction(ctx, name, "uncordon", nil)
}

func (c *mksNodeClient) postNodeAction(ctx context.Context, name, action string, payload interface{}) (*apiv1.MKSNode, error) {
	_, _, subroute := convert.MKSNodePaths(c.scope())
	var out convert.PaaSMKSNode
	if err := c.rest.Post(ctx, subroute(name, action), payload, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromPaaSMKSNode(&out), nil
	}
	return c.Get(ctx, name, gpupaas.GetOptions{})
}

// ---------------------------------------------------------------------------
// MKSWorkerNodeGroup client (cluster-scoped)
// ---------------------------------------------------------------------------

type mksWorkerNodeGroupClient struct {
	rest    *rest.Client
	project string
	cluster string
}

func newMKSWorkerNodeGroupClient(restClient *rest.Client, project, cluster string) *mksWorkerNodeGroupClient {
	return &mksWorkerNodeGroupClient{rest: restClient, project: project, cluster: cluster}
}

func (c *mksWorkerNodeGroupClient) scope() convert.PaaSMKSClusterScope {
	return convert.PaaSMKSClusterScope{Project: c.project, Cluster: c.cluster}
}

func (c *mksWorkerNodeGroupClient) Create(ctx context.Context, obj *apiv1.MKSWorkerNodeGroup, _ gpupaas.CreateOptions) (*apiv1.MKSWorkerNodeGroup, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "worker node group is nil"}
	}
	obj.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	if obj.Spec.ClusterName == "" {
		obj.Spec.ClusterName = c.cluster
	}
	wire := convert.ToPaaSMKSWorkerNodeGroup(obj, c.project)
	collection, _ := convert.MKSWorkerNodeGroupPaths(c.scope())
	var out convert.PaaSMKSWorkerNodeGroup
	if err := c.rest.Post(ctx, collection(), wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromPaaSMKSWorkerNodeGroup(&out), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

func (c *mksWorkerNodeGroupClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.MKSWorkerNodeGroup, error) {
	_, item := convert.MKSWorkerNodeGroupPaths(c.scope())
	var out convert.PaaSMKSWorkerNodeGroup
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSWorkerNodeGroup(&out), nil
}

func (c *mksWorkerNodeGroupClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSWorkerNodeGroupList, error) {
	collection, _ := convert.MKSWorkerNodeGroupPaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.PaaSMKSWorkerNodeGroupList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSWorkerNodeGroupList(&out), nil
}

func (c *mksWorkerNodeGroupClient) Delete(ctx context.Context, name string, _ gpupaas.DeleteOptions) error {
	_, item := convert.MKSWorkerNodeGroupPaths(c.scope())
	return c.rest.Delete(ctx, item(name))
}

// ---------------------------------------------------------------------------
// MKSAuditEvent client (cluster-scoped, read-only)
// ---------------------------------------------------------------------------

type mksAuditEventClient struct {
	rest    *rest.Client
	project string
	cluster string
}

func newMKSAuditEventClient(restClient *rest.Client, project, cluster string) *mksAuditEventClient {
	return &mksAuditEventClient{rest: restClient, project: project, cluster: cluster}
}

func (c *mksAuditEventClient) scope() convert.PaaSMKSClusterScope {
	return convert.PaaSMKSClusterScope{Project: c.project, Cluster: c.cluster}
}

func (c *mksAuditEventClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSAuditEventList, error) {
	collection, _ := convert.MKSAuditEventPaths(c.scope())
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.PaaSMKSAuditEventList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSAuditEventList(&out), nil
}

func (c *mksAuditEventClient) Get(ctx context.Context, id string, _ gpupaas.GetOptions) (*apiv1.MKSAuditEvent, error) {
	_, item := convert.MKSAuditEventPaths(c.scope())
	var out convert.PaaSMKSAuditEvent
	if err := c.rest.Get(ctx, item(id), &out); err != nil {
		return nil, err
	}
	return convert.FromPaaSMKSAuditEvent(&out), nil
}
