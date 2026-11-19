package v1alpha1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
)

const testProject = "system-catalog"

func waitCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// computeStub serves the read endpoints of compute instances and the EaaS run details behind them,
// with the shapes of a live controller. Instances are keyed by "workspace/name" ("/name" for project scope).
type computeStub struct {
	*httptest.Server
	mu        sync.Mutex
	requests  []string
	instances map[string]map[string]any
	logs      map[string]string // activity id -> stdout
	activity  []map[string]any
}

func newComputeStub(t *testing.T) (*computeStub, clientset.Interface) {
	t.Helper()
	s := &computeStub{instances: map[string]map[string]any{}, logs: map[string]string{}}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(s.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	return s, cs
}

func (s *computeStub) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func (s *computeStub) put(workspace, name string, status map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := map[string]any{"name": name, "project": testProject}
	if workspace != "" {
		meta["workspace"] = workspace
	}
	s.instances[workspace+"/"+name] = map[string]any{
		"apiVersion": "paas.envmgmt.io/v1", "kind": "ComputeInstance", "metadata": meta, "status": status,
	}
}

func writeStubJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *computeStub) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.RequestURI())
	s.mu.Unlock()
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// /apis/eaas.envmgmt.io/v1/projects/{p}/environments/{env}/events/{trigger}[/activity/{id}/stdout]
	if len(parts) >= 9 && parts[1] == "eaas.envmgmt.io" && parts[5] == "environments" && parts[7] == "events" {
		rest := parts[8:]
		switch {
		case len(rest) == 1:
			writeStubJSON(w, http.StatusOK, map[string]any{"workflowDetails": map[string]any{"activities": s.activity}})
		case len(rest) == 4 && rest[1] == "activity" && rest[3] == "stdout":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(s.logs[rest[2]]))
		default:
			writeStubJSON(w, http.StatusNotFound, map[string]string{"message": "404 page not found"})
		}
		return
	}
	// /apis/paas.envmgmt.io/v1/projects/{p}/[workspaces/{w}/]computeinstances[/{name}]
	if len(parts) < 7 || parts[1] != "paas.envmgmt.io" || parts[3] != "projects" || r.Method != http.MethodGet {
		writeStubJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		return
	}
	rest := parts[5:]
	workspace := ""
	if rest[0] == "workspaces" && len(rest) >= 3 {
		workspace, rest = rest[1], rest[2:]
	}
	if rest[0] != "computeinstances" {
		writeStubJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(rest) == 1 {
		items := []any{}
		for k, v := range s.instances {
			if strings.HasPrefix(k, workspace+"/") {
				items = append(items, v)
			}
		}
		writeStubJSON(w, http.StatusOK, map[string]any{"apiVersion": "paas.envmgmt.io/v1", "kind": "ComputeInstanceList", "metadata": map[string]any{"count": len(items)}, "items": items})
		return
	}
	inst, ok := s.instances[workspace+"/"+rest[1]]
	if !ok {
		// the controller answers a missing compute instance with a 502 and the reason in "internal"
		writeStubJSON(w, http.StatusBadGateway, map[string]any{"internal": "ent: workspace_compute_instance not found", "code": 1, "external": "Upstream misbehaving"})
		return
	}
	writeStubJSON(w, http.StatusOK, inst)
}

