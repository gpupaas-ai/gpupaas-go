package v1alpha1

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

// The dedicated day-2, status and audit routes of a cluster. All of them address the project
// route, whatever the scope of the client: cluster names are unique per organization.

func (c *mksClusterClient) sub(cluster, tail string) string {
	return fmt.Sprintf(convert.MKSClusterSubPath, url.PathEscape(c.project), url.PathEscape(cluster), tail)
}

func needCluster(cluster string) error {
	if cluster == "" {
		return &gpupaas.APIError{StatusCode: 400, Message: "cluster name is required"}
	}
	return nil
}

// clusterAnswer turns the platform's answer to a day-2 call into a cluster. The answer is the
// cluster; if it carries no name the cluster is read again.
func (c *mksClusterClient) clusterAnswer(ctx context.Context, cluster string, out *convert.MKSClusterObject) (*apiv1.MKSCluster, error) {
	if out.Metadata.Name != "" {
		return convert.FromMKSCluster(out, c.project, c.workspace), nil
	}
	return c.Get(ctx, cluster, gpupaas.GetOptions{})
}

func (c *mksClusterClient) AddNodePool(ctx context.Context, cluster string, pool apiv1.MKSNodeGroup) (*apiv1.MKSCluster, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	if pool.Name == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "node pool name is required"}
	}
	var out convert.MKSClusterObject
	if err := c.rest.Post(ctx, c.sub(cluster, "nodepools"), pool, &out); err != nil {
		return nil, err
	}
	return c.clusterAnswer(ctx, cluster, &out)
}

func (c *mksClusterClient) RemoveNodePool(ctx context.Context, cluster, pool string) (*apiv1.MKSCluster, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	if pool == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "node pool name is required"}
	}
	var out convert.MKSClusterObject
	path := fmt.Sprintf(convert.MKSClusterNodePoolPath, url.PathEscape(c.project), url.PathEscape(cluster), url.PathEscape(pool))
	if err := c.rest.DeleteInto(ctx, path, &out); err != nil {
		return nil, err
	}
	return c.clusterAnswer(ctx, cluster, &out)
}

func (c *mksClusterClient) Upgrade(ctx context.Context, cluster string, req apiv1.MKSUpgradeRequest) (*apiv1.MKSCluster, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	if req.K8sVersion == "" && req.Blueprint == "" && req.BlueprintVersion == "" && req.CNI == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "an upgrade needs a Kubernetes version, a blueprint or a CNI"}
	}
	var out convert.MKSClusterObject
	if err := c.rest.Post(ctx, c.sub(cluster, "upgrade"), req, &out); err != nil {
		return nil, err
	}
	return c.clusterAnswer(ctx, cluster, &out)
}

func (c *mksClusterClient) UpdateMetadata(ctx context.Context, cluster string, patch apiv1.MKSMetadataPatch) (*apiv1.MKSCluster, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	var out convert.MKSClusterObject
	if err := c.rest.Patch(ctx, c.sub(cluster, "metadata"), patch, &out); err != nil {
		return nil, err
	}
	return c.clusterAnswer(ctx, cluster, &out)
}

func (c *mksClusterClient) Redeploy(ctx context.Context, cluster string) (*apiv1.MKSCluster, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	var out convert.MKSClusterObject
	if err := c.rest.Post(ctx, c.sub(cluster, "redeploy"), nil, &out); err != nil {
		return nil, err
	}
	return c.clusterAnswer(ctx, cluster, &out)
}

