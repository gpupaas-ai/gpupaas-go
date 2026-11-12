package v1alpha1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
	"github.com/gpupaas-ai/gpupaas-go/rest"
	"gopkg.in/yaml.v3"
)

// mksClusterClient implements MKSClusterInterface for project scope (workspace == "") or
// workspace scope.
type mksClusterClient struct {
	devScopedClient
}

func newMKSClusterClient(restClient *rest.Client, project, workspace string) *mksClusterClient {
	return &mksClusterClient{devScopedClient{rest: restClient, project: project, workspace: workspace}}
}

func (c *mksClusterClient) paths() (collection func() string, item func(string) string) {
	return convert.MKSClusterPaths(c.scope())
}

// failOnExistsQuery makes the backend answer 409 instead of updating when a cluster of that name exists.
const failOnExistsQuery = "?fail-on-exists=true"

// apply posts the cluster to the collection; the backend creates it or updates the existing
// one of the same name. With failOnExists it only creates: the backend refuses (409) a name that is taken.
func (c *mksClusterClient) apply(ctx context.Context, obj *apiv1.MKSCluster, failOnExists bool) (*apiv1.MKSCluster, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "mks cluster is nil"}
	}
	scoped := *obj // the caller's object is not written to: it may be reused across clients with other scopes
	scoped.Metadata.Project = firstNonEmpty(obj.Metadata.Project, c.project)
	scoped.Metadata.Workspace = firstNonEmpty(obj.Metadata.Workspace, c.workspace)
	obj = &scoped
	wire := convert.ToMKSCluster(obj, c.project, c.workspace)
	collection, _ := c.paths()
	var out convert.MKSClusterObject
	path := collection()
	if failOnExists {
		path += failOnExistsQuery
	}
	if err := c.rest.Post(ctx, path, wire, &out); err != nil {
		return nil, err
	}
	if out.Metadata.Name != "" {
		return convert.FromMKSCluster(&out, c.project, c.workspace), nil
	}
	return c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{})
}

// Create stores a new cluster. The backend upserts by name unless asked not to, so Create sends
// fail-on-exists (an atomic 409 for a taken name) and also checks the name first, because a controller
// that does not know the parameter would update the existing cluster instead.
func (c *mksClusterClient) Create(ctx context.Context, obj *apiv1.MKSCluster, _ gpupaas.CreateOptions) (*apiv1.MKSCluster, error) {
	if obj == nil {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "mks cluster is nil"}
	}
	if _, err := c.Get(ctx, obj.Metadata.Name, gpupaas.GetOptions{}); err == nil {
		return nil, &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("mks cluster %q already exists", obj.Metadata.Name)}
	} else if !gpupaas.IsNotFound(err) {
		return nil, err
	}
	return c.apply(ctx, obj, true)
}

// Update applies a changed spec to an existing cluster.
func (c *mksClusterClient) Update(ctx context.Context, obj *apiv1.MKSCluster, _ gpupaas.UpdateOptions) (*apiv1.MKSCluster, error) {
	return c.apply(ctx, obj, false)
}

func (c *mksClusterClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.MKSCluster, error) {
	_, item := c.paths()
	var out convert.MKSClusterObject
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromMKSCluster(&out, c.project, c.workspace), nil
}

func (c *mksClusterClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSClusterList, error) {
	collection, _ := c.paths()
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.MKSClusterObjectList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromMKSClusterList(&out, c.project, c.workspace), nil
}

func (c *mksClusterClient) Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error {
	if opts.DryRun {
		return errDryRunUnsupported
	}
	_, item := c.paths()
	path := item(name)
	if opts.Force {
		path += "?force=true"
	}
	err := c.rest.Delete(ctx, path)
	if err != nil && opts.IgnoreNotFound && gpupaas.IsNotFound(err) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// ClusterProfile (read-only)
// ---------------------------------------------------------------------------

type mksClusterProfileClient struct {
	rest    *rest.Client
	project string
}

func (c *mksClusterProfileClient) Get(ctx context.Context, name string, _ gpupaas.GetOptions) (*apiv1.MKSClusterProfile, error) {
	_, item := convert.MKSClusterProfilePaths(c.project)
	var out convert.MKSClusterProfileObject
	if err := c.rest.Get(ctx, item(name), &out); err != nil {
		return nil, err
	}
	return convert.FromMKSClusterProfile(&out, c.project), nil
}

func (c *mksClusterProfileClient) List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSClusterProfileList, error) {
	collection, _ := convert.MKSClusterProfilePaths(c.project)
	path := collection()
	if q := listQuery(opts); q != "" {
		path += "?" + q
	}
	var out convert.MKSClusterProfileObjectList
	if err := c.rest.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return convert.FromMKSClusterProfileList(&out, c.project), nil
}

// ---------------------------------------------------------------------------
// ClusterNode (read-only)
// ---------------------------------------------------------------------------

type mksNodeClient struct {
	rest    *rest.Client
	project string
}

