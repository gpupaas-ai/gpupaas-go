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
	"github.com/gpupaas-ai/gpupaas-go/mock/bmapi"
)

func bmSKU() apiv1.BmProfile {
	return apiv1.BmProfile{
		Metadata: apiv1.ObjectMeta{Name: "bm-sku"},
		Spec: apiv1.BmProfileSpec{
			ProvisionerType: "bcm",
			Image:           []string{"ubuntu-24", "ubuntu-26"},
			ImageDefault:    "ubuntu-24",
			Datacenter:      []string{"dc1"},
			AssignPublicIP:  "Disabled",
			MRCContractTerm: []int32{0, 12},
			GPUName:         "custom",
			InventoryConfig: []apiv1.BmInventoryTag{{Key: "server_type", Value: "l40s"}},
		},
		Status: apiv1.BmProfileStatus{GlobalSettings: &apiv1.BmProfileGlobalSettings{Agents: []string{"agent-a"}}},
	}
}

func newBMFixture(t *testing.T, opts bmapi.Options) (*bmapi.Server, clientset.Interface) {
	t.Helper()
	opts.Profiles = []apiv1.BmProfile{bmSKU()}
	if opts.SSHKeys == nil {
		opts.SSHKeys = []string{"key1"}
	}
	srv := bmapi.New(opts)
	t.Cleanup(srv.Close)
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	return srv, cs
}

func newBM(name string) *apiv1.BareMetal {
	return &apiv1.BareMetal{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindBareMetal},
		Metadata: apiv1.ObjectMeta{Name: name},
		Spec:     apiv1.BareMetalSpec{Type: "bm-sku", Image: "ubuntu-24", SSHKey: "key1"},
	}
}

func bmWait(done func(*apiv1.BareMetal) (bool, error), stable int, gone bool) typed.BareMetalWaitOptions {
	return typed.BareMetalWaitOptions{Interval: 5 * time.Millisecond, Done: done, Stable: stable, GoneIsDone: gone}
}

