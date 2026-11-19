package v1alpha1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBareMetalSpecKeepsUnmodeledFields(t *testing.T) {
	var spec BareMetalSpec
	if err := json.Unmarshal([]byte(`{"type":"sku","image":"img","sshKey":"k","brandNew":{"a":1}}`), &spec); err != nil {
		t.Fatal(err)
	}
	if string(spec.Extra["brandNew"]) != `{"a":1}` {
		t.Fatalf("extra = %#v", spec.Extra)
	}
	spec.Image = "img2"
	out, _ := json.Marshal(spec)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(out, &m)
	if string(m["brandNew"]) != `{"a":1}` || string(m["image"]) != `"img2"` {
		t.Fatalf("output = %s", out)
	}
}

func TestBareMetalStatusPhases(t *testing.T) {
	cases := []struct {
		status                                string
		phase, power                          string
		running, inProgress, failed, deleting bool
	}{
		{"running", "running", "running", true, false, false, false},
		{"RUNNING", "running", "running", true, false, false, false},
		{"Status_RUNNING", "running", "running", true, false, false, false},
		{"stopped", "stopped", "stopped", false, false, false, false},
		{"in_progress", "in_progress", "unknown", false, true, false, false},
		{"pending", "pending", "unknown", false, true, false, false},
		{"", "", "unknown", false, true, false, false},
		{"failed", "failed", "unknown", false, false, true, false},
		{"destroyfailed", "destroyfailed", "unknown", false, false, true, false},
		{"delete_pending", "delete_pending", "unknown", false, false, true, false},
		{"deleting", "deleting", "unknown", false, false, false, true},
	}
	for _, c := range cases {
		s := BareMetalStatus{Status: c.status}
		if s.Phase() != c.phase || s.PowerState() != c.power || s.Running() != c.running || s.InProgress() != c.inProgress || s.Failed() != c.failed || s.Deleting() != c.deleting {
			t.Errorf("%q: phase=%q power=%q running=%v inProgress=%v failed=%v deleting=%v", c.status, s.Phase(), s.PowerState(), s.Running(), s.InProgress(), s.Failed(), s.Deleting())
		}
	}
}

func TestBareMetalDeepCopyIsIndependent(t *testing.T) {
	orig := &BareMetal{
		Metadata: ObjectMeta{Name: "b", Labels: map[string]string{"a": "b"}},
		Spec:     BareMetalSpec{Type: "t", Sharing: &DevSharingSpec{ShareMode: "None"}, Extra: map[string]json.RawMessage{"x": json.RawMessage(`1`)}},
		Status:   BareMetalStatus{Output: &BareMetalOutput{HostName: "h"}},
	}
	cp := orig.DeepCopyObject().(*BareMetal)
	cp.Metadata.Labels["a"] = "changed"
	cp.Spec.Sharing.ShareMode = "All"
	cp.Status.Output.HostName = "other"
	if orig.Metadata.Labels["a"] != "b" || orig.Spec.Sharing.ShareMode != "None" || orig.Status.Output.HostName != "h" {
		t.Fatalf("copy shares state: %#v", orig)
	}
	if string(cp.Spec.Extra["x"]) != "1" {
		t.Fatal("deep copy dropped extras")
	}
}

// A failed instance carries the workflow error on itself, in status.reasonDetail (seen live on
// narayana-tb1), next to the generic reason.
func TestBareMetalStatusKeepsReasonDetail(t *testing.T) {
	in := `{"status":"failed","reason":"Couldn't provision instance b due to an internal error.","reasonDetail":"activity failed: node-provision: Bond configuration failed (exit=124).","action":"deploy"}`
	var st BareMetalStatus
	if err := json.Unmarshal([]byte(in), &st); err != nil {
		t.Fatal(err)
	}
	if !st.Failed() || st.ReasonDetail != "activity failed: node-provision: Bond configuration failed (exit=124)." {
		t.Fatalf("status = %+v", st)
	}
	out, err := json.Marshal(st)
	if err != nil || !strings.Contains(string(out), `"reasonDetail":"activity failed`) {
		t.Fatalf("reasonDetail not written back: %s %v", out, err)
	}
}
