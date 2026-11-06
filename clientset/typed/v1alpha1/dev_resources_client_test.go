package v1alpha1_test

import (
	"testing"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	"github.com/gpupaas-ai/gpupaas-go/mock/devapi"
)

func newDevFixture(t *testing.T) (*devapi.Server, clientset.Interface) {
	t.Helper()
	srv := devapi.New()
	t.Cleanup(srv.Close)
	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	return srv, cs
}

func TestSshKeyLifecycleAtProjectAndWorkspaceScope(t *testing.T) {
	srv, cs := newDevFixture(t)
	ctx := waitCtx(t)
	key := func(name string) *apiv1.SshKey {
		return &apiv1.SshKey{Metadata: apiv1.ObjectMeta{Name: name}, Spec: apiv1.SshKeySpec{PublicKey: "ssh-ed25519 AAAA test"}}
	}

	proj := cs.V1alpha1().SshKeys(testProject)
	got, err := proj.Create(ctx, key("k1"), gpupaas.CreateOptions{})
	if err != nil || got.Metadata.Name != "k1" || got.Spec.PublicKey != "ssh-ed25519 AAAA test" || got.Status.Status != "success" {
		t.Fatalf("create: %#v %v", got, err)
	}
	if srv.SshKey(testProject, "", "k1") == nil {
		t.Fatal("the key must be stored at project scope")
	}

	ws := cs.V1alpha1().Workspaces(testProject).SshKeys("team-a")
	if _, err := ws.Create(ctx, key("k1"), gpupaas.CreateOptions{}); err != nil {
		t.Fatalf("the same name is free in another scope: %v", err)
	}
	if got, err := ws.Get(ctx, "k1", gpupaas.GetOptions{}); err != nil || got.Metadata.Workspace != "team-a" {
		t.Fatalf("workspace get: %#v %v", got, err)
	}

	if _, err := proj.Create(ctx, key("k1"), gpupaas.CreateOptions{}); err == nil || !gpupaas.IsConflict(err) {
		t.Fatalf("a taken name must fail with 409, got %v", err)
	}

	// a project-level list shows the keys of its workspaces too, as the live API does
	list, err := proj.List(ctx, gpupaas.ListOptions{})
	if err != nil || len(list.Items) != 2 {
		t.Fatalf("project list must hold the project-scope key and the workspace one: %#v %v", list, err)
	}
	if wsList, err := ws.List(ctx, gpupaas.ListOptions{}); err != nil || len(wsList.Items) != 1 || wsList.Items[0].Metadata.Workspace != "team-a" {
		t.Fatalf("workspace list must hold only its own key: %#v %v", wsList, err)
	}

	if err := ws.Delete(ctx, "k1", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Get(ctx, "k1", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("a deleted key must be not found, got %v", err)
	}
	if err := ws.Delete(ctx, "k1", gpupaas.DeleteOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("deleting a missing key must be not found, got %v", err)
	}
	if err := ws.Delete(ctx, "k1", gpupaas.DeleteOptions{IgnoreNotFound: true}); err != nil {
		t.Fatalf("IgnoreNotFound: %v", err)
	}
}

func TestSecurityGroupLifecycle(t *testing.T) {
	srv, cs := newDevFixture(t)
	ctx := waitCtx(t)
	sgs := cs.V1alpha1().Workspaces(testProject).SecurityGroups("team-a")
	g := &apiv1.SecurityGroup{
		Metadata: apiv1.ObjectMeta{Name: "web"},
		Spec: apiv1.SecurityGroupSpec{Rules: []apiv1.Rule{
			{SourceCIDR: "0.0.0.0/0", Application: "ssh", Protocol: "tcp", Action: "allow"},
			{SourceCIDR: "10.0.0.0/8", Application: "custom", ApplicationPort: "8080", Protocol: "tcp", Action: "allow"},
		}},
	}
	got, err := sgs.Create(ctx, g, gpupaas.CreateOptions{})
	if err != nil || len(got.Spec.Rules) != 2 || got.Spec.Rules[1].ApplicationPort != "8080" || got.Metadata.Workspace != "team-a" {
		t.Fatalf("create: %#v %v", got, err)
	}
	stored := srv.SecurityGroup(testProject, "team-a", "web")
	if stored == nil || len(stored.Spec.Rules) != 2 || stored.Spec.Rules[1].ApplicationPort != "8080" {
		t.Fatalf("stored on the wire: %#v", stored)
	}
	if _, err := sgs.Create(ctx, g, gpupaas.CreateOptions{}); !gpupaas.IsConflict(err) {
		t.Fatalf("a taken name must fail with 409, got %v", err)
	}
	bad := &apiv1.SecurityGroup{Metadata: apiv1.ObjectMeta{Name: "bad"}, Spec: apiv1.SecurityGroupSpec{Rules: []apiv1.Rule{{Application: "ssh", Action: "maybe"}}}}
	if _, err := sgs.Create(ctx, bad, gpupaas.CreateOptions{}); err == nil {
		t.Fatal("an invalid rule action must be rejected by the platform")
	}
	if list, err := sgs.List(ctx, gpupaas.ListOptions{}); err != nil || len(list.Items) != 1 {
		t.Fatalf("list: %#v %v", list, err)
	}
	if err := sgs.Delete(ctx, "web", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := sgs.Get(ctx, "web", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("deleted group must be not found, got %v", err)
	}
}

func TestWorkspaceLifecycleAndOrdering(t *testing.T) {
	srv, cs := newDevFixture(t)
	ctx := waitCtx(t)
	wss := cs.V1alpha1().Workspaces(testProject)
	in := &apiv1.Workspace{
		Metadata: apiv1.ObjectMeta{Name: "team-a", DisplayName: "Team A", Description: "d", Labels: map[string]string{"team": "a"}},
	}
	got, err := wss.Create(ctx, in, gpupaas.CreateOptions{})
	if err != nil || got.Metadata.DisplayName != "Team A" || got.Metadata.Project != testProject || got.Metadata.Description != "d" {
		t.Fatalf("create: %#v %v", got, err)
	}
	if _, err := wss.Create(ctx, in, gpupaas.CreateOptions{}); !gpupaas.IsConflict(err) {
		t.Fatalf("a taken name must fail with 409, got %v", err)
	}
	in.Metadata.Description = "changed"
	if got, err := wss.Update(ctx, in, gpupaas.UpdateOptions{}); err != nil || got.Metadata.Description != "changed" {
		t.Fatalf("update: %#v %v", got, err)
	}
	if srv.Workspace(testProject, "team-a").Metadata.Description != "changed" {
		t.Fatal("update must reach the platform")
	}
	if list, err := wss.List(ctx, gpupaas.ListOptions{}); err != nil || len(list.Items) != 1 {
		t.Fatalf("list: %#v %v", list, err)
	}

	// a workspace that still holds a key cannot be deleted
	if _, err := wss.SshKeys("team-a").Create(ctx, &apiv1.SshKey{Metadata: apiv1.ObjectMeta{Name: "k"}, Spec: apiv1.SshKeySpec{PublicKey: "ssh-ed25519 A"}}, gpupaas.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := wss.Delete(ctx, "team-a", gpupaas.DeleteOptions{}); err == nil {
		t.Fatal("a workspace holding a key must not be deletable")
	}
	if err := wss.SshKeys("team-a").Delete(ctx, "k", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := wss.Delete(ctx, "team-a", gpupaas.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := wss.Get(ctx, "team-a", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("deleted workspace must be not found, got %v", err)
	}
}
