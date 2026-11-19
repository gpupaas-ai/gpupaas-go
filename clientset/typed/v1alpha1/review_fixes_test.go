package v1alpha1_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/mock/vmapi"
)

// A server that caps its pages below ours and reports no total must not cut the node list short.
func TestMKSNodeListContinuesPastAShortPageWhenThereIsNoTotal(t *testing.T) {
	const total, cap = 120, 50
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		items := ""
		for i := offset; i < total && i < offset+cap; i++ {
			if items != "" {
				items += ","
			}
			items += fmt.Sprintf(`{"metadata":{"name":"n%d"}}`, i)
		}
		fmt.Fprintf(w, `{"items":[%s]}`, items)
	}))
	defer srv.Close()
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "t"))
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := cs.V1alpha1().MKSNodes(testProject).ListForCluster(waitCtx(t), "big")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes.Items) != total {
		t.Fatalf("read %d nodes, want %d", len(nodes.Items), total)
	}
}

// A server that ignores offset must not make the list loop through the same page 500 times.
func TestMKSNodeListStopsWhenTheServerIgnoresOffset(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"items":[{"metadata":{"name":"a"}},{"metadata":{"name":"b"}}]}`)
	}))
	defer srv.Close()
	cs, _ := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "t"))
	nodes, err := cs.V1alpha1().MKSNodes(testProject).ListForCluster(waitCtx(t), "c")
	if err != nil || len(nodes.Items) != 2 || calls > 3 {
		t.Fatalf("items=%d calls=%d err=%v", len(nodes.Items), calls, err)
	}
}

// A wait with no Done predicate and no GoneIsDone has nothing to wait for: it fails at once instead of polling until
// the context ends.
func TestWaitWithNothingToWaitForFailsAtOnce(t *testing.T) {
	_, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().Workspaces(testProject).VMInstances("ws")
	obj := &apiv1.VMInstance{Metadata: apiv1.ObjectMeta{Name: "w1"}, Spec: apiv1.VMInstanceSpec{Type: "vm-sku", Image: "ubuntu-24", SSHKey: "key1", CPUCount: "3", Memory: "3"}}
	if _, err := vms.Create(waitCtx(t), obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, err := typed.WaitForVMInstance(ctx, vms, "w1", typed.VMInstanceWaitOptions{Interval: time.Millisecond})
	if err == nil || time.Since(start) > time.Second {
		t.Fatalf("err=%v after %v", err, time.Since(start))
	}
}

// DryRun has no meaning for the API: Delete must refuse it, not delete the object for real.
func TestDeleteRefusesDryRunAndKeepsTheObject(t *testing.T) {
	srv, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().Workspaces(testProject).VMInstances("ws")
	ctx := waitCtx(t)
	obj := &apiv1.VMInstance{Metadata: apiv1.ObjectMeta{Name: "dry"}, Spec: apiv1.VMInstanceSpec{Type: "vm-sku", Image: "ubuntu-24", SSHKey: "key1", CPUCount: "3", Memory: "3"}}
	if _, err := vms.Create(ctx, obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := vms.Delete(ctx, "dry", gpupaas.DeleteOptions{DryRun: true}); err == nil {
		t.Fatal("a dry-run delete must return an error")
	}
	if !srv.Exists(testProject, "ws", "dry") {
		t.Fatal("a dry-run delete removed the machine")
	}
}

// Create must not write the client's scope into the caller's object.
func TestCreateDoesNotModifyTheCallersObject(t *testing.T) {
	_, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().Workspaces(testProject).VMInstances("ws")
	obj := &apiv1.VMInstance{Metadata: apiv1.ObjectMeta{Name: "keep"}, Spec: apiv1.VMInstanceSpec{Type: "vm-sku", Image: "ubuntu-24", SSHKey: "key1", CPUCount: "3", Memory: "3"}}
	if _, err := vms.Create(waitCtx(t), obj, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if obj.Metadata.Project != "" || obj.Metadata.Workspace != "" {
		t.Fatalf("the caller's object was written to: %+v", obj.Metadata)
	}
}

func TestMKSNodeDeepCopyDoesNotShareMapsOrSlices(t *testing.T) {
	n := &apiv1.MKSNode{Spec: apiv1.MKSNodeSpec{
		Roles:      []string{"Worker"},
		NodeLabels: map[string]string{"a": "1"},
		NodeTaints: []apiv1.MKSTaint{{Key: "k"}},
	}}
	cp := n.DeepCopyObject().(*apiv1.MKSNode)
	cp.Spec.Roles[0] = "Master"
	cp.Spec.NodeLabels["a"] = "2"
	cp.Spec.NodeTaints[0].Key = "other"
	if n.Spec.Roles[0] != "Worker" || n.Spec.NodeLabels["a"] != "1" || n.Spec.NodeTaints[0].Key != "k" {
		t.Fatalf("the copy shares state with the original: %+v", n.Spec)
	}
}

// Concurrent creates of one name in a process: one wins, the others get the 409 and nobody overwrites the winner.
func TestConcurrentCreatesOfOneNameAllButOneConflict(t *testing.T) {
	_, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().Workspaces(testProject).VMInstances("ws")
	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			obj := &apiv1.VMInstance{Metadata: apiv1.ObjectMeta{Name: "race"}, Spec: apiv1.VMInstanceSpec{Type: "vm-sku", Image: "ubuntu-24", SSHKey: "key1", CPUCount: "3", Memory: "3"}}
			_, err := vms.Create(waitCtx(t), obj, gpupaas.CreateOptions{})
			errs <- err
		}()
	}
	won, conflicts := 0, 0
	for i := 0; i < n; i++ {
		switch err := <-errs; {
		case err == nil:
			won++
		case gpupaas.IsConflict(err):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if won != 1 || conflicts != n-1 {
		t.Fatalf("won=%d conflicts=%d, want 1 and %d", won, conflicts, n-1)
	}
}