func (c *mksClusterClient) Status(ctx context.Context, cluster string) (*apiv1.MKSClusterStatusView, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	var out apiv1.MKSClusterStatusView
	if err := c.rest.Get(ctx, c.sub(cluster, "status"), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *mksClusterClient) AuditEvents(ctx context.Context, cluster string, f MKSAuditEventFilter) (*apiv1.MKSAuditEventList, error) {
	if err := needCluster(cluster); err != nil {
		return nil, err
	}
	q := url.Values{}
	for k, v := range map[string]string{"action": f.Action, "result": f.Result, "resourceType": f.ResourceType, "actor": f.Actor} {
		if v != "" {
			q.Set(k, v)
		}
	}
	if !f.From.IsZero() {
		q.Set("from", f.From.UTC().Format(time.RFC3339))
	}
	if !f.To.IsZero() {
		q.Set("to", f.To.UTC().Format(time.RFC3339))
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.Offset > 0 {
		q.Set("offset", strconv.Itoa(f.Offset))
	}
	path := c.sub(cluster, "auditevents")
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out apiv1.MKSAuditEventList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------------------------------------------------------------------
// Kubernetes SKU discovery
// ---------------------------------------------------------------------------

func (c *mksClusterProfileClient) sub(name, tail string) string {
	return fmt.Sprintf(convert.MKSClusterProfileSub, url.PathEscape(c.project), url.PathEscape(name), tail)
}

func (c *mksClusterProfileClient) CreateOptions(ctx context.Context, name string) (*apiv1.MKSCreateOptions, error) {
	if name == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "profile name is required"}
	}
	raw, err := c.rest.GetRaw(ctx, c.sub(name, "createoptions"))
	if err != nil {
		return nil, err
	}
	var out apiv1.MKSCreateOptions
	if err := decodeLoose(raw, &out); err != nil {
		return nil, fmt.Errorf("decode create options: %w", err)
	}
	return &out, nil
}

func (c *mksClusterProfileClient) AvailableCapacity(ctx context.Context, name string) (*apiv1.MKSAvailableCapacity, error) {
	if name == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "profile name is required"}
	}
	var out apiv1.MKSAvailableCapacity
	if err := c.rest.Get(ctx, c.sub(name, "availablecapacity"), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------------------------------------------------------------------
// Node operations
// ---------------------------------------------------------------------------

func nodeVerbPath(project, cluster, hostname, verb string) string {
	return fmt.Sprintf(convert.MKSClusterNodeVerbPath, url.PathEscape(project), url.PathEscape(hostname), verb) +
		"?" + url.Values{"cluster_name": {cluster}}.Encode()
}

func needNode(cluster, hostname string) error {
	if cluster == "" || hostname == "" {
		return &gpupaas.APIError{StatusCode: 400, Message: "cluster and node names are required"}
	}
	return nil
}

func (c *mksNodeClient) nodeAction(ctx context.Context, cluster, hostname, verb string) error {
	if err := needNode(cluster, hostname); err != nil {
		return err
	}
	return c.rest.Post(ctx, nodeVerbPath(c.project, cluster, hostname, verb), map[string]any{"clusterName": cluster}, nil)
}

func (c *mksNodeClient) CordonNode(ctx context.Context, cluster, hostname string) error {
	return c.nodeAction(ctx, cluster, hostname, "cordon")
}

func (c *mksNodeClient) UncordonNode(ctx context.Context, cluster, hostname string) error {
	return c.nodeAction(ctx, cluster, hostname, "uncordon")
}

func (c *mksNodeClient) DrainNode(ctx context.Context, cluster, hostname string) error {
	return c.nodeAction(ctx, cluster, hostname, "drain")
}

func (c *mksNodeClient) DeleteNode(ctx context.Context, cluster, hostname string) error {
	if err := needNode(cluster, hostname); err != nil {
		return err
	}
	path := fmt.Sprintf(convert.MKSClusterNodePath, url.PathEscape(c.project), url.PathEscape(hostname)) +
		"?" + url.Values{"cluster_name": {cluster}}.Encode()
	return c.rest.Delete(ctx, path)
}

func (c *mksNodeClient) UpdateNodeMetadata(ctx context.Context, cluster, hostname string, patch MKSNodeMetadataPatch) error {
	if err := needNode(cluster, hostname); err != nil {
		return err
	}
	body := map[string]any{}
	if len(patch.SetLabels) > 0 {
		body["setLabels"] = patch.SetLabels
	}
	if len(patch.RemoveLabels) > 0 {
		body["removeLabels"] = patch.RemoveLabels
	}
	if len(patch.SetAnnotations) > 0 {
		body["setAnnotations"] = patch.SetAnnotations
	}
	if len(patch.RemoveAnnotations) > 0 {
		body["removeAnnotations"] = patch.RemoveAnnotations
	}
	if len(body) == 0 {
		return &gpupaas.APIError{StatusCode: 400, Message: "a node metadata change must set or remove something"}
	}
	return c.rest.Put(ctx, nodeVerbPath(c.project, cluster, hostname, "metadata"), body, nil)
}

func (c *mksNodeClient) UpdateNodeTaints(ctx context.Context, cluster, hostname string, taints []apiv1.MKSTaint) error {
	if err := needNode(cluster, hostname); err != nil {
		return err
	}
	if taints == nil {
		taints = []apiv1.MKSTaint{} // an empty list clears the taints; it must not be left out
	}
	return c.rest.Put(ctx, nodeVerbPath(c.project, cluster, hostname, "taints"), map[string]any{"taints": taints}, nil)
}
