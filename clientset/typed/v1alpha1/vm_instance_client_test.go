package v1alpha1_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/mock/vmapi"
)

// vmSKU mirrors the live VmProfile vm-test: one CPU preset, a boot disk range, optional public IP.
func vmSKU() apiv1.VMProfile {
	return apiv1.VMProfile{
		Metadata: apiv1.ObjectMeta{Name: "vm-sku"},
		Spec: apiv1.VMProfileSpec{
			Image:        []string{"ubuntu-24", "ubuntu-22"},
			ImageDefault: "ubuntu-24",
			Datacenter:   []string{"dc1"},
			CPUPreset:    []apiv1.VMPreset{{Name: "Small", CPU: "3", Memory: "3"}, {Name: "Medium", CPU: "6", Memory: "12"}},
			BootDisk:     &apiv1.VMBootDisk{MinSize: "15", MaxSize: "2000", Increment: "10", Unit: "gb"},
			PublicIP:     "Optional",
			InstanceType: "ONDEMAND",
		},
		Status: apiv1.VMProfileStatus{GlobalSettings: &apiv1.VMProfileGlobalSettings{Agents: []string{"agent-a"}}},
	}
}

func newVMFixture(t *testing.T, opts vmapi.Options, profiles ...apiv1.VMProfile) (*vmapi.Server, clientset.Interface) {
	t.Helper()
	if len(profiles) == 0 {
		profiles = []apiv1.VMProfile{vmSKU()}
	}
	opts.Profiles = profiles
	if opts.SSHKeys == nil {
		opts.SSHKeys = []string{"key1"}
	}
	srv := vmapi.New(opts)
	t.Cleanup(srv.Close)
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	return srv, cs
}

func newVM(name string) *apiv1.VMInstance {
	return &apiv1.VMInstance{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindVMInstance},
		Metadata: apiv1.ObjectMeta{Name: name},
		Spec:     apiv1.VMInstanceSpec{Type: "vm-sku", CPUCount: "3", Memory: "3", Image: "ubuntu-24", SSHKey: "key1"},
	}
}

func vmWait(done func(*apiv1.VMInstance) (bool, error), stable int, gone bool) typed.VMInstanceWaitOptions {
	return typed.VMInstanceWaitOptions{Interval: 5 * time.Millisecond, Done: done, Stable: stable, GoneIsDone: gone}
}