func TestComputeInstanceGetAndListWorkspaceScope(t *testing.T) {
	s, cs := newComputeStub(t)
	s.put("ws", "vm1", map[string]any{
		"status": "failed", "reason": "Failed to deploy", "action": "publish",
		"environment":         map[string]any{"name": "env-1", "id": "e1"},
		"provisionedResource": map[string]any{"provisioner": "eaas", "eaas": map[string]any{"triggerId": "t-1"}},
		"workflowStatus":      map[string]any{"tasks": map[string]any{"a": map[string]any{"error": "boom"}}},
	})
	s.put("other", "vm2", map[string]any{"status": "success"})
	ctx := waitCtx(t)
	ci := cs.V1alpha1().Workspaces(testProject).ComputeInstances("ws")

	got, err := ci.Get(ctx, "vm1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "vm1" || got.Metadata.Project != testProject || got.Metadata.Workspace != "ws" {
		t.Fatalf("metadata = %+v", got.Metadata)
	}
	if !got.Status.Failed() || got.Status.Phase() != apiv1.ComputeStatusFailed || got.Status.TriggerID() != "t-1" || got.Status.Reason != "Failed to deploy" {
		t.Fatalf("status = %+v", got.Status)
	}
	if got.Status.WorkflowStatus == nil {
		t.Fatal("workflowStatus should be returned")
	}

	list, err := ci.List(ctx, gpupaas.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "vm1" {
		t.Fatalf("workspace list = %+v", list.Items)
	}
}

func TestComputeInstanceProjectScope(t *testing.T) {
	s, cs := newComputeStub(t)
	s.put("", "p1", map[string]any{"status": "inprogress"})
	ctx := waitCtx(t)
	ci := cs.V1alpha1().ComputeInstances(testProject)
	got, err := ci.Get(ctx, "p1", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Workspace != "" || !got.Status.InProgress() {
		t.Fatalf("project-scoped instance = %+v", got)
	}
	if _, err := cs.V1alpha1().Workspaces(testProject).ComputeInstances("ws").Get(ctx, "p1", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("a project-scoped instance is not in a workspace, got %v", err)
	}
}

func TestComputeInstanceMissingIsNotFound(t *testing.T) {
	_, cs := newComputeStub(t)
	// The controller answers a missing compute instance with 502 and "not found" in the body.
	_, err := cs.V1alpha1().ComputeInstances(testProject).Get(waitCtx(t), "nope", gpupaas.GetOptions{})
	if !gpupaas.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestComputeInstanceFailureLog(t *testing.T) {
	s, cs := newComputeStub(t)
	ctx := waitCtx(t)
	ci := cs.V1alpha1().Workspaces(testProject).ComputeInstances("ws")

	// Never published: no environment, so an empty log and no error.
	s.put("ws", "fl", map[string]any{"status": "not_deployed"})
	if lg, err := ci.FailureLog(ctx, "fl", 0); err != nil || len(lg.Lines) != 0 {
		t.Fatalf("unpublished: %+v, %v", lg, err)
	}

	// Failed run: the warning and error lines are returned; INFO and secret lines are not.
	s.put("ws", "fl", map[string]any{
		"status": "failed", "reason": "Failed to deploy", "action": "publish",
		"environment":         map[string]any{"name": "env-1"},
		"provisionedResource": map[string]any{"eaas": map[string]any{"triggerId": "t-1"}},
	})
	s.activity = []map[string]any{
		{"id": "ACT1", "name": "group.res.custom.task", "completedState": "failed"},
		{"id": "ACT2", "name": "group.res.oncompletion.reconciler", "completedState": "success"},
	}
	s.logs["ACT1"] = strings.Join([]string{
		`time=t level=INFO msg=starting`,
		`time=t level=WARNING msg=Failed to connect: unable to connect to port 22`,
		`time=t level=ERROR msg=token abc leaked`,
		`time=t level=ERROR msg=Failed to deploy`,
	}, "\n")
	lg, err := ci.FailureLog(ctx, "fl", 0)
	if err != nil {
		t.Fatal(err)
	}
	if lg.Environment != "env-1" || len(lg.Activities) != 1 || lg.Activities[0] != "group.res.custom.task" {
		t.Fatalf("log header = %+v", lg)
	}
	want := []string{"WARNING: Failed to connect: unable to connect to port 22", "ERROR: Failed to deploy"}
	if strings.Join(lg.Lines, "|") != strings.Join(want, "|") {
		t.Fatalf("lines = %q, want %q", lg.Lines, want)
	}
	if lg, _ := ci.FailureLog(ctx, "fl", 1); len(lg.Lines) != 1 || lg.Lines[0] != "ERROR: Failed to deploy" {
		t.Fatalf("maxLines should keep the newest line, got %q", lg.Lines)
	}

	// A run in which no activity failed has no lines.
	s.activity = []map[string]any{{"id": "ACT1", "name": "group.res.custom.task", "completedState": "success"}}
	if lg, err := ci.FailureLog(ctx, "fl", 0); err != nil || len(lg.Lines) != 0 {
		t.Fatalf("healthy run: %+v, %v", lg, err)
	}
	if _, err := ci.FailureLog(ctx, "missing", 0); !gpupaas.IsNotFound(err) {
		t.Fatalf("missing instance should be not found, got %v", err)
	}
}

func TestComputeReadsAskForFirstClassInstances(t *testing.T) {
	// The platform hides the compute instances behind first-class objects unless asked.
	s, cs := newComputeStub(t)
	ctx := waitCtx(t)
	s.put("ws", "vm1", map[string]any{"status": "success"})
	ci := cs.V1alpha1().Workspaces(testProject).ComputeInstances("ws")
	_, _ = ci.Get(ctx, "vm1", gpupaas.GetOptions{})
	_, _ = ci.List(ctx, gpupaas.ListOptions{})
	want := map[string]bool{"/computeinstances/vm1?include_first_class=true": false, "/computeinstances?include_first_class=true": false}
	for _, r := range s.Requests() {
		if !strings.HasPrefix(r, "GET ") {
			continue
		}
		for k := range want {
			if strings.HasSuffix(r, k) {
				want[k] = true
			}
		}
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("no GET ended with %s: %v", k, s.Requests())
		}
	}
}
