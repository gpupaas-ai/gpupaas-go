package memory_test

import (
	"context"
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/backend/memory"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func newCluster(name, project, workspace, version string) *v1alpha1.MKSCluster {
	return &v1alpha1.MKSCluster{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindMKSCluster},
		Metadata: v1alpha1.ObjectMeta{Name: name, Project: project, Workspace: workspace},
		Spec:     v1alpha1.MKSClusterSpec{KubernetesVersion: version},
	}
}

func TestApplyCreateUpdateListDelete(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindMKSCluster}

	cluster := newCluster("c1", "demo", "", "1.35")
	if _, err := store.Apply(ctx, cluster); err != nil {
		t.Fatal(err)
	}

	cluster.Spec.KubernetesVersion = "1.36"
	if _, err := store.Apply(ctx, cluster); err != nil {
		t.Fatal(err)
	}

	items, err := store.List(ctx, gvk, backend.ListOptions{Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	got, err := store.Get(ctx, gvk, "demo", "", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.(*v1alpha1.MKSCluster).Spec.KubernetesVersion != "1.36" {
		t.Fatalf("expected updated spec")
	}

	if err := store.Delete(ctx, gvk, "demo", "", "c1"); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(ctx, gvk, backend.ListOptions{Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items after delete")
	}
}

func TestScopeSeparatesProjectAndWorkspaceObjects(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindMKSCluster}

	for _, c := range []*v1alpha1.MKSCluster{
		newCluster("same", "demo", "", "1.35"),
		newCluster("same", "demo", "dev", "1.36"),
		newCluster("other", "elsewhere", "", "1.36"),
	} {
		if _, err := store.Apply(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	all, err := store.List(ctx, gvk, backend.ListOptions{Project: "demo"})
	if err != nil || len(all) != 2 {
		t.Fatalf("project list = %d, %v; want 2", len(all), err)
	}
	inWorkspace, err := store.List(ctx, gvk, backend.ListOptions{Project: "demo", Workspace: "dev"})
	if err != nil || len(inWorkspace) != 1 || inWorkspace[0].GetWorkspace() != "dev" {
		t.Fatalf("workspace list = %v, %v", inWorkspace, err)
	}
	if _, err := store.Get(ctx, gvk, "demo", "", "other"); err == nil {
		t.Fatal("an object of another project must not be found")
	}
}
