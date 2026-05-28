package memory_test

import (
	"context"
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/backend/memory"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func TestApplyCreateUpdateListDelete(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindWorkspace}

	ws := &v1alpha1.Workspace{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
		Metadata: v1alpha1.ObjectMeta{Name: "dev", Project: "demo"},
		Spec:     v1alpha1.WorkspaceSpec{Description: "v1"},
	}
	if _, err := store.Apply(ctx, ws); err != nil {
		t.Fatal(err)
	}

	ws.Spec.Description = "v2"
	if _, err := store.Apply(ctx, ws); err != nil {
		t.Fatal(err)
	}

	items, err := store.List(ctx, gvk, backend.ListOptions{Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	got, err := store.Get(ctx, gvk, "demo", "", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if got.(*v1alpha1.Workspace).Spec.Description != "v2" {
		t.Fatalf("expected updated spec")
	}

	if err := store.Delete(ctx, gvk, "demo", "", "dev"); err != nil {
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