func TestVMInstanceFullLifecycleWorkspaceScope(t *testing.T) {
	srv, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().Workspaces(testProject).VMInstances("ws")
	ctx := waitCtx(t)

	created, err := vms.Create(ctx, newVM("v1"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status.Phase() != "in_progress" || created.Metadata.Workspace != "ws" || created.Metadata.Project != testProject {
		t.Fatalf("created: phase=%q meta=%#v", created.Status.Phase(), created.Metadata)
	}
	// The platform fills in what the SKU decides.
	if sp := created.Spec; sp.Datacenter != "dc1" || sp.Sharing == nil || sp.Sharing.ShareMode != "None" {
		t.Fatalf("derived fields: %#v", sp)
	}
	// Create never takes over an existing name.
	_, err = vms.Create(ctx, newVM("v1"), gpupaas.CreateOptions{})
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("second create: %v", err)
	}

	running, err := typed.WaitForVMInstance(ctx, vms, "v1", vmWait(typed.VMInstanceRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	if o := running.Status.Output; o == nil || o.HostName == "" || o.UserName != "ubuntu" || o.InstanceID == "" || running.Status.PowerState() != "running" {
		t.Fatalf("running: %#v", running.Status)
	}

	// In-place update (a security group): the VM stays up.
	upd := running.DeepCopyObject().(*apiv1.VMInstance)
	upd.Spec.SecurityGroup = "sg1"
	updated, err := vms.Update(ctx, upd, gpupaas.UpdateOptions{})
	if err != nil || updated.Spec.SecurityGroup != "sg1" || !updated.Status.Running() {
		t.Fatalf("update: %v %#v", err, updated)
	}

	for _, step := range []struct{ verb, wantPower string }{{"stop", "stopped"}, {"start", "running"}, {"reboot", "running"}} {
		if _, err := vms.Action(ctx, "v1", step.verb, nil); err != nil {
			t.Fatalf("%s: %v", step.verb, err)
		}
		got, err := typed.WaitForVMInstance(ctx, vms, "v1", vmWait(typed.VMInstanceActionDone(step.verb), 2, false))
		if err != nil || got.Status.PowerState() != step.wantPower {
			t.Fatalf("%s: %v %#v", step.verb, err, got)
		}
	}
	if _, err := vms.Action(ctx, "v1", "start", nil); err == nil || !strings.Contains(err.Error(), "cannot start") {
		t.Fatalf("start while running: %v", err)
	}
	if _, err := vms.Action(ctx, "v1", "teleport", nil); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unknown action: %v", err)
	}

	if list, err := vms.List(ctx, gpupaas.ListOptions{}); err != nil || len(list.Items) != 1 {
		t.Fatalf("list: %v %#v", err, list)
	}
	if err := vms.Delete(ctx, "v1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if gone, err := typed.WaitForVMInstance(ctx, vms, "v1", vmWait(typed.VMInstanceGone, 1, true)); err != nil || gone != nil {
		t.Fatalf("wait gone: %v %v", gone, err)
	}
	if _, err := vms.Get(ctx, "v1", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("get after delete: %v", err)
	}
	if err := vms.Delete(ctx, "v1", gpupaas.DeleteOptions{IgnoreNotFound: true}); err != nil {
		t.Fatalf("delete of a missing VM with IgnoreNotFound: %v", err)
	}
	// every request went to the first-class collection, never to the legacy status route
	for _, r := range srv.Requests() {
		if strings.Contains(r, "/status") {
			t.Fatalf("unexpected route: %s", r)
		}
	}
}

func TestVMInstanceProjectScopeUsesProjectRoutes(t *testing.T) {
	srv, cs := newVMFixture(t, vmapi.Options{})
	ctx := waitCtx(t)
	if _, err := cs.V1alpha1().VMInstances(testProject).Create(ctx, newVM("p1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if srv.VM(testProject, "", "p1") == nil || srv.VM(testProject, "ws", "p1") != nil {
		t.Fatal("project-scope VM stored under the wrong scope")
	}
	found := false
	for _, r := range srv.Requests() {
		if r == "POST /apis/dev.envmgmt.io/v1/projects/"+testProject+"/virtualmachines" {
			found = true
		}
	}
	if !found {
		t.Fatalf("requests: %v", srv.Requests())
	}
}

func TestVMInstanceBackendValidationErrorsAreReturned(t *testing.T) {
	_, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().VMInstances(testProject)
	ctx := waitCtx(t)
	cases := []struct {
		name string
		edit func(*apiv1.VMInstance)
		want string
	}{
		{"no profile", func(v *apiv1.VMInstance) { v.Spec.Type = "nope" }, "not found"},
		{"no image", func(v *apiv1.VMInstance) { v.Spec.Image = "" }, "spec.image is required"},
		{"bad image", func(v *apiv1.VMInstance) { v.Spec.Image = "windows" }, "is not allowed by vm profile"},
		{"no auth", func(v *apiv1.VMInstance) { v.Spec.SSHKey = "" }, "spec.ssh_key or spec.guest_password"},
		{"unknown ssh key", func(v *apiv1.VMInstance) { v.Spec.SSHKey = "ghost" }, `ssh key "ghost" not found`},
		{"bad datacenter", func(v *apiv1.VMInstance) { v.Spec.Datacenter = "mars" }, "datacenter"},
		{"size not a preset", func(v *apiv1.VMInstance) { v.Spec.CPUCount = "4" }, "do not match a preset"},
		{"no size", func(v *apiv1.VMInstance) { v.Spec.CPUCount, v.Spec.Memory = "", "" }, "do not match a preset"},
		{"cpu and memory from different rows", func(v *apiv1.VMInstance) { v.Spec.CPUCount, v.Spec.Memory = "3", "12" }, "do not match a preset"},
		{"disk too small", func(v *apiv1.VMInstance) { v.Spec.BootDiskSize = 10 }, "boot_disk_size"},
		{"disk too large", func(v *apiv1.VMInstance) { v.Spec.BootDiskSize = 3000 }, "boot_disk_size"},
		{"disk off the step", func(v *apiv1.VMInstance) { v.Spec.BootDiskSize = 22 }, "boot_disk_size"},
	}
	for i, c := range cases {
		v := newVM("v" + string(rune('a'+i)))
		c.edit(v)
		_, err := vms.Create(ctx, v, gpupaas.CreateOptions{})
		var apiErr *gpupaas.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	// A password alone is enough, and is never returned; a valid boot disk and the second preset are accepted.
	v := newVM("pw")
	v.Spec.SSHKey, v.Spec.GuestPassword = "", "s3cret"
	v.Spec.CPUCount, v.Spec.Memory, v.Spec.BootDiskSize = "6", "12", 25
	got, err := vms.Create(ctx, v, gpupaas.CreateOptions{})
	if err != nil || got.Spec.GuestPassword != "" {
		t.Fatalf("password only: %v %#v", err, got.Spec)
	}
}

func TestVMInstancePublicIPFollowsTheSKUPolicy(t *testing.T) {
	none, always := vmSKU(), vmSKU()
	none.Metadata.Name, none.Spec.PublicIP = "vm-none", "None"
	always.Metadata.Name, always.Spec.PublicIP = "vm-always", "Always"
	_, cs := newVMFixture(t, vmapi.Options{}, vmSKU(), none, always)
	vms := cs.V1alpha1().VMInstances(testProject)
	ctx := waitCtx(t)

	v := newVM("n1")
	v.Spec.Type, v.Spec.AssignPublicIP = "vm-none", true
	if _, err := vms.Create(ctx, v, gpupaas.CreateOptions{}); err == nil || !strings.Contains(err.Error(), "does not allow a public IP") {
		t.Fatalf("policy None: %v", err)
	}
	v = newVM("a1")
	v.Spec.Type = "vm-always"
	got, err := vms.Create(ctx, v, gpupaas.CreateOptions{})
	if err != nil || !got.Spec.AssignPublicIP {
		t.Fatalf("policy Always pins the public IP on: %v %#v", err, got)
	}
	v = newVM("o1")
	v.Spec.AssignPublicIP = true
	if got, err = vms.Create(ctx, v, gpupaas.CreateOptions{}); err != nil || !got.Spec.AssignPublicIP {
		t.Fatalf("policy Optional honors the request: %v %#v", err, got)
	}
}

func TestVMInstanceTypeCannotChange(t *testing.T) {
	_, cs := newVMFixture(t, vmapi.Options{})
	vms := cs.V1alpha1().VMInstances(testProject)
	ctx := waitCtx(t)
	v, err := vms.Create(ctx, newVM("t1"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	v.Spec.Type = "other"
	if _, err := vms.Update(ctx, v, gpupaas.UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "cannot be changed") {
		t.Fatalf("type change: %v", err)
	}
}

func TestVMInstanceCapacityAndProvisionFailures(t *testing.T) {
	_, cs := newVMFixture(t, vmapi.Options{Capacity: 1})
	vms := cs.V1alpha1().VMInstances(testProject)
	ctx := waitCtx(t)
	if _, err := vms.Create(ctx, newVM("c1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForVMInstance(ctx, vms, "c1", vmWait(typed.VMInstanceRunning, 1, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := vms.Create(ctx, newVM("c2"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForVMInstance(ctx, vms, "c2", vmWait(typed.VMInstanceRunning, 1, false))
	var failed *typed.VMInstanceFailedError
	if !errors.As(err, &failed) || failed.Phase != "failed" || !strings.Contains(failed.Reason, "no capacity available") {
		t.Fatalf("capacity: %v", err)
	}

	srv2, cs2 := newVMFixture(t, vmapi.Options{})
	srv2.FailProvision = func(string) string { return "boom" }
	v2 := cs2.V1alpha1().VMInstances(testProject)
	if _, err := v2.Create(ctx, newVM("f1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err = typed.WaitForVMInstance(ctx, v2, "f1", vmWait(typed.VMInstanceRunning, 1, false))
	if !errors.As(err, &failed) || !strings.Contains(failed.Reason, "internal error") {
		t.Fatalf("provision failure: %v", err)
	}
}

func TestVMInstanceDeleteFailureThenRetry(t *testing.T) {
	srv, cs := newVMFixture(t, vmapi.Options{})
	var calls atomic.Int32
	srv.FailDelete = func(string) string {
		if calls.Add(1) == 1 {
			return "host unreachable"
		}
		return ""
	}
	vms := cs.V1alpha1().VMInstances(testProject)
	ctx := waitCtx(t)
	if _, err := vms.Create(ctx, newVM("d1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForVMInstance(ctx, vms, "d1", vmWait(typed.VMInstanceRunning, 1, false)); err != nil {
		t.Fatal(err)
	}
	if err := vms.Delete(ctx, "d1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForVMInstance(ctx, vms, "d1", vmWait(typed.VMInstanceGone, 1, true))
	var failed *typed.VMInstanceFailedError
	if !errors.As(err, &failed) || failed.Phase != "destroyfailed" || !strings.Contains(failed.Reason, "unreachable") {
		t.Fatalf("first delete: %v", err)
	}
	if err := vms.Delete(ctx, "d1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if gone, err := typed.WaitForVMInstance(ctx, vms, "d1", vmWait(typed.VMInstanceGone, 1, true)); err != nil || gone != nil {
		t.Fatalf("retry: %v %v", gone, err)
	}
}

func TestVMInstanceNotFoundIsDetectedForBothStatuses(t *testing.T) {
	for _, status := range []int{404, 502} {
		_, cs := newVMFixture(t, vmapi.Options{NotFoundStatus: status})
		_, err := cs.V1alpha1().VMInstances(testProject).Get(context.Background(), "missing", gpupaas.GetOptions{})
		if !gpupaas.IsNotFound(err) {
			t.Fatalf("status %d: %v", status, err)
		}
	}
}

func TestVMProfilesAndHiddenComputeInstance(t *testing.T) {
	srv, cs := newVMFixture(t, vmapi.Options{})
	srv.FailProvision = func(string) string { return "x" }
	srv.FailureCause = func(string) string { return "Invalid host configuration: host_ip is unbound." }
	ctx := waitCtx(t)
	p, err := cs.V1alpha1().VMProfiles(testProject).Get(ctx, "vm-sku", gpupaas.GetOptions{})
	if err != nil || p.Spec.PublicIP != "Optional" || len(p.Spec.Sizes()) != 2 || len(p.Status.GlobalSettings.Agents) != 1 {
		t.Fatalf("profile: %v %#v", err, p)
	}
	if l, err := cs.V1alpha1().VMProfiles(testProject).List(ctx, gpupaas.ListOptions{}); err != nil || len(l.Items) != 1 {
		t.Fatalf("profiles: %v %#v", err, l)
	}
	if _, err := cs.V1alpha1().VMProfiles(testProject).Get(ctx, "nope", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("missing profile: %v", err)
	}

	// The compute instance behind a failed VM carries the workflow error; the SDK asks for first-class
	// instances, so it is found by name.
	vms := cs.V1alpha1().Workspaces(testProject).VMInstances("ws")
	if _, err := vms.Create(ctx, newVM("h1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForVMInstance(ctx, vms, "h1", vmWait(typed.VMInstanceRunning, 1, false)); err == nil {
		t.Fatal("expected a failure")
	}
	ci, err := cs.V1alpha1().Workspaces(testProject).ComputeInstances("ws").Get(ctx, "h1", gpupaas.GetOptions{})
	if err != nil || ci.Status.Phase() != "failed" || !strings.Contains(strings.Join(workflowErrs(ci), " "), "host_ip is unbound") {
		t.Fatalf("hidden compute instance: %v %#v", err, ci)
	}
}

func TestVMInstanceOrganizationHeaderIsSent(t *testing.T) {
	srv := vmapi.New(vmapi.Options{Profiles: []apiv1.VMProfile{vmSKU()}})
	t.Cleanup(srv.Close)
	cfg := gpupaas.NewConfig(srv.URL, "token")
	cfg.OrganizationID = "org-1"
	cs, err := clientset.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.V1alpha1().VMInstances(testProject).List(context.Background(), gpupaas.ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if orgs := srv.Organizations(); len(orgs) != 1 || orgs[0] != "org-1" {
		t.Fatalf("organizations = %v", orgs)
	}
}
