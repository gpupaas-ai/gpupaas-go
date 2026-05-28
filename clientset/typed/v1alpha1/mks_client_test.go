package v1alpha1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestMKSClusterClientCRUD(t *testing.T) {
	const base = "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.PaaSMKSClusterList{
				APIVersion: convert.PaaSMKSAPIVersion,
				Kind:       convert.PaaSMKSClusterListKind,
				Items: []convert.PaaSMKSCluster{{
					APIVersion: convert.PaaSMKSAPIVersion,
					Kind:       convert.PaaSMKSClusterKind,
					Metadata:   convert.PaaSMKSMetadata{Name: "c1", Project: "demo"},
					Spec:       convert.PaaSMKSClusterSpec{KubernetesVersion: "1.31"},
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == base+"/c1":
			_ = json.NewEncoder(w).Encode(convert.PaaSMKSCluster{
				APIVersion: convert.PaaSMKSAPIVersion,
				Kind:       convert.PaaSMKSClusterKind,
				Metadata:   convert.PaaSMKSMetadata{Name: "c1", Project: "demo"},
				Spec:       convert.PaaSMKSClusterSpec{KubernetesVersion: "1.31"},
				Status:     convert.PaaSMKSClusterStatus{Condition: "MKS_CLUSTER_STATUS_RUNNING"},
			})
		case r.Method == http.MethodPost && r.URL.Path == base:
			var in convert.PaaSMKSCluster
			_ = json.NewDecoder(r.Body).Decode(&in)
			in.Status = convert.PaaSMKSClusterStatus{Condition: "MKS_CLUSTER_STATUS_PROVISIONING"}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodDelete && r.URL.Path == base+"/c1":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	clusters := cs.V1alpha1().MKSClusters("demo")

	list, err := clusters.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Spec.KubernetesVersion != "1.31" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := clusters.Get(ctx, "c1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "c1" || got.Status.Condition != "MKS_CLUSTER_STATUS_RUNNING" {
		t.Fatalf("get: %+v", got)
	}

	created, err := clusters.Create(ctx, &apiv1.MKSCluster{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
		Metadata: apiv1.ObjectMeta{Name: "c-new", Project: "demo"},
		Spec: apiv1.MKSClusterSpec{
			KubernetesVersion: "1.31",
			CNI:               "calico",
			Nodes: []apiv1.MKSNodeSpec{
				{Hostname: "master-1", PrivateIP: "10.0.0.10", Roles: []string{"master", "worker"}},
			},
		},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Metadata.Name != "c-new" || created.Status.Condition != "MKS_CLUSTER_STATUS_PROVISIONING" {
		t.Fatalf("created: %+v", created)
	}

	if err := clusters.Delete(ctx, "c1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestMKSClusterClientSubActions(t *testing.T) {
	const base = "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1"

	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond := func() {
			_ = json.NewEncoder(w).Encode(convert.PaaSMKSCluster{
				APIVersion: convert.PaaSMKSAPIVersion,
				Kind:       convert.PaaSMKSClusterKind,
				Metadata:   convert.PaaSMKSMetadata{Name: "c1", Project: "demo"},
			})
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base+"/upgrade":
			calls["upgrade"]++
			var req convert.PaaSMKSUpgradeRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.K8sVersion != "1.32" {
				t.Errorf("upgrade: expected k8sVersion=1.32, got %q", req.K8sVersion)
			}
			respond()
		case r.Method == http.MethodPost && r.URL.Path == base+"/scaleNodeGroup":
			calls["scaleNodeGroup"]++
			var req convert.PaaSMKSScaleNodeGroupRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.NodeGroupName != "wng-1" {
				t.Errorf("scaleNodeGroup: expected nodeGroupName=wng-1, got %q", req.NodeGroupName)
			}
			respond()
		case r.Method == http.MethodPost && r.URL.Path == base+"/addNodeGroup":
			calls["addNodeGroup"]++
			var req convert.PaaSMKSAddNodeGroupRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.NodeGroup == nil || req.NodeGroup.ID != "wng-2" {
				t.Errorf("addNodeGroup: unexpected body %+v", req)
			}
			respond()
		case r.Method == http.MethodPost && r.URL.Path == base+"/removeNodeGroup":
			calls["removeNodeGroup"]++
			var req convert.PaaSMKSRemoveNodeGroupRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.NodeGroupName != "wng-1" {
				t.Errorf("removeNodeGroup: expected nodeGroupName=wng-1, got %q", req.NodeGroupName)
			}
			respond()
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	clusters := cs.V1alpha1().MKSClusters("demo")

	if _, err := clusters.Upgrade(ctx, "c1", &apiv1.MKSUpgradeRequest{K8sVersion: "1.32"}, gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	desired := int32(3)
	if _, err := clusters.ScaleNodeGroup(ctx, "c1", &apiv1.MKSScaleNodeGroupRequest{NodeGroupName: "wng-1", DesiredCount: &desired}, gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := clusters.AddNodeGroup(ctx, "c1", &apiv1.MKSNodeGroup{ID: "wng-2", SKU: "small"}, gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := clusters.RemoveNodeGroup(ctx, "c1", "wng-1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}

	// Validation errors should not hit the server.
	if _, err := clusters.ScaleNodeGroup(ctx, "c1", &apiv1.MKSScaleNodeGroupRequest{}, gpupaas.ActionOptions{}); err == nil {
		t.Fatal("expected error for missing nodeGroupName")
	}
	if _, err := clusters.RemoveNodeGroup(ctx, "c1", "", gpupaas.ActionOptions{}); err == nil {
		t.Fatal("expected error for empty nodeGroupName")
	}

	for _, want := range []string{"upgrade", "scaleNodeGroup", "addNodeGroup", "removeNodeGroup"} {
		if calls[want] != 1 {
			t.Errorf("expected exactly 1 call to %s, got %d", want, calls[want])
		}
	}
}

func TestMKSNodeClient(t *testing.T) {
	const base = "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/mksnodes"

	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node := convert.PaaSMKSNode{
			APIVersion: convert.PaaSMKSAPIVersion,
			Kind:       convert.PaaSMKSNodeKind,
			Metadata:   convert.PaaSMKSMetadata{Name: "master-1", Project: "demo"},
			Status:     convert.PaaSMKSNodeStatus{Phase: "MKS_NODE_PHASE_RUNNING"},
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.PaaSMKSNodeList{Items: []convert.PaaSMKSNode{node}})
		case r.Method == http.MethodGet && r.URL.Path == base+"/master-1":
			_ = json.NewEncoder(w).Encode(node)
		case r.Method == http.MethodPost && r.URL.Path == base+"/master-1/drain":
			calls["drain"]++
			var req convert.PaaSMKSDrainRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Force == nil || !*req.Force {
				t.Errorf("drain: expected force=true")
			}
			_ = json.NewEncoder(w).Encode(node)
		case r.Method == http.MethodPost && r.URL.Path == base+"/master-1/cordon":
			calls["cordon"]++
			_ = json.NewEncoder(w).Encode(node)
		case r.Method == http.MethodPost && r.URL.Path == base+"/master-1/uncordon":
			calls["uncordon"]++
			_ = json.NewEncoder(w).Encode(node)
		case r.Method == http.MethodDelete && r.URL.Path == base+"/master-1":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	nodes := cs.V1alpha1().MKSClusters("demo").Nodes("c1")

	list, err := nodes.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "master-1" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := nodes.Get(ctx, "master-1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != "MKS_NODE_PHASE_RUNNING" {
		t.Fatalf("get: %+v", got)
	}

	force := true
	if _, err := nodes.Drain(ctx, "master-1", &apiv1.MKSDrainRequest{Force: &force}, gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.Cordon(ctx, "master-1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.Uncordon(ctx, "master-1", gpupaas.ActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := nodes.Delete(ctx, "master-1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"drain", "cordon", "uncordon"} {
		if calls[want] != 1 {
			t.Errorf("expected exactly 1 call to %s, got %d", want, calls[want])
		}
	}
}

func TestMKSWorkerNodeGroupClient(t *testing.T) {
	const base = "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/workernodegroups"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wng := convert.PaaSMKSWorkerNodeGroup{
			APIVersion: convert.PaaSMKSAPIVersion,
			Kind:       convert.PaaSMKSWorkerNodeGroupKind,
			Metadata:   convert.PaaSMKSMetadata{Name: "wng-1", Project: "demo"},
			Spec: convert.PaaSMKSWorkerNodeGroupSpec{
				ClusterName: "c1",
				NodeGroup:   &convert.PaaSMKSNodeGroup{ID: "wng-1", SKU: "medium"},
			},
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.PaaSMKSWorkerNodeGroupList{Items: []convert.PaaSMKSWorkerNodeGroup{wng}})
		case r.Method == http.MethodGet && r.URL.Path == base+"/wng-1":
			_ = json.NewEncoder(w).Encode(wng)
		case r.Method == http.MethodPost && r.URL.Path == base:
			var in convert.PaaSMKSWorkerNodeGroup
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.Spec.ClusterName != "c1" {
				t.Errorf("create: expected clusterName=c1, got %q", in.Spec.ClusterName)
			}
			_ = json.NewEncoder(w).Encode(in)
		case r.Method == http.MethodDelete && r.URL.Path == base+"/wng-1":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	wngs := cs.V1alpha1().MKSClusters("demo").WorkerNodeGroups("c1")

	list, err := wngs.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "wng-1" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := wngs.Get(ctx, "wng-1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.NodeGroup == nil || got.Spec.NodeGroup.SKU != "medium" {
		t.Fatalf("get: %+v", got)
	}

	created, err := wngs.Create(ctx, &apiv1.MKSWorkerNodeGroup{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSWorkerNodeGroup},
		Metadata: apiv1.ObjectMeta{Name: "wng-2", Project: "demo"},
		Spec:     apiv1.MKSWorkerNodeGroupSpec{NodeGroup: &apiv1.MKSNodeGroup{ID: "wng-2", SKU: "small"}},
	}, gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Spec.ClusterName != "c1" {
		t.Fatalf("created: %+v", created)
	}

	if err := wngs.Delete(ctx, "wng-1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestMKSAuditEventClient(t *testing.T) {
	const base = "/apis/paas.envmgmt.io/v1/projects/demo/mksclusters/c1/auditevents"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evt := convert.PaaSMKSAuditEvent{
			ID:           "evt-1",
			ResourceType: "MKSCluster",
			ResourceName: "c1",
			Action:       "create",
			Actor:        "alice",
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(convert.PaaSMKSAuditEventList{Items: []convert.PaaSMKSAuditEvent{evt}})
		case r.Method == http.MethodGet && r.URL.Path == base+"/evt-1":
			_ = json.NewEncoder(w).Encode(evt)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	events := cs.V1alpha1().MKSClusters("demo").AuditEvents("c1")

	list, err := events.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "evt-1" {
		t.Fatalf("list: %+v", list.Items)
	}

	got, err := events.Get(ctx, "evt-1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "create" || got.Actor != "alice" {
		t.Fatalf("get: %+v", got)
	}
}
