package convert

import (
	"encoding/json"
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

func TestDevResourcePaths(t *testing.T) {
	for name, tc := range map[string]struct {
		paths func(DevScope) (func() string, func(string) string)
		kind  string
	}{
		"ssh key":        {SshKeyPaths, "sshkeys"},
		"security group": {SecurityGroupPaths, "securitygroups"},
	} {
		coll, item := tc.paths(DevScope{Project: "p"})
		if coll() != "/apis/dev.envmgmt.io/v1/projects/p/"+tc.kind || item("x y") != "/apis/dev.envmgmt.io/v1/projects/p/"+tc.kind+"/x%20y" {
			t.Fatalf("%s project paths: %s %s", name, coll(), item("x y"))
		}
		coll, item = tc.paths(DevScope{Project: "p", Workspace: "w"})
		if coll() != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/"+tc.kind || item("k") != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/"+tc.kind+"/k" {
			t.Fatalf("%s workspace paths: %s %s", name, coll(), item("k"))
		}
	}
}

func TestFromDevSecurityGroupReadsLiveShapedResponse(t *testing.T) {
	w := loadFixture[DevSecurityGroup](t, "securitygroup_live.json")
	g := FromDevSecurityGroup(&w, "")
	if g.Metadata.Name != "aa" || g.Metadata.Project != "system-catalog" || g.Metadata.Workspace != "notebook-ws" {
		t.Fatalf("metadata: %#v", g.Metadata)
	}
	if len(g.Spec.Rules) != 2 || g.Spec.Rules[0].SourceCIDR != "0.0.0.0/0" || g.Spec.Rules[0].Application != "ssh" || g.Spec.Rules[1].ApplicationPort != "8080" {
		t.Fatalf("rules: %#v", g.Spec.Rules)
	}
	if g.Spec.Sharing == nil || g.Spec.Sharing.ShareMode != "None" || g.Status.Status != "success" {
		t.Fatalf("sharing/status: %#v %#v", g.Spec.Sharing, g.Status)
	}
}

func TestToDevSecurityGroupWritesSnakeCaseRulesAndNoServerFields(t *testing.T) {
	g := &apiv1.SecurityGroup{
		Metadata: apiv1.ObjectMeta{Name: "web", CreatedBy: &apiv1.UserMeta{Username: "x"}},
		Spec: apiv1.SecurityGroupSpec{Rules: []apiv1.Rule{
			{SourceCIDR: "0.0.0.0/0", Application: "custom", ApplicationPort: "443", Protocol: "tcp", Action: "allow"},
		}},
	}
	b, err := json.Marshal(ToDevSecurityGroup(g, "p", "w"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	meta := m["metadata"].(map[string]any)
	if meta["project"] != "p" || meta["workspace"] != "w" || meta["createdBy"] != nil {
		t.Fatalf("metadata: %v", meta)
	}
	rule := m["spec"].(map[string]any)["rules"].([]any)[0].(map[string]any)
	for k, want := range map[string]string{"source_cidr": "0.0.0.0/0", "application": "custom", "application_port": "443", "protocol": "tcp", "action": "allow"} {
		if rule[k] != want {
			t.Fatalf("rule[%s] = %v, want %s (%s)", k, rule[k], want, b)
		}
	}
	if _, ok := m["spec"].(map[string]any)["type"]; ok {
		t.Fatalf("an unset type must not be written: %s", b)
	}
}

func TestSshKeyRoundTrip(t *testing.T) {
	w := loadFixture[DevSshKey](t, "sshkey_live.json")
	k := FromDevSshKey(&w, "")
	if k.Metadata.Name != "tf-key" || k.Metadata.Workspace != "tf-ws" || k.Spec.PublicKey == "" || k.Status.Status != "success" {
		t.Fatalf("ssh key: %#v", k)
	}
	b, _ := json.Marshal(ToDevSshKey(k, "system-catalog", "tf-ws"))
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	spec := m["spec"].(map[string]any)
	if spec["public_key"] != k.Spec.PublicKey {
		t.Fatalf("public_key must be written snake_case: %s", b)
	}
	if _, ok := spec["ssh_key"]; ok {
		t.Fatalf("the wire has no ssh_key reference: %s", b)
	}
}

func TestWorkspaceRoundTripKeepsDisplayNameAndDescriptionInMetadata(t *testing.T) {
	w := loadFixture[PaaSWorkspace](t, "workspace_live.json")
	ws := FromPaaSWorkspace(&w)
	if ws.Metadata.DisplayName != "Team A" || ws.Metadata.Description != "shared workspace" || ws.Metadata.Labels["team"] != "a" {
		t.Fatalf("workspace: %#v", ws)
	}
	out := ToPaaSWorkspace(ws, "system-catalog")
	if out.Metadata.DisplayName != "Team A" || out.Metadata.Description != "shared workspace" || out.APIVersion != PaaSWorkspaceAPIVer {
		t.Fatalf("wire: %#v", out)
	}
}
