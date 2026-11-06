package convert_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func loadComputeFixture(t *testing.T, name string) *convert.PaaSComputeInstance {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var w convert.PaaSComputeInstance
	if err := json.Unmarshal(data, &w); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return &w
}

// Fixtures are real responses captured from a live controller (values redacted).

func TestComputeInstanceFromWireNotDeployed(t *testing.T) {
	got := convert.FromPaaSComputeInstance(loadComputeFixture(t, "computeinstance_notdeployed.json"), "system-catalog", "")
	if got.Kind != apiv1.KindComputeInstance || got.APIVersion != apiv1.APIVersion {
		t.Fatalf("type meta = %q %q", got.APIVersion, got.Kind)
	}
	if got.Metadata.Name != "tf-e2e-vm-1" || got.Metadata.Project != "system-catalog" || got.Metadata.Workspace != "tf-e2e-ws" {
		t.Fatalf("metadata = %+v", got.Metadata)
	}
	if got.Status.Phase() != apiv1.ComputeStatusNotDeployed || got.Status.Status != "not_deployed" {
		t.Fatalf("status = %q / %q", got.Status.Status, got.Status.Phase())
	}
	if got.Status.InProgress() || got.Status.Failed() {
		t.Fatalf("a not deployed instance is neither in progress nor failed: %+v", got.Status)
	}
}

func TestComputeInstanceFromWireSuccess(t *testing.T) {
	got := convert.FromPaaSComputeInstance(loadComputeFixture(t, "computeinstance_success.json"), "system-catalog", "")
	st := got.Status
	if st.Phase() != apiv1.ComputeStatusSuccess || st.Action != "publish" || st.Reason == "" {
		t.Fatalf("status = %+v", st)
	}
	if st.Environment == nil || st.Environment.Name == "" {
		t.Fatalf("environment = %+v", st.Environment)
	}
	if st.ProvisionedResource == nil || st.ProvisionedResource.Provisioner != "eaas" || st.ProvisionedResource.TriggerID == "" {
		t.Fatalf("provisioned resource = %+v (triggerId is nested under eaas on the wire)", st.ProvisionedResource)
	}
	if st.TriggerID() != st.ProvisionedResource.TriggerID {
		t.Fatalf("TriggerID() = %q, want %q", st.TriggerID(), st.ProvisionedResource.TriggerID)
	}
	if st.InProgress() || st.Failed() {
		t.Fatalf("inprogress=%v failed=%v", st.InProgress(), st.Failed())
	}
}

func TestComputeInstanceFromWireFailures(t *testing.T) {
	failed := convert.FromPaaSComputeInstance(loadComputeFixture(t, "computeinstance_failed.json"), "system-catalog", "")
	if !failed.Status.Failed() || failed.Status.Phase() != apiv1.ComputeStatusFailed || failed.Status.Action != "publish" {
		t.Fatalf("failed status = %+v", failed.Status)
	}
	if !strings.Contains(failed.Status.Reason, "Failed to deploy") {
		t.Fatalf("reason = %q", failed.Status.Reason)
	}
	if failed.Status.WorkflowStatus == nil {
		t.Fatal("workflowStatus should carry the per-task error details")
	}
	df := convert.FromPaaSComputeInstance(loadComputeFixture(t, "computeinstance_destroyfailed.json"), "system-catalog", "")
	if df.Status.Phase() != apiv1.ComputeStatusDestroyFailed || !df.Status.Failed() || df.Status.Action != "destroy" {
		t.Fatalf("destroyfailed status = %+v", df.Status)
	}
	if df.Status.ProvisionedResource != nil {
		t.Fatalf("provisioned resource is absent after a destroy failure: %+v", df.Status.ProvisionedResource)
	}
}

func TestComputeInstancePaths(t *testing.T) {
	col, item := convert.ComputeInstancePaths(convert.DevScope{Project: "p", Workspace: "w"})
	if col() != "/apis/paas.envmgmt.io/v1/projects/p/workspaces/w/computeinstances" {
		t.Fatalf("workspace collection = %s", col())
	}
	if item("vm") != "/apis/paas.envmgmt.io/v1/projects/p/workspaces/w/computeinstances/vm" {
		t.Fatalf("workspace item = %s", item("vm"))
	}
	col, item = convert.ComputeInstancePaths(convert.DevScope{Project: "p"})
	if col() != "/apis/paas.envmgmt.io/v1/projects/p/computeinstances" || item("vm") != "/apis/paas.envmgmt.io/v1/projects/p/computeinstances/vm" {
		t.Fatalf("project paths = %s %s", col(), item("vm"))
	}
}

func TestComputeInstanceListContinue(t *testing.T) {
	list := &convert.PaaSComputeInstanceList{
		Metadata: convert.PaaSListMetadata{Count: 25, Limit: 10, Offset: 10},
		Items:    []convert.PaaSComputeInstance{*loadComputeFixture(t, "computeinstance_success.json")},
	}
	got := convert.FromPaaSComputeInstanceList(list, "system-catalog", "")
	if len(got.Items) != 1 || got.Kind != "ComputeInstanceList" {
		t.Fatalf("list = %+v", got)
	}
	if got.Metadata.Continue != "11" {
		t.Fatalf("continue = %q, want 11", got.Metadata.Continue)
	}
}