func (c *mksNodeClient) ListForCluster(ctx context.Context, cluster string) (*apiv1.MKSNodeList, error) {
	if cluster == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "cluster name is required to list nodes"}
	}
	base := fmt.Sprintf(convert.MKSClusterNodesPath, url.PathEscape(c.project))
	// A large cluster has more nodes than one response holds, and the server may cap the page size
	// below ours, so pages are read by offset until the reported total is reached.
	var all convert.MKSNodeObjectList
	for page := 0; page < maxNodePages; page++ {
		q := url.Values{"cluster_name": {cluster}, "limit": {strconv.Itoa(nodePageSize)}, "offset": {strconv.Itoa(len(all.Items))}}
		var out convert.MKSNodeObjectList
		if err := c.rest.Get(ctx, base+"?"+q.Encode(), &out); err != nil {
			return nil, err
		}
		if page == 0 {
			all.APIVersion, all.Kind, all.Metadata = out.APIVersion, out.Kind, out.Metadata
		}
		// A server that ignores offset would return the first page again and again: stop on a repeated first node.
		if page > 0 && len(out.Items) > 0 && len(all.Items) > 0 && out.Items[0].Metadata.Name == all.Items[0].Metadata.Name {
			break
		}
		all.Items = append(all.Items, out.Items...)
		got, total := int64(len(all.Items)), out.Metadata.Count
		// Without a reported total a short page is not the last one (the server may cap pages below ours), so the
		// list ends at an empty page.
		if len(out.Items) == 0 || (total > 0 && got >= total) {
			break
		}
	}
	return convert.FromMKSNodeList(&all, c.project), nil
}

const (
	// nodePageSize is how many nodes one request asks for.
	nodePageSize = 100
	// maxNodePages bounds the pages read for one cluster (50,000 nodes).
	maxNodePages = 500
)

// kubeconfigPath is the sentry service's route for user kubeconfigs; it is not part of
// paas.k8smgmt.io/v3. The console asks it for the cluster by name.
const kubeconfigPath = "/v2/sentry/kubeconfig/user"

// Kubeconfig downloads the user kubeconfig of the named cluster, as the console's "Download
// Kubeconfig" does. The platform answers 500 while the cluster is not running and registered.
func (c *mksClusterClient) Kubeconfig(ctx context.Context, name string) ([]byte, error) {
	// A cluster that does not exist is a plain not found, not sentry's generic 500.
	if _, err := c.Get(ctx, name, gpupaas.GetOptions{}); err != nil {
		return nil, err
	}
	query := url.Values{"opts.selector": {"rafay.dev/clusterName=" + name}}
	raw, err := c.rest.GetRaw(ctx, kubeconfigPath+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	raw = decodeKubeconfig(raw)
	if err := checkKubeconfig(raw, name); err != nil {
		return nil, err
	}
	return raw, nil
}

// checkKubeconfig accepts a kubeconfig that holds exactly one cluster. sentry looks the cluster up
// by name only, so a name that exists in several projects would otherwise hand back another
// cluster's credentials, or a merged file with several clusters.
func checkKubeconfig(raw []byte, name string) error {
	var cfg struct {
		Clusters []struct {
			Name string `yaml:"name"`
		} `yaml:"clusters"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil || len(cfg.Clusters) == 0 {
		return &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("the platform returned no kubeconfig for cluster %q", name)}
	}
	if len(cfg.Clusters) > 1 {
		return &gpupaas.APIError{StatusCode: 409, Message: fmt.Sprintf("the platform returned %d clusters for the name %q: the name is not unique, so the kubeconfig is refused rather than risk another cluster's credentials", len(cfg.Clusters), name)}
	}
	return nil
}

// poolPath is the route of an operation on a worker pool: nodepools/{pool}/{verb}?cluster_name=.
func poolPath(project, cluster, pool, verb string) string {
	return fmt.Sprintf(convert.MKSNodePoolPath, url.PathEscape(project), url.PathEscape(pool), verb) + "?" + url.Values{"cluster_name": {cluster}}.Encode()
}

func (c *mksNodeClient) poolOp(ctx context.Context, cluster, pool, verb string, body map[string]any) (*MKSPoolOperationResult, error) {
	if cluster == "" || pool == "" {
		return nil, &gpupaas.APIError{StatusCode: 400, Message: "cluster and pool names are required"}
	}
	body["clusterName"] = cluster
	var out MKSPoolOperationResult
	if err := c.rest.Post(ctx, poolPath(c.project, cluster, pool, verb), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *mksNodeClient) CordonPool(ctx context.Context, cluster, pool string) (*MKSPoolOperationResult, error) {
	return c.poolOp(ctx, cluster, pool, "cordon", map[string]any{})
}

func (c *mksNodeClient) UncordonPool(ctx context.Context, cluster, pool string) (*MKSPoolOperationResult, error) {
	return c.poolOp(ctx, cluster, pool, "uncordon", map[string]any{})
}

func (c *mksNodeClient) DrainPool(ctx context.Context, cluster, pool string, opts MKSDrainOptions) (*MKSPoolOperationResult, error) {
	body := map[string]any{}
	if opts.Force != nil {
		body["force"] = *opts.Force
	}
	if opts.IgnoreDaemonsets != nil {
		body["ignoreDaemonsets"] = *opts.IgnoreDaemonsets
	}
	if opts.GracePeriodSeconds != nil {
		body["gracePeriodSeconds"] = *opts.GracePeriodSeconds
	}
	if opts.DeleteEmptyDirData != nil {
		body["deleteEmptyDirData"] = *opts.DeleteEmptyDirData
	}
	return c.poolOp(ctx, cluster, pool, "drain", body)
}

// decodeKubeconfig unwraps the answer of the sentry service. Asked for JSON, which is what the SDK's
// client always does, it wraps the file as {"content_type": ..., "data": "<base64>"}; for any other
// Accept it returns the YAML itself, which is passed through unchanged.
func decodeKubeconfig(raw []byte) []byte {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return raw
	}
	var envelope struct {
		ContentType string `json:"content_type"`
		Data        string `json:"data"`
	}
	if err := json.Unmarshal(trimmed, &envelope); err != nil || envelope.Data == "" {
		return raw
	}
	if decoded, err := base64.StdEncoding.DecodeString(envelope.Data); err == nil {
		return decoded
	}
	return []byte(envelope.Data)
}
