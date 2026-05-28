package client_test

import (
	"context"
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func TestGoClientBasicFlow(t *testing.T) {
	c := client.New(client.Options{UseMemory: true})
	ctx := context.Background()

	ws := &v1alpha1.Workspace{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
		Metadata: v1alpha1.ObjectMeta{Name: "example", Project: "demo"},
	}
	applied, err := c.Apply(ctx, ws, "demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if applied.GetName() != "example" {
		t.Fatalf("unexpected name %q", applied.GetName())
	}

	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindWorkspace}
	items, err := c.List(ctx, gvk, "demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(items))
	}
}
