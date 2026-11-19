package convert

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVMInstancePaths(t *testing.T) {
	coll, item, action := VMInstancePaths(DevScope{Project: "p"})
	if coll() != "/apis/dev.envmgmt.io/v1/projects/p/virtualmachines" || item("v") != "/apis/dev.envmgmt.io/v1/projects/p/virtualmachines/v" {
		t.Fatalf("project paths: %s %s", coll(), item("v"))
	}
	if got := action("v", "reboot"); got != "/apis/dev.envmgmt.io/v1/projects/p/virtualmachines/v/action/reboot" {
		t.Fatalf("action path = %s", got)
	}
	coll, item, _ = VMInstancePaths(DevScope{Project: "p", Workspace: "w"})
	if coll() != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/virtualmachines" || item("v") != "/apis/dev.envmgmt.io/v1/projects/p/workspaces/w/virtualmachines/v" {
		t.Fatalf("workspace paths: %s %s", coll(), item("v"))
	}
	pc, pi := VMProfilePaths("p")
	if pc() != "/apis/dev.envmgmt.io/v1/projects/p/vmprofiles" || pi("vm-test") != "/apis/dev.envmgmt.io/v1/projects/p/vmprofiles/vm-test" {
		t.Fatalf("profile paths: %s %s", pc(), pi("vm-test"))
	}
}

func TestFromVMInstanceReadsLiveShapedResponses(t *testing.T) {
	w := loadFixture[VMInstanceObject](t, "vm_instance_running.json")
	v := FromVMInstance(&w, "system-catalog", "")
	s := v.Spec
	if s.Type != "vm-test" || s.CPUCount != "3" || s.Memory != "3" || s.Image != "ubuntu-22-04-system" || s.SSHKey != "my-key" || s.BootDiskSize != 15 || s.Datacenter != "vmaas-functional-dc" {
		t.Fatalf("spec: %#v", s)
	}
	if strings.ReplaceAll(string(s.Extra["something_new"]), " ", "") != `{"a":1}` {
		t.Fatalf("unmodeled spec fields must be kept: %#v", s.Extra)
	}
	if !v.Status.Running() || v.Status.PowerState() != "running" || v.Status.Action != "deploy" {
		t.Fatalf("status: %#v", v.Status)
	}
	o := v.Status.Output
	if o == nil || o.HostName != "h" || o.InstanceID != "i-1" || o.OSName != "ubuntu" || o.PrivateIP != "10.0.0.7" || o.ServerHost != "host-1" || o.UserName != "ubuntu" {
		t.Fatalf("output: %#v", o)
	}
	if v.Metadata.Workspace != "sample-ws" {
		t.Fatalf("metadata: %#v", v.Metadata)
	}
}

func TestToVMInstanceDropsStatusAndWritesSnakeCase(t *testing.T) {
	w := loadFixture[VMInstanceObject](t, "vm_instance_running.json")
	v := FromVMInstance(&w, "system-catalog", "")
	v.Spec.SecurityGroup = "sg1"
	v.Spec.GuestPassword = "s3cret"
	out := ToVMInstance(v, "system-catalog", "ws")
	if out.Status != nil {
		t.Fatal("status must not be sent")
	}
	raw, _ := json.Marshal(out)
	for _, want := range []string{`"security_group":"sg1"`, `"guest_password":"s3cret"`, `"cpu_count":"3"`, `"boot_disk_size":15`, `"ssh_key":"my-key"`,
		`"something_new":{"a":1}`, `"workspace":"sample-ws"`, `"kind":"VirtualMachine"`, `"apiVersion":"dev.envmgmt.io/v1"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %s in %s", want, raw)
		}
	}
	// the legacy resource reference must not be sent: first-class VMs are matched by spec.type
	if strings.Contains(string(raw), "virtual_machine") {
		t.Errorf("the legacy virtual_machine reference leaked into %s", raw)
	}
}

func TestFromVMProfileReadsLiveResponse(t *testing.T) {
	w := loadFixture[VMProfileObject](t, "vmprofile.json")
	p := FromVMProfile(&w, "system-catalog")
	s := p.Spec
	if len(s.Image) != 2 || s.ImageDefault != "ubuntu-24-04-system" || len(s.Datacenter) != 1 || s.PublicIP != "Optional" || s.InstanceType != "ONDEMAND" || s.TemplateVersion != "v3.1.41" {
		t.Fatalf("spec: %#v", s)
	}
	sizes := s.Sizes()
	if s.GPUsEnabled || len(sizes) != 1 || sizes[0].Name != "Small" || sizes[0].CPU != "3" || sizes[0].Memory != "3" {
		t.Fatalf("sizes: %#v", sizes)
	}
	if s.BootDisk == nil || s.BootDisk.MinSize != "15" || s.BootDisk.MaxSize != "2000" || s.BootDisk.Increment != "10" {
		t.Fatalf("boot disk: %#v", s.BootDisk)
	}
	if p.Status.Status != "success" || p.Status.GlobalSettings == nil || len(p.Status.GlobalSettings.Agents) != 1 {
		t.Fatalf("status: %#v", p.Status)
	}
}
