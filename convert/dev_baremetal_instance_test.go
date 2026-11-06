package convert

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBareMetalPaths(t *testing.T) {
	coll, item, action := BareMetalPaths(DevScope{Project: "p"})
	if coll() != "/apis/dev.envmgmt.io/v1/projects/p/baremetals" || item("b") != "/apis/dev.envmgmt.io/v1/projects/p/baremetals/b" {
		t.Fatalf("project paths: %s %s", coll(), item("b"))
	}
	if got := action("b", "Reinstall OS"); got != "/apis/dev.envmgmt.io/v1/projects/p/baremetals/b/action/Reinstall%20OS" {
		t.Fatalf("action path = %s", got)
	}
	coll, item, _ = BareMetalPaths(DevScope{Project: "p", Workspace: "w"})
	if coll() != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/baremetals" || item("b") != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/baremetals/b" {
		t.Fatalf("workspace paths: %s %s", coll(), item("b"))
	}
}

func TestFromBareMetalReadsLiveResponses(t *testing.T) {
	w := loadFixture[BareMetalObject](t, "baremetal_running.json")
	b := FromBareMetal(&w, "system-catalog", "")
	if b.Spec.Type != "bmaas-ironic-demo" || b.Spec.Image != "bmaas-ironic-img-demo" || b.Spec.SSHKey == "" || b.Spec.Datacenter != "ironic-scale-dc-demo" {
		t.Fatalf("spec: %#v", b.Spec)
	}
	if b.Spec.ProvisionerType != "non_bcm" || b.Spec.ComputeInstanceID == "" || b.Spec.GPUName != "custom" || b.Spec.Sharing == nil || b.Spec.Sharing.ShareMode != "None" {
		t.Fatalf("platform-set fields: %#v", b.Spec)
	}
	if !b.Status.Running() || b.Status.PowerState() != "running" || b.Status.Action != "deploy" || b.Status.Output == nil || b.Status.Output.UserName != "ubuntu" {
		t.Fatalf("status: %#v", b.Status)
	}
	if b.Metadata.Project != "system-catalog" {
		t.Fatalf("metadata: %#v", b.Metadata)
	}

	f := loadFixture[BareMetalObject](t, "baremetal_failed_capacity.json")
	fb := FromBareMetal(&f, "system-catalog", "ws")
	if !fb.Status.Failed() || !strings.Contains(fb.Status.Reason, "no capacity available") || fb.Metadata.Workspace != "notebook-ws" {
		t.Fatalf("failed instance: %#v %#v", fb.Status, fb.Metadata)
	}
}

func TestToBareMetalDropsStatusAndKeepsTheRest(t *testing.T) {
	w := loadFixture[BareMetalObject](t, "baremetal_running.json")
	b := FromBareMetal(&w, "system-catalog", "")
	b.Spec.SecurityGroup = "sg1"
	b.Spec.UserPassword = "s3cret"
	out := ToBareMetal(b, "system-catalog", "ws")
	if out.Status != nil {
		t.Fatal("status must not be sent")
	}
	raw, _ := json.Marshal(out)
	for _, want := range []string{`"securityGroup":"sg1"`, `"userPassword":"s3cret"`, `"computeInstanceId"`, `"workspace":"ws"`, `"kind":"BareMetal"`, `"apiVersion":"dev.envmgmt.io/v1"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %s in %s", want, raw)
		}
	}
}

func TestFromBmProfileReadsLiveResponse(t *testing.T) {
	w := loadFixture[BmProfileObject](t, "bmprofile.json")
	p := FromBmProfile(&w, "system-catalog")
	s := p.Spec
	if s.ProvisionerType != "non_bcm" || len(s.Image) != 1 || s.ImageDefault != "bmaas-ironic-img-demo" || len(s.Datacenter) != 1 || s.AssignPublicIP != "CustomerSelectable" {
		t.Fatalf("spec: %#v", s)
	}
	if len(s.InventoryConfig) != 1 || s.InventoryConfig[0].Key != "server_type" || s.GPUName != "custom" || s.UseCase != "bare_metal" {
		t.Fatalf("inventory/gpu: %#v", s)
	}
	if p.Status.Status != "success" || p.Status.GlobalSettings == nil || len(p.Status.GlobalSettings.Agents) != 1 {
		t.Fatalf("status: %#v", p.Status)
	}
}
