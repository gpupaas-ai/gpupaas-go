package v1alpha1

import "testing"

func TestNormalizeComputeStatus(t *testing.T) {
	cases := map[string]string{
		"not_deployed":        ComputeStatusNotDeployed,
		"NOT_DEPLOYED":        ComputeStatusNotDeployed,
		"Status_NOT_DEPLOYED": ComputeStatusNotDeployed,
		"notdeployed":         ComputeStatusNotDeployed,
		"success":             ComputeStatusSuccess,
		"Status_SUCCESS":      ComputeStatusSuccess,
		"inprogress":          ComputeStatusInProgress,
		"actioncomplete":      ComputeStatusActionComplete,
		"destroyfailed":       ComputeStatusDestroyFailed,
		" Destroy-Pending ":   ComputeStatusDestroyPending,
		"":                    "",
	}
	for in, want := range cases {
		if got := NormalizeComputeStatus(in); got != want {
			t.Errorf("NormalizeComputeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComputeStatusClassification(t *testing.T) {
	for _, s := range []string{"pending", "inprogress", "updating", "destroying", "actionpending", "actioninprogress"} {
		if !(ComputeInstanceStatus{Status: s}).InProgress() {
			t.Errorf("%s should be in progress", s)
		}
	}
	for _, s := range []string{"success", "failed", "not_deployed", "actioncomplete", "destroyed", "destroyfailed"} {
		if (ComputeInstanceStatus{Status: s}).InProgress() {
			t.Errorf("%s should not be in progress", s)
		}
	}
	for _, s := range []string{"failed", "updatefailed", "actionfailed", "destroyfailed", "cancelfailed"} {
		if !(ComputeInstanceStatus{Status: s}).Failed() {
			t.Errorf("%s should be failed", s)
		}
	}
	if (ComputeInstanceStatus{Status: "cancelled"}).Failed() {
		t.Error("cancelled is not a failure state on its own")
	}
}

func TestComputeStatusTriggerID(t *testing.T) {
	if got := (ComputeInstanceStatus{}).TriggerID(); got != "" {
		t.Errorf("TriggerID without a provisioned resource = %q, want empty", got)
	}
	st := ComputeInstanceStatus{ProvisionedResource: &ComputeProvisionedResource{TriggerID: "t-1"}}
	if got := st.TriggerID(); got != "t-1" {
		t.Errorf("TriggerID = %q, want t-1", got)
	}
}

func TestComputeInstanceDeepCopyIsIndependent(t *testing.T) {
	orig := &ComputeInstance{
		Metadata: ObjectMeta{Name: "a", Labels: map[string]string{"k": "v"}},
		Status: ComputeInstanceStatus{
			WorkflowStatus:      map[string]any{"w": map[string]any{"x": "1"}},
			Environment:         &ComputeEnvironmentRef{Name: "e"},
			ProvisionedResource: &ComputeProvisionedResource{TriggerID: "t"},
		},
	}
	cp := orig.DeepCopyObject().(*ComputeInstance)
	cp.Metadata.Labels["k"] = "changed"
	cp.Status.WorkflowStatus["w"].(map[string]any)["x"] = "changed"
	cp.Status.Environment.Name = "changed"
	cp.Status.ProvisionedResource.TriggerID = "changed"
	if orig.Metadata.Labels["k"] != "v" || orig.Status.WorkflowStatus["w"].(map[string]any)["x"] != "1" ||
		orig.Status.Environment.Name != "e" || orig.Status.ProvisionedResource.TriggerID != "t" {
		t.Fatalf("DeepCopyObject shares memory with the original: %+v", orig)
	}
}
