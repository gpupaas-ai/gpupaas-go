package v1alpha1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/mock/clusterapi"
)

// runningMKS creates a one-node cluster with the per-pool network rules on and waits until it runs.
func runningMKS(t *testing.T, name string, opts clusterapi.Options) (*clusterapi.Server, clientset.Interface, typed.MKSClusterInterface, context.Context) {
	t.Helper()
	opts.PerPoolNetwork = true
	srv, cs := newMKSFixture(t, opts)
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	obj := oneNodeCluster(name)
	obj.Spec.ControlPlane.Network = &apiv1.MKSNodePoolNetwork{VPC: "vpc-1", Subnet: "sub-1"}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, name, mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	return srv, cs, clusters, ctx
}

func pool(name string, count int32) apiv1.MKSNodeGroup {
	return apiv1.MKSNodeGroup{Name: name, SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26",
		Network: &apiv1.MKSNodePoolNetwork{VPC: "vpc-1", Subnet: "sub-1"}, Scaling: &apiv1.MKSScaling{Mode: "manual", Count: count}}
}

func apiStatus(err error) int {
	var apiErr *gpupaas.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

func hasRequest(srv *clusterapi.Server, want string) bool {
	for _, r := range srv.Requests() {
		if strings.HasPrefix(r, want) {
			return true
		}
	}
	return false
}

func TestMKSAddAndRemoveNodePoolRoutes(t *testing.T) {
	srv, cs, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{})
	_ = cs

	// An unset assignPublicIp takes the value of the existing worker pools.
	yes := true
	first := pool("pool-a", 1)
	first.AssignPublicIP = &yes
	added, err := clusters.AddNodePool(ctx, "c1", first)
	if err != nil {
		t.Fatal(err)
	}
	if added.Status.Phase() != "updating" || len(added.Spec.WorkerPools) != 1 {
		t.Fatalf("added: phase=%q pools=%d", added.Status.Phase(), len(added.Spec.WorkerPools))
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-b", 2)); err != nil {
		t.Fatal(err)
	}
	got, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	var b *apiv1.MKSNodeGroup
	for i := range got.Spec.WorkerPools {
		if got.Spec.WorkerPools[i].Name == "pool-b" {
			b = &got.Spec.WorkerPools[i]
		}
	}
	if b == nil || b.AssignPublicIP == nil || !*b.AssignPublicIP || b.Scaling.Count != 2 {
		t.Fatalf("pool-b did not take the workers' public IP setting: %#v", b)
	}

	// The same name twice and a pool without a network are refused by the platform.
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 1)); apiStatus(err) != 400 {
		t.Fatalf("duplicate pool: %v", err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 1, false)); err != nil {
		t.Fatal(err)
	}
	bare := pool("pool-c", 1)
	bare.Network = nil
	if _, err := clusters.AddNodePool(ctx, "c1", bare); apiStatus(err) != 400 || !strings.Contains(err.Error(), "Select a VPC and a subnet") {
		t.Fatalf("pool without network: %v", err)
	}
	if _, err := clusters.AddNodePool(ctx, "c1", apiv1.MKSNodeGroup{}); apiStatus(err) != 400 {
		t.Fatalf("nameless pool: %v", err)
	}

	removed, err := clusters.RemoveNodePool(ctx, "c1", "pool-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Spec.WorkerPools) != 1 || removed.Spec.WorkerPools[0].Name != "pool-b" {
		t.Fatalf("removed: %#v", removed.Spec.WorkerPools)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := clusters.RemoveNodePool(ctx, "c1", "nope"); apiStatus(err) != 400 {
		t.Fatalf("unknown pool: %v", err)
	}
	if !hasRequest(srv, "POST /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusters/c1/nodepools") ||
		!hasRequest(srv, "DELETE /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusters/c1/nodepools/pool-a") {
		t.Fatalf("routes not used: %v", srv.Requests())
	}
	if _, err := clusters.RemoveNodePool(ctx, "", "x"); apiStatus(err) != 400 {
		t.Fatalf("empty cluster name: %v", err)
	}
}

func TestMKSDay2IsRefusedWhileAnotherOperationRuns(t *testing.T) {
	_, _, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{Step: 150 * time.Millisecond})
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 1)); err != nil {
		t.Fatal(err)
	}
	// The cluster is UPDATING now: a second operation answers 412.
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-b", 1)); apiStatus(err) != 412 {
		t.Fatalf("second operation: %v", err)
	}
}

