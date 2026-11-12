package v1alpha1_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/mock/clusterapi"
)

func mksProfile() apiv1.MKSClusterProfile {
	return apiv1.MKSClusterProfile{
		Metadata: apiv1.ObjectMeta{Name: "mks-oneclick-1"},
		Spec: apiv1.MKSClusterProfileSpec{
			ControlPlane:                &apiv1.MKSGroupProfile{DefaultSKUName: "bmaas-test-1", DefaultImage: "ubuntu-26", SKUNames: []string{"bmaas-test-1"}, SKUTypes: []string{"BMAAS"}, Images: []string{"ubuntu-26"}},
			Workers:                     &apiv1.MKSGroupProfile{DefaultSKUName: "bmaas-test-1", DefaultImage: "ubuntu-26", SKUNames: []string{"bmaas-test-1"}, SKUTypes: []string{"BMAAS"}, Images: []string{"ubuntu-26"}},
			Datacenters:                 []string{"default-dc"},
			SupportedCNIs:               []apiv1.MKSCNI{{Name: "Calico", Version: "3.32.2"}},
			DefaultCNI:                  &apiv1.MKSCNI{Name: "Calico", Version: "3.32.2"},
			DefaultBlueprint:            &apiv1.MKSBlueprint{Name: "minimal", Version: "latest"},
			DefaultKubernetesVersion:    "1.36",
			SupportedKubernetesVersions: []string{"1.36", "1.35"},
			DefaultPodSubnet:            "10.244.0.0/16",
			DefaultServiceSubnet:        "10.96.0.0/12",
		},
	}
}

func newMKSFixture(t *testing.T, opts clusterapi.Options) (*clusterapi.Server, clientset.Interface) {
	t.Helper()
	opts.Profiles = []apiv1.MKSClusterProfile{mksProfile()}
	srv := clusterapi.New(opts)
	t.Cleanup(srv.Close)
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	return srv, cs
}

