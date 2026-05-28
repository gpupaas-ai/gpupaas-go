package apply_test

import (
	"context"
	"strings"
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/apply"
	"github.com/gpupaas-ai/gpupaas-go/client"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func TestApplyReaderMultiDoc(t *testing.T) {
	c := client.New(client.Options{UseMemory: true})
	ctx := context.Background()

	yaml := `apiVersion: gpupaas.ai/v1alpha1
kind: Project
metadata:
  name: demo
---
apiVersion: gpupaas.ai/v1alpha1
kind: Workspace
metadata:
  name: dev
  project: demo
`
	if err := apply.ApplyReader(ctx, c, strings.NewReader(yaml), "demo", ""); err != nil {
		t.Fatal(err)
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

func TestDeleteReaderIgnoreNotFound(t *testing.T) {
	c := client.New(client.Options{UseMemory: true})
	ctx := context.Background()

	yaml := `apiVersion: gpupaas.ai/v1alpha1
kind: Workspace
metadata:
  name: missing
  project: demo
`
	if err := apply.DeleteReader(ctx, c, strings.NewReader(yaml), "demo", "", true); err != nil {
		t.Fatal(err)
	}
}
