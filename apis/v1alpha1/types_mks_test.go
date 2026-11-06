package v1alpha1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMKSClusterSpecKeepsUnmodeledFields(t *testing.T) {
	in := `{"profile":{"name":"p"},"kubernetesVersion":"v1.36.4","futureField":{"a":1},
	  "controlPlane":{"name":"control-plane","skuName":"s","newThing":"x","scaling":{"count":1,"mode":"manual"}}}`
	var spec MKSClusterSpec
	if err := json.Unmarshal([]byte(in), &spec); err != nil {
		t.Fatal(err)
	}
	if _, ok := spec.Extra["futureField"]; !ok {
		t.Fatalf("top-level unmodeled field dropped: %#v", spec.Extra)
	}
	if _, ok := spec.ControlPlane.Extra["newThing"]; !ok {
		t.Fatalf("node group unmodeled field dropped: %#v", spec.ControlPlane.Extra)
	}

	// A typed change is written; the unmodeled fields come back unchanged.
	spec.KubernetesVersion = "v1.37.0"
	out, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]json.RawMessage
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if string(back["futureField"]) != `{"a":1}` || string(back["kubernetesVersion"]) != `"v1.37.0"` {
		t.Fatalf("unexpected output: %s", out)
	}
	var cp map[string]json.RawMessage
	_ = json.Unmarshal(back["controlPlane"], &cp)
	if string(cp["newThing"]) != `"x"` {
		t.Fatalf("node group extra lost: %s", back["controlPlane"])
	}
}

func TestMKSClusterSpecAlwaysWritesWorkerPools(t *testing.T) {
	// An empty list must reach the backend, otherwise removing the last pool is a no-op.
	out, err := json.Marshal(MKSClusterSpec{KubernetesVersion: "1.36"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(out, &m)
	if string(m["workerPools"]) != "[]" {
		t.Fatalf("workerPools = %s, want []", m["workerPools"])
	}
}

func TestMKSClusterStatusPhase(t *testing.T) {
	cases := []struct {
		status  string
		phase   string
		running bool
		failed  bool
		deleted bool
	}{
		{"MKS_CLUSTER_STATUS_RUNNING", "running", true, false, false},
		{"MKS_CLUSTER_STATUS_ERROR", "error", false, true, false},
		{"MKS_CLUSTER_STATUS_DESTROYFAILED", "destroyfailed", false, true, false},
		{"MKS_CLUSTER_STATUS_DESTROYING", "destroying", false, false, true},
		{"MKS_CLUSTER_STATUS_DESTROYPENDING", "destroypending", false, false, true},
		{"MKS_CLUSTER_STATUS_UPDATING", "updating", false, false, false},
		{"running", "running", true, false, false},
		{"", "", false, false, false},
	}
	for _, c := range cases {
		s := MKSClusterStatus{Status: c.status}
		if s.Phase() != c.phase || s.Running() != c.running || s.Failed() != c.failed || s.Deleting() != c.deleted {
			t.Errorf("%q: phase=%q running=%v failed=%v deleting=%v", c.status, s.Phase(), s.Running(), s.Failed(), s.Deleting())
		}
	}
}

func TestMKSClusterDeepCopyIsIndependent(t *testing.T) {
	orig := &MKSCluster{
		Metadata: ObjectMeta{Name: "c", Labels: map[string]string{"a": "b"}},
		Spec: MKSClusterSpec{
			Tags:         map[string]string{"t": "1"},
			ControlPlane: &MKSNodeGroup{Name: "control-plane", Scaling: &MKSScaling{Count: 1}, Extra: map[string]json.RawMessage{"x": json.RawMessage(`1`)}},
			WorkerPools:  []MKSNodeGroup{{Name: "w", Scaling: &MKSScaling{Count: 2}}},
		},
	}
	cp := orig.DeepCopyObject().(*MKSCluster)
	cp.Metadata.Labels["a"] = "changed"
	cp.Spec.Tags["t"] = "changed"
	cp.Spec.ControlPlane.Scaling.Count = 9
	cp.Spec.WorkerPools[0].Scaling.Count = 9
	if orig.Metadata.Labels["a"] != "b" || orig.Spec.Tags["t"] != "1" || orig.Spec.ControlPlane.Scaling.Count != 1 || orig.Spec.WorkerPools[0].Scaling.Count != 2 {
		t.Fatalf("copy shares state with the original: %#v", orig)
	}
	if string(cp.Spec.ControlPlane.Extra["x"]) != "1" {
		t.Fatalf("deep copy dropped extras: %#v", cp.Spec.ControlPlane.Extra)
	}
}

func TestMKSIntAcceptsNumbersAndQuotedStrings(t *testing.T) {
	for in, want := range map[string]MKSInt{`60`: 60, `"60"`: 60, `0`: 0, `""`: 0, `null`: 0} {
		var g MKSNodeGroup
		if err := json.Unmarshal([]byte(`{"name":"a","bootDiskSizeGb":`+in+`}`), &g); err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if g.BootDiskSizeGb != want {
			t.Errorf("%s = %d, want %d", in, g.BootDiskSizeGb, want)
		}
	}
	var g MKSNodeGroup
	if err := json.Unmarshal([]byte(`{"bootDiskSizeGb":"big"}`), &g); err == nil {
		t.Error("a non-numeric disk size was accepted")
	}
	out, _ := json.Marshal(MKSNodeGroup{Name: "a", BootDiskSizeGb: 60})
	if !strings.Contains(string(out), `"bootDiskSizeGb":60`) {
		t.Errorf("written as %s", out)
	}
}
