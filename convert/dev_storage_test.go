package convert

import (
	"encoding/json"
	"strings"
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

func TestStoragePaths(t *testing.T) {
	coll, item := StoragePaths(DevScope{Project: "p"})
	if coll() != "/apis/dev.envmgmt.io/v1/projects/p/storages" || item("a b") != "/apis/dev.envmgmt.io/v1/projects/p/storages/a%20b" {
		t.Fatalf("project paths: %s %s", coll(), item("a b"))
	}
	coll, item = StoragePaths(DevScope{Project: "p", Workspace: "w"})
	if coll() != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/storages" || item("s") != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/storages/s" {
		t.Fatalf("workspace paths: %s %s", coll(), item("s"))
	}
}

func TestFromDevStorageReadsALiveShapedFilesystem(t *testing.T) {
	w := loadFixture[DevStorage](t, "storage_file_live.json")
	s := FromDevStorage(&w, "")
	if s.Metadata.Name != "weka-fs-demo" || s.Metadata.Project != "system-catalog" || s.Metadata.Workspace != "" {
		t.Fatalf("metadata: %#v", s.Metadata)
	}
	want := apiv1.StorageSpec{Type: "weka-fs", Size: "100", Datacenter: "lab-dc", StorageType: "file", EnableEncryptionAtRest: "true", EnableEncryptionInTransit: "false"}
	if s.Spec != want {
		t.Fatalf("spec: %#v", s.Spec)
	}
	if !s.Status.Ready() || s.Status.Action != "deploy" || s.Status.Reason != "Storage Deployment Successful" {
		t.Fatalf("status: %#v", s.Status)
	}
	// a number stays an integer string (no 1e+11), strings stay as they are
	if got := s.Status.Output["volume_size"]; got != "100000000000" {
		t.Fatalf("volume_size %q", got)
	}
	if s.Status.Output["filesystem_servers"] != "10.0.0.7" || s.Status.Output["mount_password"] != "<masked>" || len(s.Status.Output) != 9 {
		t.Fatalf("output: %#v", s.Status.Output)
	}
}

func TestFromDevStorageReadsABlockVolumeWithoutASizeAndAFailedRecord(t *testing.T) {
	w := loadFixture[DevStorage](t, "storage_block_live.json")
	s := FromDevStorage(&w, "")
	if s.Metadata.Workspace != "lab-ws" || s.Spec.Size != "" || s.Spec.StorageType != "" || s.Status.Output["volume_size"] != "2000000000" {
		t.Fatalf("block: %#v", s)
	}
	w = loadFixture[DevStorage](t, "storage_failed_live.json")
	s = FromDevStorage(&w, "")
	if !s.Status.Failed() || s.Status.Ready() || s.Status.Output != nil || !strings.Contains(s.Status.Reason, "internal error") {
		t.Fatalf("failed: %#v", s.Status)
	}
	// the platform stamps the profile name into storage_type for some profiles: it is carried as it is
	if s.Spec.StorageType != "hss-profile" {
		t.Fatalf("storage_type %q", s.Spec.StorageType)
	}
}

// Observed on a live controller (narayana-tb1, an object profile whose backend was unavailable): the volume goes
// in_progress, then fails on deploy; a delete of the failed volume fails once on destroy, and the next delete works.
func TestLiveFailuresReadAsFailedWithTheirActionAndReason(t *testing.T) {
	deploy := FromDevStorage(ptr(loadFixture[DevStorage](t, "storage_provision_unavailable_live.json")), "")
	if !deploy.Status.Failed() || deploy.Status.Action != "deploy" || deploy.Status.Output != nil || deploy.Spec.Size != "1" {
		t.Fatalf("deploy failure: %#v", deploy)
	}
	destroy := FromDevStorage(ptr(loadFixture[DevStorage](t, "storage_destroy_failed_live.json")), "")
	if !destroy.Status.Failed() || destroy.Status.Phase() != apiv1.StoragePhaseFailed || destroy.Status.Action != "destroy" {
		t.Fatalf("destroy failure: %#v", destroy)
	}
}

func ptr[T any](v T) *T { return &v }

func TestToDevStorageWritesSnakeCaseNamesTheProfileAgainAndNoServerFields(t *testing.T) {
	s := &apiv1.Storage{
		Metadata: apiv1.ObjectMeta{Name: "data", CreatedBy: &apiv1.UserMeta{Username: "x"}},
		Spec: apiv1.StorageSpec{Type: "weka-fs", Size: "50", Datacenter: "dc", StorageType: "file", AccessPolicy: "rw", ContractTerm: "12m",
			EnableEncryptionAtRest: "true", Sharing: &apiv1.DevSharingSpec{ShareMode: "None"}},
		Status: apiv1.StorageStatus{Status: "success", Output: map[string]string{"k": "v"}},
	}
	b, err := json.Marshal(ToDevStorage(s, "p", "w"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	spec := got["spec"].(map[string]any)
	for k, want := range map[string]string{"type": "weka-fs", "size": "50", "datacenter": "dc", "storage_type": "file", "access_policy": "rw",
		"contract_term": "12m", "enable_encryption_at_rest": "true"} {
		if spec[k] != want {
			t.Errorf("spec.%s = %v, want %s", k, spec[k], want)
		}
	}
	if ref, _ := spec["storage"].(map[string]any); ref["name"] != "weka-fs" {
		t.Errorf("spec.storage = %v, want the profile named again", spec["storage"])
	}
	if _, ok := spec["enable_encryption_in_transit"]; ok {
		t.Error("an unset flag must be left out")
	}
	if _, ok := got["status"].(map[string]any)["status"]; ok {
		t.Errorf("status is observed, never written: %v", got["status"])
	}
	m := got["metadata"].(map[string]any)
	if m["project"] != "p" || m["workspace"] != "w" || m["createdBy"] != nil {
		t.Errorf("metadata: %v", m)
	}
}

func TestOutputValuesOfAnyTypeBecomeStrings(t *testing.T) {
	var w DevStorage
	raw := `{"metadata":{"name":"x"},"status":{"status":"success","output":{"big":12345678901234,"frac":1.5,"flag":true,"none":null,"nested":{"a":[1,2]},"text":"t"}}}`
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		t.Fatal(err)
	}
	out := FromDevStorage(&w, "").Status.Output
	want := map[string]string{"big": "12345678901234", "frac": "1.5", "flag": "true", "none": "", "nested": `{"a":[1,2]}`, "text": "t"}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("output[%s] = %q, want %q", k, out[k], v)
		}
	}
}