func TestMKSUpgradeRoute(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{PerPoolNetwork: true})
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	obj := oneNodeCluster("c1")
	obj.Spec.KubernetesVersion = "1.35"
	obj.Spec.ControlPlane.Network = &apiv1.MKSNodePoolNetwork{VPC: "vpc-1", Subnet: "sub-1"}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	cur, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	if got := cur.Status.AvailableKubernetesVersions; len(got) != 1 || got[0] != "1.36" {
		t.Fatalf("upgrade targets: %v", got)
	}

	if _, err := clusters.Upgrade(ctx, "c1", apiv1.MKSUpgradeRequest{}); apiStatus(err) != 400 {
		t.Fatalf("empty upgrade: %v", err)
	}
	if _, err := clusters.Upgrade(ctx, "c1", apiv1.MKSUpgradeRequest{K8sVersion: "1.37"}); apiStatus(err) != 400 || !strings.Contains(err.Error(), "not an upgrade target") {
		t.Fatalf("skipped version: %v", err)
	}
	up, err := clusters.Upgrade(ctx, "c1", apiv1.MKSUpgradeRequest{K8sVersion: "1.36", Blueprint: "minimal", BlueprintVersion: "2"})
	if err != nil {
		t.Fatal(err)
	}
	if up.Status.Phase() != "upgrading" || up.Spec.KubernetesVersion != "1.36" && !strings.HasPrefix(up.Spec.KubernetesVersion, "v1.36") {
		t.Fatalf("upgrade answer: phase=%q version=%q", up.Status.Phase(), up.Spec.KubernetesVersion)
	}
	done, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	if done.Spec.Blueprint == nil || done.Spec.Blueprint.Version != "2" {
		t.Fatalf("blueprint after upgrade: %#v", done.Spec.Blueprint)
	}
	if !hasRequest(srv, "POST /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusters/c1/upgrade") {
		t.Fatalf("upgrade route not used: %v", srv.Requests())
	}
	// The body carries only what changes, with the reference's names.
	body, _ := json.Marshal(apiv1.MKSUpgradeRequest{K8sVersion: "1.36", CNI: &apiv1.MKSCNI{Name: "Calico"}})
	if string(body) != `{"k8sVersion":"1.36","cni":{"name":"Calico"}}` {
		t.Fatalf("upgrade body: %s", body)
	}
}