func TestBareMetalFullLifecycleWorkspaceScope(t *testing.T) {
	srv, cs := newBMFixture(t, bmapi.Options{})
	bms := cs.V1alpha1().Workspaces(testProject).BareMetals("ws")
	ctx := waitCtx(t)

	created, err := bms.Create(ctx, newBM("b1"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status.Phase() != "in_progress" || created.Metadata.Workspace != "ws" || created.Metadata.Project != testProject {
		t.Fatalf("created: phase=%q meta=%#v", created.Status.Phase(), created.Metadata)
	}
	// The platform fills in what the SKU decides.
	sp := created.Spec
	if sp.Datacenter != "dc1" || sp.ProvisionerType != "bcm" || sp.GPUName != "custom" || sp.ComputeInstanceID == "" || sp.Sharing == nil || sp.Sharing.ShareMode != "None" {
		t.Fatalf("derived fields: %#v", sp)
	}

	// Create never takes over an existing name.
	_, err = bms.Create(ctx, newBM("b1"), gpupaas.CreateOptions{})
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("second create: %v", err)
	}

	running, err := typed.WaitForBareMetal(ctx, bms, "b1", bmWait(typed.BareMetalRunning, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	if running.Status.Output == nil || running.Status.Output.HostName == "" || running.Status.Output.UserName != "ubuntu" || running.Status.PowerState() != "running" {
		t.Fatalf("running: %#v", running.Status)
	}

	// In-place update (a security group): the machine stays up.
	upd := running.DeepCopyObject().(*apiv1.BareMetal)
	upd.Spec.SecurityGroup = "sg1"
	updated, err := bms.Update(ctx, upd, gpupaas.UpdateOptions{})
	if err != nil || updated.Spec.SecurityGroup != "sg1" || !updated.Status.Running() {
		t.Fatalf("update: %v %#v", err, updated)
	}

	// Power actions.
	for _, step := range []struct{ verb, wantPower string }{{"stop", "stopped"}, {"start", "running"}, {"reboot", "running"}} {
		if _, err := bms.Action(ctx, "b1", step.verb, nil); err != nil {
			t.Fatalf("%s: %v", step.verb, err)
		}
		got, err := typed.WaitForBareMetal(ctx, bms, "b1", bmWait(typed.BareMetalActionDone(step.verb), 2, false))
		if err != nil || got.Status.PowerState() != step.wantPower {
			t.Fatalf("%s: %v %#v", step.verb, err, got)
		}
	}
	// "restart" is the same action as "reboot".
	if _, err := bms.Action(ctx, "b1", "restart", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForBareMetal(ctx, bms, "b1", bmWait(typed.BareMetalActionDone("reboot"), 2, false)); err != nil {
		t.Fatal(err)
	}
	// An action in the wrong state is rejected by the platform.
	if _, err := bms.Action(ctx, "b1", "start", nil); err == nil || !strings.Contains(err.Error(), "cannot start") {
		t.Fatalf("start while running: %v", err)
	}
	// "Reinstall OS" has a space in its name and takes a body.
	if _, err := bms.Action(ctx, "b1", "Reinstall OS", map[string]any{"image": "ubuntu-26"}); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if _, err := typed.WaitForBareMetal(ctx, bms, "b1", bmWait(typed.BareMetalActionDone("Reinstall OS"), 2, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := bms.Action(ctx, "b1", "Reinstall OS", nil); err == nil || !strings.Contains(err.Error(), "requires an image") {
		t.Fatalf("reinstall without image: %v", err)
	}

	list, err := bms.List(ctx, gpupaas.ListOptions{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list: %v %#v", err, list)
	}

	if err := bms.Delete(ctx, "b1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if gone, err := typed.WaitForBareMetal(ctx, bms, "b1", bmWait(typed.BareMetalGone, 1, true)); err != nil || gone != nil {
		t.Fatalf("wait gone: %v %v", gone, err)
	}
	if _, err := bms.Get(ctx, "b1", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("get after delete: %v", err)
	}
	if err := bms.Delete(ctx, "b1", gpupaas.DeleteOptions{IgnoreNotFound: true}); err != nil {
		t.Fatalf("delete of a missing instance with IgnoreNotFound: %v", err)
	}
	for _, r := range srv.Requests() {
		if strings.Contains(r, "baremetalmachines") {
			t.Fatalf("the admin inventory route was used: %s", r)
		}
	}
}

func TestBareMetalProjectScopeUsesProjectRoutes(t *testing.T) {
	srv, cs := newBMFixture(t, bmapi.Options{})
	ctx := waitCtx(t)
	if _, err := cs.V1alpha1().BareMetals(testProject).Create(ctx, newBM("p1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if srv.BareMetal(testProject, "", "p1") == nil || srv.BareMetal(testProject, "ws", "p1") != nil {
		t.Fatal("project-scope instance stored under the wrong scope")
	}
	found := false
	for _, r := range srv.Requests() {
		if r == "POST /apis/dev.envmgmt.io/v1/projects/"+testProject+"/baremetals" {
			found = true
		}
	}
	if !found {
		t.Fatalf("requests: %v", srv.Requests())
	}
}

func TestBareMetalBackendValidationErrorsAreReturned(t *testing.T) {
	_, cs := newBMFixture(t, bmapi.Options{})
	bms := cs.V1alpha1().BareMetals(testProject)
	ctx := waitCtx(t)
	cases := []struct {
		name string
		edit func(*apiv1.BareMetal)
		want string
	}{
		{"no profile", func(b *apiv1.BareMetal) { b.Spec.Type = "nope" }, "not found"},
		{"no image", func(b *apiv1.BareMetal) { b.Spec.Image = "" }, "spec.image is required"},
		{"bad image", func(b *apiv1.BareMetal) { b.Spec.Image = "windows" }, "is not allowed by bm profile"},
		{"no auth", func(b *apiv1.BareMetal) { b.Spec.SSHKey = "" }, "sshKey or spec.userPassword"},
		{"unknown ssh key", func(b *apiv1.BareMetal) { b.Spec.SSHKey = "ghost" }, `ssh key "ghost" not found`},
		{"bad datacenter", func(b *apiv1.BareMetal) { b.Spec.Datacenter = "mars" }, "datacenter"},
		{"public ip not allowed", func(b *apiv1.BareMetal) { b.Spec.AssignPublicIP = true }, "does not allow a public IP"},
		{"bad contract term", func(b *apiv1.BareMetal) { b.Spec.ContractTerm = 7 }, "contract term"},
	}
	for i, c := range cases {
		b := newBM("v" + string(rune('a'+i)))
		c.edit(b)
		_, err := bms.Create(ctx, b, gpupaas.CreateOptions{})
		var apiErr *gpupaas.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	// A password alone is enough, and is never returned.
	b := newBM("pw")
	b.Spec.SSHKey, b.Spec.UserPassword = "", "s3cret"
	got, err := bms.Create(ctx, b, gpupaas.CreateOptions{})
	if err != nil || got.Spec.UserPassword != "" {
		t.Fatalf("password only: %v %#v", err, got.Spec)
	}
}

func TestBareMetalTypeCannotChange(t *testing.T) {
	_, cs := newBMFixture(t, bmapi.Options{})
	bms := cs.V1alpha1().BareMetals(testProject)
	ctx := waitCtx(t)
	b, err := bms.Create(ctx, newBM("t1"), gpupaas.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b.Spec.Type = "other"
	if _, err := bms.Update(ctx, b, gpupaas.UpdateOptions{}); err == nil || !strings.Contains(err.Error(), "cannot be changed") {
		t.Fatalf("type change: %v", err)
	}
}

func TestBareMetalCapacityAndProvisionFailures(t *testing.T) {
	_, cs := newBMFixture(t, bmapi.Options{Capacity: 1})
	bms := cs.V1alpha1().BareMetals(testProject)
	ctx := waitCtx(t)
	if _, err := bms.Create(ctx, newBM("c1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForBareMetal(ctx, bms, "c1", bmWait(typed.BareMetalRunning, 1, false)); err != nil {
		t.Fatal(err)
	}
	// The second one gets no machine, like the live "no capacity available right now".
	if _, err := bms.Create(ctx, newBM("c2"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForBareMetal(ctx, bms, "c2", bmWait(typed.BareMetalRunning, 1, false))
	var failed *typed.BareMetalFailedError
	if !errors.As(err, &failed) || failed.Phase != "failed" || !strings.Contains(failed.Reason, "no capacity available") {
		t.Fatalf("capacity: %v", err)
	}

	srv2, cs2 := newBMFixture(t, bmapi.Options{})
	srv2.FailProvision = func(string) string { return "boom" }
	b2 := cs2.V1alpha1().BareMetals(testProject)
	if _, err := b2.Create(ctx, newBM("f1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err = typed.WaitForBareMetal(ctx, b2, "f1", bmWait(typed.BareMetalRunning, 1, false))
	if !errors.As(err, &failed) || !strings.Contains(failed.Reason, "internal error") {
		t.Fatalf("provision failure: %v", err)
	}
}

func TestBareMetalDeleteFailureThenRetry(t *testing.T) {
	srv, cs := newBMFixture(t, bmapi.Options{})
	var calls atomic.Int32
	srv.FailDelete = func(string) string {
		if calls.Add(1) == 1 {
			return "provisioner unreachable"
		}
		return ""
	}
	bms := cs.V1alpha1().BareMetals(testProject)
	ctx := waitCtx(t)
	if _, err := bms.Create(ctx, newBM("d1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForBareMetal(ctx, bms, "d1", bmWait(typed.BareMetalRunning, 1, false)); err != nil {
		t.Fatal(err)
	}
	if err := bms.Delete(ctx, "d1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := typed.WaitForBareMetal(ctx, bms, "d1", bmWait(typed.BareMetalGone, 1, true))
	var failed *typed.BareMetalFailedError
	if !errors.As(err, &failed) || failed.Phase != "destroyfailed" || !strings.Contains(failed.Reason, "unreachable") {
		t.Fatalf("first delete: %v", err)
	}
	if err := bms.Delete(ctx, "d1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if gone, err := typed.WaitForBareMetal(ctx, bms, "d1", bmWait(typed.BareMetalGone, 1, true)); err != nil || gone != nil {
		t.Fatalf("retry: %v %v", gone, err)
	}
}

func TestBareMetalNotFoundIsDetectedForBothStatuses(t *testing.T) {
	for _, status := range []int{404, 502} {
		_, cs := newBMFixture(t, bmapi.Options{NotFoundStatus: status})
		_, err := cs.V1alpha1().BareMetals(testProject).Get(context.Background(), "missing", gpupaas.GetOptions{})
		if !gpupaas.IsNotFound(err) {
			t.Fatalf("status %d: %v", status, err)
		}
	}
}

func TestBmProfilesAndHiddenComputeInstance(t *testing.T) {
	srv, cs := newBMFixture(t, bmapi.Options{})
	srv.FailProvision = func(string) string { return "x" }
	srv.FailureCause = func(string) string {
		return "Failed to execute Node Provisioning handler: Bond configuration failed (exit=124)."
	}
	ctx := waitCtx(t)
	p, err := cs.V1alpha1().BmProfiles(testProject).Get(ctx, "bm-sku", gpupaas.GetOptions{})
	if err != nil || p.Spec.ProvisionerType != "bcm" || p.Spec.AssignPublicIP != "Disabled" || len(p.Status.GlobalSettings.Agents) != 1 {
		t.Fatalf("profile: %v %#v", err, p)
	}
	if l, err := cs.V1alpha1().BmProfiles(testProject).List(ctx, gpupaas.ListOptions{}); err != nil || len(l.Items) != 1 {
		t.Fatalf("profiles: %v %#v", err, l)
	}
	if _, err := cs.V1alpha1().BmProfiles(testProject).Get(ctx, "nope", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("missing profile: %v", err)
	}

	// The compute instance behind a failed instance carries the workflow error; the SDK asks for
	// first-class instances, so it is found by name.
	bms := cs.V1alpha1().Workspaces(testProject).BareMetals("ws")
	if _, err := bms.Create(ctx, newBM("h1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := typed.WaitForBareMetal(ctx, bms, "h1", bmWait(typed.BareMetalRunning, 1, false)); err == nil {
		t.Fatal("expected a failure")
	}
	ci, err := cs.V1alpha1().Workspaces(testProject).ComputeInstances("ws").Get(ctx, "h1", gpupaas.GetOptions{})
	if err != nil || ci.Status.Phase() != "failed" || !strings.Contains(strings.Join(workflowErrs(ci), " "), "Bond configuration failed") {
		t.Fatalf("hidden compute instance: %v %#v", err, ci)
	}
}

func workflowErrs(ci *apiv1.ComputeInstance) []string {
	var out []string
	resources, _ := ci.Status.WorkflowStatus["resources"].([]any)
	for _, r := range resources {
		rm, _ := r.(map[string]any)
		errs, _ := rm["errors"].([]any)
		for _, e := range errs {
			if em, ok := e.(map[string]any); ok {
				if raw, ok := em["raw"].(string); ok {
					out = append(out, raw)
				}
			}
		}
	}
	return out
}

func TestBareMetalOrganizationHeaderIsSent(t *testing.T) {
	srv := bmapi.New(bmapi.Options{Profiles: []apiv1.BmProfile{bmSKU()}})
	t.Cleanup(srv.Close)
	cfg := gpupaas.NewConfig(srv.URL, "token")
	cfg.OrganizationID = "org-1"
	cs, err := clientset.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.V1alpha1().BareMetals(testProject).List(context.Background(), gpupaas.ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if orgs := srv.Organizations(); len(orgs) != 1 || orgs[0] != "org-1" {
		t.Fatalf("organizations = %v", orgs)
	}
}