func oneNodeCluster(name string) *apiv1.MKSCluster {
	return &apiv1.MKSCluster{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
		Metadata: apiv1.ObjectMeta{Name: name},
		Spec: apiv1.MKSClusterSpec{
			Profile:           &apiv1.MKSProfileRef{Name: "mks-oneclick-1"},
			KubernetesVersion: "1.36",
			// The live API rejects a create without a CNI: it does not apply profile defaults.
			Networking:   &apiv1.MKSNetworking{CNI: &apiv1.MKSCNI{Name: "Calico", Version: "3.32.2"}},
			ControlPlane: &apiv1.MKSNodeGroup{Name: "control-plane", SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26", SSHKey: "mine", Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 1}},
		},
	}
}

func mksWait(done func(*apiv1.MKSCluster) (bool, error), stable int, gone bool) typed.MKSWaitOptions {
	return typed.MKSWaitOptions{Interval: 5 * time.Millisecond, Done: done, Stable: stable, GoneIsDone: gone}
}

func TestMKSClusterFullLifecycleWorkspaceScope(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	clusters := cs.V1alpha1().Workspaces(testProject).MKSClusters("ws")
	ctx := waitCtx(t)

	created, err := clusters.Create(ctx, oneNodeCluster("c1"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status.Phase() != "provisioning" || created.Metadata.Workspace != "ws" || created.Metadata.Project != testProject {
		t.Fatalf("created: phase=%q meta=%#v", created.Status.Phase(), created.Metadata)
	}
	// The backend fills defaults and derived fields.
	cp := created.Spec.ControlPlane
	if created.Spec.Location != "default-dc" || created.Spec.KubernetesVersion != "v1.36.4" || created.Spec.Networking.CNI.Name != "Calico" || cp.ComputeType != "GPU" || cp.Shape == nil {
		t.Fatalf("derived fields missing: %#v", created.Spec)
	}

	// Create refuses to take over an existing name.
	_, err = clusters.Create(ctx, oneNodeCluster("c1"), gpupaas.CreateOptions{})
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("second create: %v", err)
	}

	running, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	if !running.Status.Running() || running.Status.Output.ClusterIDEdgesrv != "c1" {
		t.Fatalf("running: %#v", running.Status)
	}

	// Nodes appear once the cluster runs.
	nodes, err := cs.V1alpha1().MKSNodes(testProject).ListForCluster(ctx, "c1")
	if err != nil || len(nodes.Items) != 1 || nodes.Items[0].NodePool() != "master" {
		t.Fatalf("nodes: %v %#v", err, nodes)
	}

	// Update: add a worker pool and upgrade; the spec read back is what a later update starts from.
	next := running.DeepCopyObject().(*apiv1.MKSCluster)
	next.Spec.KubernetesVersion = "1.35"
	next.Spec.WorkerPools = []apiv1.MKSNodeGroup{{Name: "pool-a", SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26", Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 2}}}
	updated, err := clusters.Update(ctx, next, gpupaas.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase() != "updating" || updated.Version != running.Version+1 {
		t.Fatalf("updated: phase=%q version=%d (was %d)", updated.Status.Phase(), updated.Version, running.Version)
	}
	settled, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterMatches(next.Spec), 2, false))
	if err != nil {
		t.Fatal(err)
	}
	if len(settled.Spec.WorkerPools) != 1 || settled.Spec.WorkerPools[0].Scaling.Count != 2 || !strings.HasPrefix(settled.Spec.KubernetesVersion, "v1.35") {
		t.Fatalf("settled spec: %#v", settled.Spec)
	}
	nodes, _ = cs.V1alpha1().MKSNodes(testProject).ListForCluster(ctx, "c1")
	if len(nodes.Items) != 3 {
		t.Fatalf("nodes after resize = %d, want 3", len(nodes.Items))
	}

	list, err := clusters.List(ctx, gpupaas.ListOptions{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list: %v %#v", err, list)
	}

	if err := clusters.Delete(ctx, "c1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	gone, err := typed.WaitForMKSCluster(ctx, clusters, "c1", mksWait(typed.MKSClusterGone, 1, true))
	if err != nil || gone != nil {
		t.Fatalf("wait gone: %v %v", gone, err)
	}
	if _, err := clusters.Get(ctx, "c1", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("get after delete: %v", err)
	}
	if err := clusters.Delete(ctx, "c1", gpupaas.DeleteOptions{IgnoreNotFound: true}); err != nil {
		t.Fatalf("delete of a missing cluster with IgnoreNotFound: %v", err)
	}

	for _, r := range srv.Requests() {
		if strings.Contains(r, "mksclusters") {
			t.Fatalf("legacy route used: %s", r)
		}
	}
}

func TestMKSClusterProjectScopeUsesProjectRoutes(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	ctx := waitCtx(t)
	clusters := cs.V1alpha1().MKSClusters(testProject)
	if _, err := clusters.Create(ctx, oneNodeCluster("p1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if srv.Cluster(testProject, "", "p1") == nil || srv.Cluster(testProject, "ws", "p1") != nil {
		t.Fatal("project-scope cluster stored under the wrong scope")
	}
	found := false
	for _, r := range srv.Requests() {
		if r == "POST /apis/paas.k8smgmt.io/v3/projects/"+testProject+"/clusters?fail-on-exists=true" {
			found = true
		}
	}
	if !found {
		t.Fatalf("requests: %v", srv.Requests())
	}
}

func TestMKSClusterBackendValidationErrorsAreReturned(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{})
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)

	bad := oneNodeCluster("v1")
	bad.Spec.KubernetesVersion = "1.20"
	_, err := clusters.Create(ctx, bad, gpupaas.CreateOptions{})
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || !strings.Contains(err.Error(), "not supported by profile") {
		t.Fatalf("version: %v", err)
	}

	bad = oneNodeCluster("v2")
	bad.Spec.ControlPlane.SKUName = "unknown-sku"
	if _, err := clusters.Create(ctx, bad, gpupaas.CreateOptions{}); err == nil || !strings.Contains(err.Error(), "sku") {
		t.Fatalf("sku: %v", err)
	}

	bad = oneNodeCluster("v3")
	bad.Spec.Profile = &apiv1.MKSProfileRef{Name: "nope"}
	if _, err := clusters.Create(ctx, bad, gpupaas.CreateOptions{}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("profile: %v", err)
	}
}

func TestMKSWaitReportsProvisionFailure(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	srv.FailProvision = func(string) string { return "no capacity in datacenter" }
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	if _, err := clusters.Create(ctx, oneNodeCluster("f1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForMKSCluster(ctx, clusters, "f1", mksWait(typed.MKSClusterRunning, 1, false))
	var failed *typed.MKSClusterFailedError
	if !errors.As(err, &failed) || failed.Phase != "error" || failed.Reason != "no capacity in datacenter" {
		t.Fatalf("err = %v", err)
	}
}

func TestMKSWaitSurvivesFlapAndStaleReads(t *testing.T) {
	// After an update the backend can still show the old spec as RUNNING, then flap through
	// UPDATING. Waiting for the new spec with three stable polls rides this out.
	_, cs := newMKSFixture(t, clusterapi.Options{Flap: true, StaleReads: 3})
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	if _, err := clusters.Create(ctx, oneNodeCluster("fl"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	running, err := typed.WaitForMKSCluster(ctx, clusters, "fl", mksWait(typed.MKSClusterRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}

	next := running.DeepCopyObject().(*apiv1.MKSCluster)
	next.Spec.WorkerPools = []apiv1.MKSNodeGroup{{Name: "p", SKUName: "bmaas-test-1", OSImage: "ubuntu-26", Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 1}}}
	if _, err := clusters.Update(ctx, next, gpupaas.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}

	// A plain "is it RUNNING" check is fooled by the stale read.
	stale, err := clusters.Get(ctx, "fl", gpupaas.GetOptions{})
	if err != nil || !stale.Status.Running() || len(stale.Spec.WorkerPools) != 0 {
		t.Fatalf("expected a stale RUNNING read of the old spec, got phase=%q pools=%d err=%v", stale.Status.Phase(), len(stale.Spec.WorkerPools), err)
	}

	settled, err := typed.WaitForMKSCluster(ctx, clusters, "fl", mksWait(typed.MKSClusterMatches(next.Spec), 3, false))
	if err != nil {
		t.Fatal(err)
	}
	if len(settled.Spec.WorkerPools) != 1 || !settled.Status.Running() {
		t.Fatalf("settled: %#v", settled)
	}
}

func TestMKSUpdateWithStaleVersionConflicts(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{})
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	first, err := clusters.Create(ctx, oneNodeCluster("cv"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	a := first.DeepCopyObject().(*apiv1.MKSCluster)
	if _, err := clusters.Update(ctx, a, gpupaas.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	// The second writer still holds the first revision.
	b := first.DeepCopyObject().(*apiv1.MKSCluster)
	_, err = clusters.Update(ctx, b, gpupaas.UpdateOptions{})
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("stale update: %v", err)
	}
}

func TestMKSDeleteFailureThenRetry(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	var calls atomic.Int32
	srv.FailDestroy = func(string) string {
		if calls.Add(1) == 1 {
			return "runner could not reach the node"
		}
		return ""
	}
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	if _, err := clusters.Create(ctx, oneNodeCluster("d1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "d1", mksWait(typed.MKSClusterRunning, 1, false)); err != nil {
		t.Fatal(err)
	}

	if err := clusters.Delete(ctx, "d1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForMKSCluster(ctx, clusters, "d1", mksWait(typed.MKSClusterGone, 1, true))
	var failed *typed.MKSClusterFailedError
	if !errors.As(err, &failed) || failed.Phase != "destroyfailed" || !strings.Contains(failed.Reason, "could not reach") {
		t.Fatalf("first delete: %v", err)
	}

	// A second DELETE retries the teardown.
	if err := clusters.Delete(ctx, "d1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	gone, err := typed.WaitForMKSCluster(ctx, clusters, "d1", mksWait(typed.MKSClusterGone, 1, true))
	if err != nil || gone != nil {
		t.Fatalf("retry: %v %v", gone, err)
	}
}

func TestMKSNotFoundIsDetectedForBothStatuses(t *testing.T) {
	for _, status := range []int{404, 502} {
		_, cs := newMKSFixture(t, clusterapi.Options{NotFoundStatus: status})
		_, err := cs.V1alpha1().MKSClusters(testProject).Get(context.Background(), "missing", gpupaas.GetOptions{})
		if !gpupaas.IsNotFound(err) {
			t.Fatalf("status %d: %v", status, err)
		}
	}
}

func TestMKSClusterProfilesAndNodes(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{})
	ctx := waitCtx(t)
	p, err := cs.V1alpha1().MKSClusterProfiles(testProject).Get(ctx, "mks-oneclick-1", gpupaas.GetOptions{})
	if err != nil || p.Spec.ControlPlane.DefaultSKUName != "bmaas-test-1" || p.Spec.DefaultCNI.Name != "Calico" {
		t.Fatalf("profile: %v %#v", err, p)
	}
	list, err := cs.V1alpha1().MKSClusterProfiles(testProject).List(ctx, gpupaas.ListOptions{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("profiles: %v %#v", err, list)
	}
	if _, err := cs.V1alpha1().MKSClusterProfiles(testProject).Get(ctx, "nope", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("missing profile: %v", err)
	}
	if _, err := cs.V1alpha1().MKSNodes(testProject).ListForCluster(ctx, ""); err == nil {
		t.Fatal("empty cluster name must be rejected")
	}
}

func TestMKSOrganizationHeaderIsSent(t *testing.T) {
	srv := clusterapi.New(clusterapi.Options{Profiles: []apiv1.MKSClusterProfile{mksProfile()}})
	t.Cleanup(srv.Close)
	cfg := gpupaas.NewConfig(srv.URL, "token")
	cfg.OrganizationID = "org-1"
	cs, err := clientset.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.V1alpha1().MKSClusters(testProject).List(context.Background(), gpupaas.ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if orgs := srv.Organizations(); len(orgs) != 1 || orgs[0] != "org-1" {
		t.Fatalf("organizations = %v", orgs)
	}
}

func TestMKSSpecSatisfied(t *testing.T) {
	pool := func(name string, n int32) apiv1.MKSNodeGroup {
		return apiv1.MKSNodeGroup{Name: name, Scaling: &apiv1.MKSScaling{Count: n}}
	}
	got := apiv1.MKSClusterSpec{KubernetesVersion: "v1.36.4", ControlPlane: &apiv1.MKSNodeGroup{Scaling: &apiv1.MKSScaling{Count: 1}}, WorkerPools: []apiv1.MKSNodeGroup{pool("a", 2)}}
	cases := []struct {
		name string
		want apiv1.MKSClusterSpec
		ok   bool
	}{
		{"short version", apiv1.MKSClusterSpec{KubernetesVersion: "1.36", WorkerPools: []apiv1.MKSNodeGroup{pool("a", 2)}}, true},
		{"exact version with v", apiv1.MKSClusterSpec{KubernetesVersion: "v1.36.4", WorkerPools: []apiv1.MKSNodeGroup{pool("a", 2)}}, true},
		{"newer version", apiv1.MKSClusterSpec{KubernetesVersion: "1.37", WorkerPools: []apiv1.MKSNodeGroup{pool("a", 2)}}, false},
		{"prefix is not a match", apiv1.MKSClusterSpec{KubernetesVersion: "1.3", WorkerPools: []apiv1.MKSNodeGroup{pool("a", 2)}}, false},
		{"pool count", apiv1.MKSClusterSpec{WorkerPools: []apiv1.MKSNodeGroup{pool("a", 3)}}, false},
		{"missing pool", apiv1.MKSClusterSpec{WorkerPools: []apiv1.MKSNodeGroup{pool("b", 2)}}, false},
		{"extra pool observed", apiv1.MKSClusterSpec{}, false},
		{"control plane count", apiv1.MKSClusterSpec{ControlPlane: &apiv1.MKSNodeGroup{Scaling: &apiv1.MKSScaling{Count: 3}}, WorkerPools: []apiv1.MKSNodeGroup{pool("a", 2)}}, false},
	}
	for _, c := range cases {
		if typed.MKSSpecSatisfied(c.want, got) != c.ok {
			t.Errorf("%s: want %v", c.name, c.ok)
		}
	}
}

func TestMKSCreateWithoutCNIIsRejectedLikeTheLiveAPI(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{})
	c := oneNodeCluster("nocni")
	c.Spec.Networking = nil
	_, err := cs.V1alpha1().MKSClusters(testProject).Create(context.Background(), c, gpupaas.CreateOptions{})
	if err == nil || !strings.Contains(err.Error(), "spec.networking.cni.name is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestMKSFailedCreateKeepsTheRecordAndNeedsForceToDelete(t *testing.T) {
	// Live behaviour: when starting the provisioning fails, the call fails (404) but the cluster
	// is stored in ERROR; a plain DELETE is refused and force=true removes the record.
	srv, cs := newMKSFixture(t, clusterapi.Options{SSHKeys: []string{"other"}})
	clusters := cs.V1alpha1().MKSClusters(testProject)
	ctx := waitCtx(t)
	_, err := clusters.Create(ctx, oneNodeCluster("kept"), gpupaas.CreateOptions{}) // ssh key "mine" does not exist
	if err == nil || !strings.Contains(err.Error(), `SSH key "mine" doesn't exist`) {
		t.Fatalf("create: %v", err)
	}
	kept, err := clusters.Get(ctx, "kept", gpupaas.GetOptions{})
	if err != nil || kept.Status.Phase() != "error" {
		t.Fatalf("the record must be kept in ERROR: %v %#v", err, kept)
	}
	if err := clusters.Delete(ctx, "kept", gpupaas.DeleteOptions{}); err == nil || !strings.Contains(err.Error(), "force=true") {
		t.Fatalf("plain delete: %v", err)
	}
	if err := clusters.Delete(ctx, "kept", gpupaas.DeleteOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if srv.Exists(testProject, "", "kept") {
		t.Fatal("force delete left the record")
	}
}

// A large cluster has more nodes than one response holds. The list must follow the pages whatever
// the server's own cap is, or a resize would wait for nodes it can never see.
func TestMKSNodeListIsPaged(t *testing.T) {
	for _, tc := range []struct {
		name string
		cap  int
	}{{"no server cap", 0}, {"server caps below the requested size", 60}, {"server caps above the requested size", 1000}} {
		t.Run(tc.name, func(t *testing.T) {
			_, cs := newMKSFixture(t, clusterapi.Options{NodePageSize: tc.cap})
			clusters := cs.V1alpha1().Workspaces(testProject).MKSClusters("ws")
			ctx := waitCtx(t)

			obj := oneNodeCluster("big")
			obj.Spec.WorkerPools = []apiv1.MKSNodeGroup{{
				Name: "w", SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26", SSHKey: "mine",
				Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 249},
			}}
			if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
			if _, err := typed.WaitForMKSCluster(ctx, clusters, "big", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
				t.Fatal(err)
			}
			nodes, err := cs.V1alpha1().MKSNodes(testProject).ListForCluster(ctx, "big")
			if err != nil {
				t.Fatal(err)
			}
			if len(nodes.Items) != 250 {
				t.Fatalf("read %d nodes, want 250", len(nodes.Items))
			}
			seen := map[string]bool{}
			for _, n := range nodes.Items {
				seen[n.Metadata.Name] = true
			}
			if len(seen) != 250 {
				t.Fatalf("%d distinct nodes among 250: pages overlapped or repeated", len(seen))
			}
		})
	}
}

func TestMKSKubeconfigDownload(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	_ = srv
	clusters := cs.V1alpha1().Workspaces(testProject).MKSClusters("ws")
	ctx := waitCtx(t)

	if _, err := clusters.Kubeconfig(ctx, "nope"); !gpupaas.IsNotFound(err) {
		t.Fatalf("a missing cluster must be not found, got %v", err)
	}
	if _, err := clusters.Create(ctx, oneNodeCluster("kc"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	// Not running yet: the sentry service has no such cluster.
	if _, err := clusters.Kubeconfig(ctx, "kc"); err == nil {
		t.Fatal("a cluster that is still provisioning must not give a kubeconfig")
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "kc", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	raw, err := clusters.Kubeconfig(ctx, "kc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "kind: Config") || !strings.Contains(string(raw), "current-context: kc") {
		t.Fatalf("not a kubeconfig of cluster kc: %.80q", raw)
	}
}

func TestMKSPoolOperations(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	clusters := cs.V1alpha1().Workspaces(testProject).MKSClusters("ws")
	nodes := cs.V1alpha1().MKSNodes(testProject)
	ctx := waitCtx(t)

	obj := oneNodeCluster("pc")
	obj.Spec.WorkerPools = []apiv1.MKSNodeGroup{{
		Name: "pool-a", SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26", SSHKey: "mine",
		Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 1},
	}}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	// A cluster that is not running refuses pool operations.
	if _, err := nodes.CordonPool(ctx, "pc", "pool-a"); err == nil {
		t.Fatal("cordon on a provisioning cluster must fail")
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "pc", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}

	if _, err := nodes.CordonPool(ctx, "pc", "pool-a"); err != nil {
		t.Fatal(err)
	}
	if !srv.Cordoned(testProject, "pc", "pool-a") {
		t.Fatal("pool-a is not cordoned")
	}
	if _, err := nodes.UncordonPool(ctx, "pc", "pool-a"); err != nil {
		t.Fatal(err)
	}
	if srv.Cordoned(testProject, "pc", "pool-a") {
		t.Fatal("pool-a is still cordoned")
	}
	force, grace := true, 45
	if _, err := nodes.DrainPool(ctx, "pc", "pool-a", typed.MKSDrainOptions{Force: &force, GracePeriodSeconds: &grace}); err != nil {
		t.Fatal(err)
	}
	ops := srv.PoolOps()
	want := []string{"cordon " + testProject + "/pc/pool-a", "uncordon " + testProject + "/pc/pool-a", "drain " + testProject + "/pc/pool-a"}
	if len(ops) != 3 || ops[0] != want[0] || ops[1] != want[1] || ops[2] != want[2] {
		t.Fatalf("operations = %v, want %v", ops, want)
	}

	if _, err := nodes.CordonPool(ctx, "pc", "no-such-pool"); !gpupaas.IsNotFound(err) {
		t.Fatalf("an unknown pool must be not found, got %v", err)
	}
	if _, err := nodes.CordonPool(ctx, "", "pool-a"); err == nil {
		t.Fatal("an empty cluster name must be refused")
	}
}

// While the cluster is busy the platform answers 412; the SDK reports it as an API error with that status,
// so callers can wait and retry.
func TestMKSPoolOperationsReport412WhileTheClusterIsBusy(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{PoolOpBusy: 1})
	clusters := cs.V1alpha1().Workspaces(testProject).MKSClusters("ws")
	ctx := waitCtx(t)
	obj := oneNodeCluster("busy")
	obj.Spec.WorkerPools = []apiv1.MKSNodeGroup{{
		Name: "pool-a", SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26", SSHKey: "mine",
		Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 1},
	}}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "busy", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	nodes := cs.V1alpha1().MKSNodes(testProject)
	_, err := nodes.CordonPool(ctx, "busy", "pool-a")
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 412 {
		t.Fatalf("got %v, want an API error with status 412", err)
	}
	if _, err := nodes.CordonPool(ctx, "busy", "pool-a"); err != nil {
		t.Fatalf("the second call must succeed: %v", err)
	}
}

// The platform answers a pool operation with one result per node; the SDK returns them as they are,
// including the node's own message.
func TestMKSPoolOperationReturnsThePerNodeResults(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{PoolOpFailedNodes: 1})
	clusters := cs.V1alpha1().Workspaces(testProject).MKSClusters("ws")
	ctx := waitCtx(t)
	obj := oneNodeCluster("res")
	obj.Spec.WorkerPools = []apiv1.MKSNodeGroup{{
		Name: "pool-a", SKUName: "bmaas-test-1", SKUType: "BMAAS", OSImage: "ubuntu-26", SSHKey: "mine",
		Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 2},
	}}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForMKSCluster(ctx, clusters, "res", mksWait(typed.MKSClusterRunning, 2, false)); err != nil {
		t.Fatal(err)
	}
	res, err := cs.V1alpha1().MKSNodes(testProject).CordonPool(ctx, "res", "pool-a")
	if err != nil {
		t.Fatal(err)
	}
	if res.Requested != 2 || res.Succeeded != 1 || res.Failed != 1 || len(res.Results) != 2 || res.Action != "Cordon" || res.NodePool != "pool-a" {
		t.Fatalf("result = %+v", res)
	}
	if res.Results[0].Succeeded || !strings.Contains(res.Results[0].Message, "unable to cordon") || !res.Results[1].Succeeded {
		t.Fatalf("per-node results = %+v", res.Results)
	}
}

// Create asks the backend not to update an existing cluster, so a taken name is refused atomically even
// when another client created it between the existence check and the POST.
func TestMKSCreateSendsFailOnExistsAndUpdateDoesNot(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{})
	ctx := waitCtx(t)
	clusters := cs.V1alpha1().MKSClusters(testProject)
	created, err := clusters.Create(ctx, oneNodeCluster("fe"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	created.Spec.Tags = map[string]string{"k": "v"}
	if _, err := clusters.Update(ctx, created, gpupaas.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	var creates, updates int
	for _, r := range srv.Requests() {
		switch r {
		case "POST /apis/paas.k8smgmt.io/v3/projects/" + testProject + "/clusters?fail-on-exists=true":
			creates++
		case "POST /apis/paas.k8smgmt.io/v3/projects/" + testProject + "/clusters":
			updates++
		}
	}
	if creates != 1 || updates != 1 {
		t.Fatalf("creates %d, updates %d; requests %v", creates, updates, srv.Requests())
	}
}

func TestMKSCreateOfATakenNameIsRefusedByTheBackendAndByTheCheck(t *testing.T) {
	for name, opts := range map[string]clusterapi.Options{
		"backend honors fail-on-exists":  {},
		"backend ignores fail-on-exists": {IgnoreFailOnExists: true},
	} {
		t.Run(name, func(t *testing.T) {
			srv, cs := newMKSFixture(t, opts)
			ctx := waitCtx(t)
			clusters := cs.V1alpha1().MKSClusters(testProject)
			if _, err := clusters.Create(ctx, oneNodeCluster("taken"), gpupaas.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
			second := oneNodeCluster("taken")
			second.Spec.Tags = map[string]string{"intruder": "yes"}
			_, err := clusters.Create(ctx, second, gpupaas.CreateOptions{})
			var apiErr *gpupaas.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 {
				t.Fatalf("want a 409, got %v", err)
			}
			if got := srv.Cluster(testProject, "", "taken"); got == nil || got.Spec.Tags["intruder"] != "" {
				t.Fatalf("the second create changed the existing cluster: %+v", got)
			}
		})
	}
}

// The per-pool fields of the v3.1.41 API round-trip through the typed client and the backend shows the
// password masked.
func TestMKSNodeGroupCarriesPoolNetworkDiskPublicIPAndPassword(t *testing.T) {
	srv, cs := newMKSFixture(t, clusterapi.Options{PerPoolNetwork: true})
	ctx := waitCtx(t)
	clusters := cs.V1alpha1().MKSClusters(testProject)
	no := false
	obj := oneNodeCluster("pools")
	obj.Spec.ControlPlane.Network = &apiv1.MKSNodePoolNetwork{VPC: "v", Subnet: "s"}
	obj.Spec.ControlPlane.UserPassword = "cp-secret"
	obj.Spec.WorkerPools = []apiv1.MKSNodeGroup{{
		Name: "w", SKUName: "bmaas-test-1", SKUType: "VMAAS", OSImage: "ubuntu-26", Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 1},
		Network: &apiv1.MKSNodePoolNetwork{VPC: "v", Subnet: "s", SecurityGroup: "sg"}, BootDiskSizeGb: 30, AssignPublicIP: &no, UserPassword: "w-secret",
	}}
	if _, err := clusters.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	got, err := clusters.Get(ctx, "pools", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w := got.Spec.WorkerPools[0]
	if w.Network == nil || *w.Network != (apiv1.MKSNodePoolNetwork{VPC: "v", Subnet: "s", SecurityGroup: "sg"}) || w.BootDiskSizeGb != 30 || w.AssignPublicIP == nil || *w.AssignPublicIP {
		t.Fatalf("worker pool %+v", w)
	}
	if w.UserPassword != "******" || got.Spec.ControlPlane.UserPassword != "******" {
		t.Fatalf("passwords must read back masked: %q, %q", w.UserPassword, got.Spec.ControlPlane.UserPassword)
	}
	if got.Spec.Networking.VPC != "" || got.Spec.Networking.Subnet != "" {
		t.Fatalf("the cluster-level network must be cleared: %+v", got.Spec.Networking)
	}
	// Sending back what a read returned keeps the stored password.
	got.Spec.Tags = map[string]string{"k": "v"}
	if _, err := clusters.Update(ctx, got, gpupaas.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if srv.PoolPassword(testProject, "", "pools", "w") != "w-secret" || srv.PoolPassword(testProject, "", "pools", "") != "cp-secret" {
		t.Fatal("an update that sent the masked password replaced the stored one")
	}
}

func TestMKSPerPoolRulesOfTheV3141API(t *testing.T) {
	yes, no := true, false
	base := func() *apiv1.MKSCluster {
		o := oneNodeCluster("rules")
		o.Spec.ControlPlane.Network = &apiv1.MKSNodePoolNetwork{VPC: "v", Subnet: "s"}
		return o
	}
	worker := func(mod func(*apiv1.MKSNodeGroup)) apiv1.MKSNodeGroup {
		g := apiv1.MKSNodeGroup{Name: "w", SKUName: "bmaas-test-1", SKUType: "VMAAS", OSImage: "ubuntu-26", Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 1}, Network: &apiv1.MKSNodePoolNetwork{VPC: "v", Subnet: "s"}}
		if mod != nil {
			mod(&g)
		}
		return g
	}
	cases := []struct {
		name string
		edit func(*apiv1.MKSCluster)
		want string
	}{
		{"no network on a new pool", func(o *apiv1.MKSCluster) { o.Spec.ControlPlane.Network = nil }, "Select a VPC and a subnet for this pool"},
		{"subnet missing", func(o *apiv1.MKSCluster) {
			o.Spec.WorkerPools = []apiv1.MKSNodeGroup{worker(func(g *apiv1.MKSNodeGroup) { g.Network.Subnet = "" })}
		}, "choose a subnet in VPC"},
		{"boot disk on bare metal", func(o *apiv1.MKSCluster) {
			o.Spec.WorkerPools = []apiv1.MKSNodeGroup{worker(func(g *apiv1.MKSNodeGroup) { g.SKUType, g.BootDiskSizeGb = "BMAAS", 20 })}
		}, "only virtual machine pools take a boot disk size"},
		{"public IP disagrees", func(o *apiv1.MKSCluster) {
			o.Spec.WorkerPools = []apiv1.MKSNodeGroup{worker(func(g *apiv1.MKSNodeGroup) { g.AssignPublicIP = &yes }), worker(func(g *apiv1.MKSNodeGroup) { g.Name, g.AssignPublicIP = "w2", &no })}
		}, "every worker pool must carry the same value"},
		{"automatic scaling out of order", func(o *apiv1.MKSCluster) {
			o.Spec.WorkerPools = []apiv1.MKSNodeGroup{worker(func(g *apiv1.MKSNodeGroup) {
				g.Scaling = &apiv1.MKSScaling{Mode: "automatic", Min: 3, Desired: 2, Max: 5}
			})}
		}, "min <= desired <= max"},
		{"bad taint effect", func(o *apiv1.MKSCluster) {
			o.Spec.WorkerPools = []apiv1.MKSNodeGroup{worker(func(g *apiv1.MKSNodeGroup) {
				g.Taints = []any{map[string]any{"key": "k", "value": "v", "effect": "Sometimes"}}
			})}
		}, "taint effect"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, cs := newMKSFixture(t, clusterapi.Options{PerPoolNetwork: true})
			o := base()
			c.edit(o)
			_, err := cs.V1alpha1().MKSClusters(testProject).Create(waitCtx(t), o, gpupaas.CreateOptions{})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error mentioning %q, got %v", c.want, err)
			}
		})
	}

	t.Run("network and boot disk are fixed after create", func(t *testing.T) {
		_, cs := newMKSFixture(t, clusterapi.Options{PerPoolNetwork: true})
		ctx := waitCtx(t)
		clusters := cs.V1alpha1().MKSClusters(testProject)
		o := base()
		o.Spec.WorkerPools = []apiv1.MKSNodeGroup{worker(func(g *apiv1.MKSNodeGroup) { g.BootDiskSizeGb = 20 })}
		created, err := clusters.Create(ctx, o, gpupaas.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		moved := *created
		moved.Spec.WorkerPools = []apiv1.MKSNodeGroup{created.Spec.WorkerPools[0]}
		moved.Spec.WorkerPools[0].Network = &apiv1.MKSNodePoolNetwork{VPC: "v", Subnet: "other"}
		if _, err := clusters.Update(ctx, &moved, gpupaas.UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "can't be changed after the pool is created") {
			t.Fatalf("a changed pool network was accepted: %v", err)
		}
		resized := *created
		resized.Spec.WorkerPools = []apiv1.MKSNodeGroup{created.Spec.WorkerPools[0]}
		resized.Spec.WorkerPools[0].BootDiskSizeGb = 40
		if _, err := clusters.Update(ctx, &resized, gpupaas.UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "boot disk size can't be changed") {
			t.Fatalf("a changed boot disk was accepted: %v", err)
		}
	})
}

func TestMKSStatusListsTheUpgradeTargets(t *testing.T) {
	_, cs := newMKSFixture(t, clusterapi.Options{})
	ctx := waitCtx(t)
	o := oneNodeCluster("up")
	o.Spec.KubernetesVersion = "1.35"
	got, err := cs.V1alpha1().MKSClusters(testProject).Create(ctx, o, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"1.36"}; len(got.Status.AvailableKubernetesVersions) != 1 || got.Status.AvailableKubernetesVersions[0] != want[0] {
		t.Fatalf("upgrade targets of 1.35: %v", got.Status.AvailableKubernetesVersions)
	}
}

func TestMKSSpecSatisfiedComparesTheBoundsOfAnAutomaticPool(t *testing.T) {
	want := apiv1.MKSClusterSpec{WorkerPools: []apiv1.MKSNodeGroup{{Name: "a", Scaling: &apiv1.MKSScaling{Mode: "automatic", Min: 1, Desired: 2, Max: 5}}}}
	got := func(sc apiv1.MKSScaling) apiv1.MKSClusterSpec {
		return apiv1.MKSClusterSpec{WorkerPools: []apiv1.MKSNodeGroup{{Name: "a", Scaling: &sc}}}
	}
	if !typed.MKSSpecSatisfied(want, got(apiv1.MKSScaling{Mode: "automatic", Min: 1, Desired: 4, Max: 5})) {
		t.Error("the autoscaler moving the size inside the bounds must not look unsatisfied")
	}
	if typed.MKSSpecSatisfied(want, got(apiv1.MKSScaling{Mode: "automatic", Min: 1, Desired: 2, Max: 6})) {
		t.Error("different bounds were accepted")
	}
	if typed.MKSSpecSatisfied(want, got(apiv1.MKSScaling{Mode: "manual", Count: 2})) {
		t.Error("a manual pool satisfied an automatic one")
	}
	manual := apiv1.MKSClusterSpec{WorkerPools: []apiv1.MKSNodeGroup{{Name: "a", Scaling: &apiv1.MKSScaling{Mode: "manual", Count: 2}}}}
	if !typed.MKSSpecSatisfied(manual, got(apiv1.MKSScaling{Mode: "manual", Count: 2})) || typed.MKSSpecSatisfied(manual, got(apiv1.MKSScaling{Mode: "automatic", Min: 2, Desired: 2, Max: 2})) {
		t.Error("manual pools are compared by count and mode")
	}
}