func TestMKSMetadataPatchIsAFastPath(t *testing.T) {
	srv, _, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{})
	before, err := clusters.Get(ctx, "c1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := clusters.UpdateMetadata(ctx, "c1", apiv1.MKSMetadataPatch{Labels: map[string]string{"team": "ml"}, Tags: map[string]string{"env": "dev", "owner": "me"}})
	if err != nil {
		t.Fatal(err)
	}
	// No redeploy: the cluster stays RUNNING.
	if got.Status.Phase() != "running" || got.Metadata.Labels["team"] != "ml" || got.Spec.Tags["env"] != "dev" || got.Version != before.Version+1 {
		t.Fatalf("patched: phase=%q labels=%v tags=%v version=%d", got.Status.Phase(), got.Metadata.Labels, got.Spec.Tags, got.Version)
	}
	// Without a replace flag a map is merged into the stored one; a nil map leaves a field alone.
	got, err = clusters.UpdateMetadata(ctx, "c1", apiv1.MKSMetadataPatch{Tags: map[string]string{"env": "prod"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Tags["env"] != "prod" || got.Spec.Tags["owner"] != "me" || got.Metadata.Labels["team"] != "ml" {
		t.Fatalf("merge: labels=%v tags=%v", got.Metadata.Labels, got.Spec.Tags)
	}
	// With the flag the map replaces the stored one; an empty map clears it.
	got, err = clusters.UpdateMetadata(ctx, "c1", apiv1.MKSMetadataPatch{Tags: map[string]string{"only": "this"}, ReplaceTags: true, Labels: map[string]string{}, ReplaceLabels: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Spec.Tags) != 1 || got.Spec.Tags["only"] != "this" || len(got.Metadata.Labels) != 0 {
		t.Fatalf("replace: labels=%v tags=%v", got.Metadata.Labels, got.Spec.Tags)
	}
	if !hasRequest(srv, "PATCH /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusters/c1/metadata") {
		t.Fatalf("metadata route not used: %v", srv.Requests())
	}

	// A nil map is left out of the body, an empty one is written.
	b, _ := json.Marshal(apiv1.MKSMetadataPatch{Labels: map[string]string{}, ReplaceLabels: true})
	if string(b) != `{"labels":{},"replaceLabels":true}` {
		t.Fatalf("patch body: %s", b)
	}
	b, _ = json.Marshal(apiv1.MKSMetadataPatch{})
	if string(b) != `{}` {
		t.Fatalf("empty patch body: %s", b)
	}
}

func TestMKSRedeployOnlyFromErrorOrDegraded(t *testing.T) {
	failing := true
	srv, cs := newMKSFixture(t, clusterapi.Options{PerPoolNetwork: true})
	srv.FailProvision = func(string) string {
		if failing {
			return "boom"
		}
		return ""
	}
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	obj := oneNodeCluster("c1")
	obj.Spec.ControlPlane.Network = &apiv1.MKSNodePoolNetwork{VPC: "vpc-1", Subnet: "sub-1"}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	// The create ends in ERROR.
	failed := func(c *apiv1.MKSCluster) (bool, error) { return c.Status.Failed(), nil }
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(failed, 1, false)); err != nil {
		t.Fatal(err)
	}
	failing = false
	again, err := clusters.Redeploy(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if again.Status.Phase() != "provisioning" {
		t.Fatalf("redeploy answer: %q", again.Status.Phase())
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	// RUNNING: refused.
	if _, err := clusters.Redeploy(ctx, "c1"); apiStatus(err) != 412 || !strings.Contains(err.Error(), "ERROR or DEGRADED") {
		t.Fatalf("redeploy of a running cluster: %v", err)
	}
	if _, err := clusters.Redeploy(ctx, "missing"); !gpupaas.IsNotFound(err) {
		t.Fatalf("redeploy of an unknown cluster: %v", err)
	}
}

func TestMKSStatusDocument(t *testing.T) {
	_, _, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{})
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 2)); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	st, err := clusters.Status(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	cp := st.Spec.ControlPlane
	if cp == nil || len(cp.Nodes) != 1 || cp.Nodes[0].Roles[0] != "Master" || cp.Nodes[0].PrivateIP == "" || cp.Nodes[0].Status != "READY" || cp.Status != "ACTIVE" {
		t.Fatalf("control plane: %#v", cp)
	}
	if len(st.Spec.WorkerPools) != 1 || st.Spec.WorkerPools[0].Name != "pool-a" || len(st.Spec.WorkerPools[0].Nodes) != 2 {
		t.Fatalf("workers: %#v", st.Spec.WorkerPools)
	}
	n := st.Spec.WorkerPools[0].Nodes[0]
	if n.Roles[0] != "Worker" || n.KubeVersion == "" || n.State["ClusterNodeReady"] != "true" {
		t.Fatalf("worker node: %#v", n)
	}
	if st.Status.Health == nil || st.Status.Health.Overall != "MKS_HEALTH_STATE_HEALTHY" || st.Status.Health.NodePools["pool-a"] != "MKS_HEALTH_STATE_HEALTHY" || st.Status.Status != "MKS_CLUSTER_STATUS_RUNNING" {
		t.Fatalf("status: %#v", st.Status)
	}
	if _, err := clusters.Status(ctx, "missing"); !gpupaas.IsNotFound(err) {
		t.Fatalf("status of an unknown cluster: %v", err)
	}
}

func TestMKSAuditEvents(t *testing.T) {
	srv, _, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{})
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := clusters.UpdateMetadata(ctx, "c1", apiv1.MKSMetadataPatch{Tags: map[string]string{"a": "b"}}); err != nil {
		t.Fatal(err)
	}

	all, err := clusters.AuditEvents(ctx, "c1", typed.MKSAuditEventFilter{})
	if err != nil {
		t.Fatal(err)
	}
	// The platform writes the total and the timestamp as strings; both are read. Newest first.
	if all.Count != 3 || len(all.Items) != 3 || all.Items[0].Action != "cluster.metadata.update" || all.Items[2].Action != "cluster.apply" || all.Items[0].Timestamp == 0 {
		t.Fatalf("all: %#v", all)
	}
	one, err := clusters.AuditEvents(ctx, "c1", typed.MKSAuditEventFilter{Action: "cluster.nodepool.create", Limit: 10})
	if err != nil || one.Count != 1 || one.Items[0].ResourceName != "pool-a" || one.Items[0].ResourceType != "nodePool" {
		t.Fatalf("filtered: %v %#v", err, one)
	}
	page, err := clusters.AuditEvents(ctx, "c1", typed.MKSAuditEventFilter{Limit: 1, Offset: 1})
	if err != nil || page.Count != 3 || len(page.Items) != 1 || page.Items[0].Action != "cluster.nodepool.create" {
		t.Fatalf("page: %v %#v", err, page)
	}
	none, err := clusters.AuditEvents(ctx, "c1", typed.MKSAuditEventFilter{Result: "failed", Actor: "nobody", From: time.Unix(1, 0), To: time.Unix(2, 0)})
	if err != nil || none.Count != 0 || len(none.Items) != 0 {
		t.Fatalf("none: %v %#v", err, none)
	}
	if !hasRequest(srv, "GET /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusters/c1/auditevents?action=cluster.nodepool.create&limit=10") {
		t.Fatalf("query: %v", srv.Requests())
	}
	if got := srv.AuditActions(testProject, "", "c1"); len(got) != 3 {
		t.Fatalf("recorded: %v", got)
	}
}

func TestMKSAuditEventListReadsEveryLayout(t *testing.T) {
	for name, doc := range map[string]string{
		"camel":  `{"auditEvents":[{"id":"1","action":"a","timestamp":"7"}],"metadata":{"count":"9"}}`,
		"snake":  `{"audit_events":[{"id":"1","action":"a","timestamp":7}],"metadata":{"count":9}}`,
		"items":  `{"items":[{"id":"1","action":"a","timestamp":"7"}],"metadata":{"count":9}}`,
		"nocnt":  `{"items":[{"id":"1","action":"a"},{"id":"2","action":"b"}]}`,
		"nodata": `{}`,
	} {
		var l apiv1.MKSAuditEventList
		if err := json.Unmarshal([]byte(doc), &l); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := 1
		switch name {
		case "nodata":
			want = 0
		case "nocnt":
			want = 2
		}
		if len(l.Items) != want {
			t.Fatalf("%s: %d items", name, len(l.Items))
		}
		if name == "camel" && (l.Count != 9 || l.Items[0].Timestamp != 7) {
			t.Fatalf("camel: %#v", l)
		}
		// Without a total, the count is the size of the page.
		if name == "nocnt" && (len(l.Items) != 2 || l.Count != 2) {
			t.Fatalf("nocnt: %#v", l)
		}
	}
}

func TestMKSProfileDiscovery(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{})
	profiles := cs.V1alpha1().MKSClusterProfiles(testProject)
	ctx := waitCtx(t)

	opts, err := profiles.CreateOptions(ctx, "mks-oneclick-1")
	if err != nil {
		t.Fatal(err)
	}
	if opts.ControlPlane == nil || opts.WorkerNode == nil || opts.DefaultKubernetesVersion != "1.36" || len(opts.DataCenterRegions) != 1 ||
		opts.WorkerNode.SKUNames[0] != "bmaas-test-1" || opts.WorkerNode.SKUKind[0] != "bmaas" || opts.WorkerNode.Images[0] != "ubuntu-26" ||
		len(opts.SupportedKubernetesVersion) != 2 {
		t.Fatalf("create options: %#v", opts)
	}
	capacity, err := profiles.AvailableCapacity(ctx, "mks-oneclick-1")
	if err != nil {
		t.Fatal(err)
	}
	// The shape of the answer is not part of the API reference: it is kept as received.
	var doc struct {
		Profile string           `json:"profile"`
		SKUs    []map[string]any `json:"skus"`
	}
	if err := json.Unmarshal(capacity.Raw, &doc); err != nil || doc.Profile != "mks-oneclick-1" || len(doc.SKUs) != 2 {
		t.Fatalf("capacity: %v %s", err, capacity.Raw)
	}
	if again, err := json.Marshal(capacity); err != nil || string(again) != string(capacity.Raw) {
		t.Fatalf("capacity round trip: %v %s", err, again)
	}
	if _, err := profiles.CreateOptions(ctx, "nope"); !gpupaas.IsNotFound(err) {
		t.Fatalf("unknown profile: %v", err)
	}
	if _, err := profiles.AvailableCapacity(ctx, ""); apiStatus(err) != 400 {
		t.Fatalf("empty profile name: %v", err)
	}
}

func TestMKSMatchmaker(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	mm := cs.V1alpha1().MKSMatchmaker(testProject)
	ctx := waitCtx(t)

	v, err := mm.KubernetesVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Minors) != 2 || v.Minors[0] != "1.36" || v.Versions[0].LatestPatch != "v1.36.4" || !v.Versions[0].IsDefault || v.Versions[1].IsDefault {
		t.Fatalf("versions: %#v", v)
	}

	q := typed.MKSMatchQuery{Datacenters: []string{"default-dc", "dc-2"}, SKUKinds: []string{"bmaas", "vmaas"}, ComputeTypes: []string{"cpu", "gpu"}}
	imgs, err := mm.AvailableImages(ctx, q)
	if err != nil || len(imgs) != 1 || imgs[0].Name != "ubuntu-26" || !imgs[0].Default {
		t.Fatalf("images: %v %#v", err, imgs)
	}
	// The query follows the API reference: datacenters repeat, kinds and types are joined with a plus, which is percent-encoded
	// (a raw plus means a space to a server that decodes it).
	if !hasRequest(srv, "GET /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clustermatchmakers/availableimages?datacenter=default-dc&datacenter=dc-2&sku_kind=bmaas%2Bvmaas&compute_type=cpu%2Bgpu") {
		t.Fatalf("query: %v", srv.Requests())
	}
	os, err := mm.AvailableOS(ctx, q)
	if err != nil || len(os.OSImages) != 1 || os.OSImages[0].Name != "Ubuntu" || os.OSImages[0].Versions[0].ImageName != "ubuntu-26" {
		t.Fatalf("os: %v %#v", err, os)
	}
	q.Images = []string{"ubuntu-26"}
	specs, err := mm.AvailableSpecs(ctx, q)
	if err != nil || len(specs) != 1 || specs[0].CPU != 8 || specs[0].RAMGb != 32 || specs[0].BootDiskRange == nil || specs[0].BootDiskRange.IncrementGb != 10 || specs[0].SKUNames[0] != "bmaas-test-1" {
		t.Fatalf("specs: %v %#v", err, specs)
	}
	comp, err := mm.AvailableComputeSKUs(ctx, typed.MKSMatchQuery{Datacenters: []string{"default-dc"}, SKUKinds: []string{"bmaas"}, ComputeTypes: []string{"cpu"}, Images: []string{"ubuntu-26"}})
	if err != nil || len(comp) != 1 || comp[0].Name != "bmaas-test-1" || comp[0].Images[0] != "ubuntu-26" {
		t.Fatalf("compute skus: %v %#v", err, comp)
	}
	skus, err := mm.MatchSKUs(ctx, typed.MKSMatchQuery{Datacenters: []string{"default-dc"}, SKUKinds: []string{"bmaas"}, ComputeTypes: []string{"cpu"}, CPU: 8, RAMGb: 32, StorageGb: 100})
	if err != nil || len(skus) != 1 || skus[0].Name != "bmaas-test-1" || skus[0].RAMGb != 32 {
		t.Fatalf("match skus: %v %#v", err, skus)
	}
	if !hasRequest(srv, "GET /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clustermatchmakers/matchskus?datacenter=default-dc&sku_kind=bmaas&compute_type=cpu&cpu=8&ram_gb=32&storage_gb=100") {
		t.Fatalf("matchskus query: %v", srv.Requests())
	}

	// Required parameters are checked before a request is made.
	n := len(srv.Requests())
	for name, call := range map[string]func() error{
		"no datacenter": func() error {
			_, err := mm.AvailableImages(ctx, typed.MKSMatchQuery{SKUKinds: []string{"bmaas"}, ComputeTypes: []string{"cpu"}})
			return err
		},
		"no kind": func() error {
			_, err := mm.AvailableOS(ctx, typed.MKSMatchQuery{Datacenters: []string{"d"}, ComputeTypes: []string{"cpu"}})
			return err
		},
		"no type": func() error {
			_, err := mm.AvailableSpecs(ctx, typed.MKSMatchQuery{Datacenters: []string{"d"}, SKUKinds: []string{"bmaas"}, Images: []string{"i"}})
			return err
		},
		"no image": func() error {
			_, err := mm.AvailableComputeSKUs(ctx, typed.MKSMatchQuery{Datacenters: []string{"d"}, SKUKinds: []string{"bmaas"}, ComputeTypes: []string{"cpu"}})
			return err
		},
		"gpu without a type": func() error {
			_, err := mm.MatchSKUs(ctx, typed.MKSMatchQuery{Datacenters: []string{"d"}, SKUKinds: []string{"bmaas"}, ComputeTypes: []string{"gpu"}})
			return err
		},
	} {
		if err := call(); apiStatus(err) != 400 {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(srv.Requests()) != n {
		t.Fatalf("a request was sent for an incomplete query: %v", srv.Requests()[n:])
	}
}

// The platform may write keys as camelCase (protobuf JSON) or as the proto's snake_case names.
// The API reference accepts compute_type "custom" (treated as cpu), but a v3.1.41 controller answers
// 400 `compute_type must be "cpu" or "gpu" (or cpu+gpu)` for it. The client sends it as cpu, once.
func TestMKSMatchmakerSendsCustomAsCPU(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	mm := cs.V1alpha1().MKSMatchmaker(testProject)
	ctx := waitCtx(t)
	base := "GET /apis/paas.k8smgmt.io/v3/projects/" + testProject + "/clustermatchmakers/availableimages?datacenter=default-dc&sku_kind=bmaas&compute_type="
	for _, c := range []struct {
		types []string
		want  string
	}{
		{[]string{"custom"}, "cpu"},
		{[]string{"Custom", "cpu"}, "cpu"},
		{[]string{"custom", "gpu"}, "cpu%2Bgpu"},
		{[]string{"gpu", "custom"}, "gpu%2Bcpu"},
	} {
		if _, err := mm.AvailableImages(ctx, typed.MKSMatchQuery{Datacenters: []string{"default-dc"}, SKUKinds: []string{"bmaas"}, ComputeTypes: c.types}); err != nil {
			t.Fatalf("%v: %v", c.types, err)
		}
		if !hasRequest(srv, base+c.want) {
			t.Fatalf("%v: want compute_type=%s, got %v", c.types, c.want, srv.Requests())
		}
	}
}

func TestMKSMatchmakerReadsSnakeCaseAnswers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/kubernetesversions"):
			_, _ = w.Write([]byte(`{"versions":[{"minor":"1.36","latest_patch":"v1.36.4","is_default":true}]}`))
		case strings.HasSuffix(r.URL.Path, "/availablespecs"):
			_, _ = w.Write([]byte(`{"specs":[{"sku_kind":"vmaas","compute_type":"gpu","ram_gb":64,"gpu_type":"h100","gpu_count":2,"gpu_memory_gb":80,"boot_disk_range":{"min_gb":50,"max_gb":200,"increment_gb":10,"unit":"GB"},"sku_names":["a"]}]}`))
		case strings.HasSuffix(r.URL.Path, "/createoptions"):
			_, _ = w.Write([]byte(`{"control_plane":{"sku_names":["cp"]},"worker_node":{"instance_types":[{"name":"w","compute_type":"gpu","shapes":[{"ram_gb":8}]}]},"data_center_regions":["dc"],"default_kubernetes_version":"1.36","enable_oidc":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(ts.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := waitCtx(t)
	v, err := cs.V1alpha1().MKSMatchmaker(testProject).KubernetesVersions(ctx)
	if err != nil || len(v.Versions) != 1 || v.Versions[0].LatestPatch != "v1.36.4" || !v.Versions[0].IsDefault || len(v.Minors) != 1 || v.Minors[0] != "1.36" {
		t.Fatalf("versions: %v %#v", err, v)
	}
	specs, err := cs.V1alpha1().MKSMatchmaker(testProject).AvailableSpecs(ctx, typed.MKSMatchQuery{Datacenters: []string{"d"}, SKUKinds: []string{"vmaas"}, ComputeTypes: []string{"gpu"}, Images: []string{"i"}})
	if err != nil || len(specs) != 1 || specs[0].GPUCount != 2 || specs[0].GPUMemoryGb != 80 || specs[0].RAMGb != 64 || specs[0].BootDiskRange.MaxGb != 200 {
		t.Fatalf("specs: %v %#v", err, specs)
	}
	opts, err := cs.V1alpha1().MKSClusterProfiles(testProject).CreateOptions(ctx, "p")
	if err != nil || opts.ControlPlane.SKUNames[0] != "cp" || opts.WorkerNode.InstanceTypes[0].Shapes[0].RAMGb != 8 || !opts.EnableOIDC || opts.DataCenterRegions[0] != "dc" {
		t.Fatalf("create options: %v %#v", err, opts)
	}
}

func TestMKSNodeOperations(t *testing.T) {
	srv, cs, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{})
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	nodes := cs.V1alpha1().MKSNodes(testProject)
	list, err := nodes.ListForCluster(ctx, "c1")
	if err != nil || len(list.Items) != 2 {
		t.Fatalf("nodes: %v %#v", err, list)
	}
	// The richer node fields of the v3.1.41 controllers are read.
	var worker apiv1.MKSNode
	for _, n := range list.Items {
		if n.NodePool() == "pool-a" {
			worker = n
		}
	}
	if worker.Spec.PrivateIP == "" || worker.Spec.Roles[0] != "Worker" || worker.Spec.NodePool != "pool-a" || worker.Spec.SKU != "bmaas-test-1" || worker.Status.EdgeStatus != "READY" || worker.Status.Phase != "MKS_NODE_PHASE_RUNNING" {
		t.Fatalf("worker node: %#v", worker)
	}
	host := worker.Spec.Hostname

	for _, step := range []struct {
		name string
		call func() error
	}{
		{"cordon", func() error { return nodes.CordonNode(ctx, "c1", host) }},
		{"uncordon", func() error { return nodes.UncordonNode(ctx, "c1", host) }},
		{"drain", func() error { return nodes.DrainNode(ctx, "c1", host) }},
		{"metadata", func() error {
			return nodes.UpdateNodeMetadata(ctx, "c1", host, typed.MKSNodeMetadataPatch{SetLabels: map[string]string{"a": "b"}, RemoveAnnotations: []string{"x"}})
		}},
		{"taints", func() error {
			return nodes.UpdateNodeTaints(ctx, "c1", host, []apiv1.MKSTaint{{Key: "k", Value: "v", Effect: apiv1.MKSTaintNoSchedule}})
		}},
		{"clear taints", func() error { return nodes.UpdateNodeTaints(ctx, "c1", host, nil) }},
		{"delete", func() error { return nodes.DeleteNode(ctx, "c1", host) }},
	} {
		if err := step.call(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}
	want := []string{"cordon", "uncordon", "drain", "metadata", "taints", "taints", "delete"}
	got := srv.NodeOps()
	if len(got) != len(want) {
		t.Fatalf("operations: %v", got)
	}
	for i, w := range want {
		if got[i] != w+" "+testProject+"/c1/"+host {
			t.Fatalf("operation %d: %q", i, got[i])
		}
	}
	// The request addresses the node by hostname and names the cluster in the query.
	if !hasRequest(srv, "POST /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusternodes/"+host+"/drain?cluster_name=c1") ||
		!hasRequest(srv, "PUT /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusternodes/"+host+"/taints?cluster_name=c1") ||
		!hasRequest(srv, "DELETE /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusternodes/"+host+"?cluster_name=c1") {
		t.Fatalf("routes: %v", srv.Requests())
	}

	if err := nodes.CordonNode(ctx, "c1", "no-such-node"); !gpupaas.IsNotFound(err) {
		t.Fatalf("unknown node: %v", err)
	}
	if err := nodes.UpdateNodeMetadata(ctx, "c1", host, typed.MKSNodeMetadataPatch{}); apiStatus(err) != 400 {
		t.Fatalf("empty metadata change: %v", err)
	}
	if err := nodes.UpdateNodeMetadata(ctx, "c1", host, typed.MKSNodeMetadataPatch{SetLabels: map[string]string{"a": "b"}, RemoveLabels: []string{"a"}}); apiStatus(err) != 400 {
		t.Fatalf("contradictory metadata change: %v", err)
	}
	if err := nodes.DrainNode(ctx, "", host); apiStatus(err) != 400 {
		t.Fatalf("empty cluster name: %v", err)
	}
}

// The remaining validation rules of the v3.1.41 reference, enforced by the mock with PerPoolNetwork.
func TestMKSMockEnforcesTheReferencesRemainingRules(t *testing.T) {
	prof := mksProfile()
	prof.Spec.ExposeSSHKeyChoice = true // a new cluster must choose a password or an SSH key, and the SKU has no default
	srv := clusterapi.New(clusterapi.Options{Profiles: []apiv1.MKSClusterProfile{prof}, PerPoolNetwork: true})
	t.Cleanup(srv.Close)
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	build := func(edit func(*apiv1.MKSCluster)) *apiv1.MKSCluster {
		c := oneNodeCluster("c1")
		c.Spec.ControlPlane.Network = &apiv1.MKSNodePoolNetwork{VPC: "vpc-1", Subnet: "sub-1"}
		if edit != nil {
			edit(c)
		}
		return c
	}
	refuse := func(name, want string, c *apiv1.MKSCluster) {
		t.Helper()
		if _, err := clusters.Create(ctx, c, gpupaas.CreateOptions{}); apiStatus(err) != 400 || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	refuse("two control-plane nodes", "1 or 3 nodes", build(func(c *apiv1.MKSCluster) { c.Spec.ControlPlane.Scaling.Count = 2 }))
	refuse("another datacenter", "per-pool datacenter", build(func(c *apiv1.MKSCluster) {
		c.Spec.Location = "default-dc"
		c.Spec.ControlPlane.Datacenter = "elsewhere"
	}))
	refuse("an inline key", "inline public keys are not accepted", build(func(c *apiv1.MKSCluster) { c.Spec.ControlPlane.SSHKey = "ssh-ed25519 AAAA me@host" }))
	refuse("no node access", "Either a node password or an SSH key is required", build(func(c *apiv1.MKSCluster) { c.Spec.ControlPlane.SSHKey = "" }))

	// A password is enough; so is a key on a pool (one key serves the cluster).
	if _, err := clusters.Create(ctx, build(func(c *apiv1.MKSCluster) { c.Spec.ControlPlane.SSHKey, c.Spec.ControlPlane.UserPassword = "", "pw" }), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	// The size of the control plane is fixed once it exists.
	cur, err := clusters.Get(ctx, "c1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	resized := cur.DeepCopyObject().(*apiv1.MKSCluster)
	resized.Spec.ControlPlane.Scaling.Count = 3
	if _, err := clusters.Update(ctx, resized, gpupaas.UpdateOptions{}); apiStatus(err) != 400 || !strings.Contains(err.Error(), "can't be changed") {
		t.Fatalf("resize of the control plane: %v", err)
	}
	// A pool of an existing cluster needs no access of its own.
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 1)); err != nil {
		t.Fatalf("pool of an existing cluster: %v", err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	if err := clusters.Delete(ctx, "c1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(nil, 1, true)); err != nil {
		t.Fatal(err)
	}
	onPool := build(func(c *apiv1.MKSCluster) {
		c.Spec.ControlPlane.SSHKey = ""
		w := pool("pool-a", 1)
		w.SSHKey = "mine"
		c.Spec.WorkerPools = []apiv1.MKSNodeGroup{w}
	})
	if _, err := clusters.Create(ctx, onPool, gpupaas.CreateOptions{}); err != nil {
		t.Fatalf("key on a pool: %v", err)
	}
}

// The answer of a day-2 call that carries no cluster is replaced by reading the cluster.
func TestMKSDay2AnswerWithoutAClusterIsReadAgain(t *testing.T) {
	var gets int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/clusters/c1"):
			gets++
			_, _ = w.Write([]byte(`{"apiVersion":"paas.k8smgmt.io/v3","kind":"Cluster","metadata":{"name":"c1","project":"p"},"spec":{"kubernetesVersion":"v1.36.4"},"status":{"status":"MKS_CLUSTER_STATUS_RUNNING"}}`))
		case r.Method != http.MethodGet:
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(ts.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	clusters := cs.V1alpha1().MKSClusters("p")
	ctx := waitCtx(t)
	for name, call := range map[string]func() (*apiv1.MKSCluster, error){
		"add":    func() (*apiv1.MKSCluster, error) { return clusters.AddNodePool(ctx, "c1", pool("a", 1)) },
		"remove": func() (*apiv1.MKSCluster, error) { return clusters.RemoveNodePool(ctx, "c1", "a") },
		"upgrade": func() (*apiv1.MKSCluster, error) {
			return clusters.Upgrade(ctx, "c1", apiv1.MKSUpgradeRequest{K8sVersion: "1.37"})
		},
		"metadata": func() (*apiv1.MKSCluster, error) { return clusters.UpdateMetadata(ctx, "c1", apiv1.MKSMetadataPatch{}) },
		"redeploy": func() (*apiv1.MKSCluster, error) { return clusters.Redeploy(ctx, "c1") },
	} {
		before := gets
		got, err := call()
		if err != nil || got == nil || got.Metadata.Name != "c1" || !got.Status.Running() || gets != before+1 {
			t.Errorf("%s: %v %#v (reads %d)", name, err, got, gets-before)
		}
	}
}

func TestMKSAuditEventFilters(t *testing.T) {
	_, _, clusters, ctx := runningMKS(t, "c1", clusterapi.Options{})
	if _, err := clusters.AddNodePool(ctx, "c1", pool("pool-a", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	count := func(f typed.MKSAuditEventFilter) int64 {
		t.Helper()
		l, err := clusters.AuditEvents(ctx, "c1", f)
		if err != nil {
			t.Fatal(err)
		}
		return l.Count
	}
	// apply (cluster, accepted) and nodepool create (nodePool, accepted).
	if n := count(typed.MKSAuditEventFilter{}); n != 2 {
		t.Fatalf("all: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{ResourceType: "nodePool"}); n != 1 {
		t.Errorf("resource type nodePool: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{ResourceType: "cluster"}); n != 1 {
		t.Errorf("resource type cluster: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{Result: "accepted"}); n != 2 {
		t.Errorf("result accepted: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{Result: "failed"}); n != 0 {
		t.Errorf("result failed: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{Actor: "tester"}); n != 2 {
		t.Errorf("actor: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{Actor: "someone-else"}); n != 0 {
		t.Errorf("other actor: %d", n)
	}
	// The mock stamps its events in November 2023.
	if n := count(typed.MKSAuditEventFilter{From: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}); n != 0 {
		t.Errorf("from the future: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}); n != 2 {
		t.Errorf("from the past: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{To: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}); n != 0 {
		t.Errorf("to the past: %d", n)
	}
	if n := count(typed.MKSAuditEventFilter{To: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}); n != 2 {
		t.Errorf("to the future: %d", n)
	}
}

// Kinds and compute types are sent in lower case, whatever case the caller writes.
func TestMKSMatchmakerSendsLowerCaseKinds(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	_, err := cs.V1alpha1().MKSMatchmaker(testProject).AvailableImages(waitCtx(t), typed.MKSMatchQuery{Datacenters: []string{"d"}, SKUKinds: []string{"BMAAS", "VmAAS"}, ComputeTypes: []string{"GPU"}})
	if err != nil {
		t.Fatal(err)
	}
	if !hasRequest(srv, "GET /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clustermatchmakers/availableimages?datacenter=d&sku_kind=bmaas%2Bvmaas&compute_type=gpu") {
		t.Fatalf("query: %v", srv.Requests())
	}
}