func TestOutputNumbersKeepEveryDigit(t *testing.T) {
	var w DevStorage
	// above 2^53 a float64 would round these
	raw := `{"metadata":{"name":"x"},"status":{"status":"success","output":{"volume_size":9007199254740993,"id":18446744073709551615}}}`
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		t.Fatal(err)
	}
	out := FromDevStorage(&w, "").Status.Output
	if out["volume_size"] != "9007199254740993" || out["id"] != "18446744073709551615" {
		t.Fatalf("output: %#v", out)
	}
}

func TestFromDevStorageListKeepsEveryItemAndTheContinuation(t *testing.T) {
	a := loadFixture[DevStorage](t, "storage_file_live.json")
	b := loadFixture[DevStorage](t, "storage_block_live.json")
	l := FromDevStorageList(&DevStorageList{Metadata: PaaSListMetadata{Count: 5, Offset: 0}, Items: []DevStorage{a, b}}, "")
	if len(l.Items) != 2 || l.Items[1].Metadata.Name != "ddn-hss" || l.Metadata.Continue != "2" {
		t.Fatalf("list: %#v", l)
	}
	if empty := FromDevStorageList(nil, ""); empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("a nil list must be an empty one: %#v", empty)
	}
}

func TestStorageStatusPhases(t *testing.T) {
	for status, want := range map[string]struct{ ready, inProgress, failed, deleting bool }{
		"success":        {ready: true},
		"SUCCESS":        {ready: true},
		"status_success": {ready: true},
		"":               {inProgress: true},
		"in_progress":    {inProgress: true},
		"failed":         {failed: true},
		"destroyfailed":  {failed: true},
		"delete_pending": {failed: true},
		"deleting":       {deleting: true},
	} {
		st := apiv1.StorageStatus{Status: status}
		if st.Ready() != want.ready || st.InProgress() != want.inProgress || st.Failed() != want.failed || st.Deleting() != want.deleting {
			t.Errorf("%q: ready %v inProgress %v failed %v deleting %v", status, st.Ready(), st.InProgress(), st.Failed(), st.Deleting())
		}
	}
}
