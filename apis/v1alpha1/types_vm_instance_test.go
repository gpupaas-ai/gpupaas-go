package v1alpha1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVMInstanceSpecKeepsUnmodeledFields(t *testing.T) {
	var spec VMInstanceSpec
	if err := json.Unmarshal([]byte(`{"type":"sku","image":"img","ssh_key":"k","cpu_count":"3","brand_new":{"a":1}}`), &spec); err != nil {
		t.Fatal(err)
	}
	if string(spec.Extra["brand_new"]) != `{"a":1}` || spec.SSHKey != "k" || spec.CPUCount != "3" {
		t.Fatalf("spec = %#v", spec)
	}
	spec.Image = "img2"
	out, _ := json.Marshal(spec)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(out, &m)
	if string(m["brand_new"]) != `{"a":1}` || string(m["image"]) != `"img2"` || string(m["cpu_count"]) != `"3"` {
		t.Fatalf("output = %s", out)
	}
	// the wire is snake_case: the camelCase bare-metal spelling must not appear
	if strings.Contains(string(out), "sshKey") || strings.Contains(string(out), "cpuCount") {
		t.Fatalf("camelCase field in %s", out)
	}
}

func TestVMInstanceStatusPhases(t *testing.T) {
	cases := []struct {
		status                                string
		phase, power                          string
		running, inProgress, failed, deleting bool
	}{
		{"running", "running", "running", true, false, false, false},
		{"Status_RUNNING", "running", "running", true, false, false, false},
		{"stopped", "stopped", "stopped", false, false, false, false},
		{"in_progress", "in_progress", "unknown", false, true, false, false},
		{"", "", "unknown", false, true, false, false},
		{"failed", "failed", "unknown", false, false, true, false},
		{"delete_pending", "delete_pending", "unknown", false, false, true, false},
		{"deleting", "deleting", "unknown", false, false, false, true},
	}
	for _, c := range cases {
		s := VMInstanceStatus{Status: c.status}
		if s.Phase() != c.phase || s.PowerState() != c.power || s.Running() != c.running || s.InProgress() != c.inProgress || s.Failed() != c.failed || s.Deleting() != c.deleting {
			t.Errorf("%q: phase=%q power=%q running=%v inProgress=%v failed=%v deleting=%v", c.status, s.Phase(), s.PowerState(), s.Running(), s.InProgress(), s.Failed(), s.Deleting())
		}
	}
}

func TestVMInstanceDeepCopyIsIndependent(t *testing.T) {
	orig := &VMInstance{
		Metadata: ObjectMeta{Name: "v", Labels: map[string]string{"a": "b"}},
		Spec:     VMInstanceSpec{Type: "t", Sharing: &DevSharingSpec{ShareMode: "None"}, Extra: map[string]json.RawMessage{"x": json.RawMessage(`1`)}},
		Status:   VMInstanceStatus{Output: &VMInstanceOutput{HostName: "h"}},
	}
	cp := orig.DeepCopyObject().(*VMInstance)
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

func TestVMProfileSizesFollowGPUsEnabled(t *testing.T) {
	cpu := []VMPreset{{Name: "Small", CPU: "3", Memory: "3"}}
	gpu := []VMPreset{{Name: "G1", CPU: "8", Memory: "64", GPU: "1"}}
	if got := (VMProfileSpec{CPUPreset: cpu, GPUPreset: gpu}).Sizes(); len(got) != 1 || got[0].Name != "Small" {
		t.Fatalf("cpu sku: %#v", got)
	}
	if got := (VMProfileSpec{GPUsEnabled: true, CPUPreset: cpu, GPUPreset: gpu}).Sizes(); len(got) != 1 || got[0].Name != "G1" {
		t.Fatalf("gpu sku: %#v", got)
	}
}

func TestVMInstanceStatusKeepsReasonDetail(t *testing.T) {
	in := `{"status":"failed","reason":"Couldn't provision instance v.","reasonDetail":"activity failed: vm-provision: no host available","action":"deploy"}`
	var st VMInstanceStatus
	if err := json.Unmarshal([]byte(in), &st); err != nil {
		t.Fatal(err)
	}
	if !st.Failed() || st.ReasonDetail != "activity failed: vm-provision: no host available" {
		t.Fatalf("status = %+v", st)
	}
}

func TestVMInstanceStatusReadsReasonDetailInEitherSpelling(t *testing.T) {
	for _, in := range []string{
		`{"status":"failed","reasonDetail":"no host"}`,
		`{"status":"failed","reason_detail":"no host"}`,
	} {
		var st VMInstanceStatus
		if err := json.Unmarshal([]byte(in), &st); err != nil || st.ReasonDetail != "no host" || !st.Failed() {
			t.Errorf("%s: %+v %v", in, st, err)
		}
	}
	var st VMInstanceStatus
	if err := json.Unmarshal([]byte(`{"status":"running","output":{"host_name":"h","public_ip":"No public IP configured"}}`), &st); err != nil || st.Output == nil || st.Output.HostName != "h" {
		t.Fatalf("output must still decode: %+v %v", st, err)
	}
}
