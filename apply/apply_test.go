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
kind: MKSCluster
metadata:
  name: c1
  project: demo
---
apiVersion: gpupaas.ai/v1alpha1
kind: MKSCluster
metadata:
  name: c2
  project: demo
  workspace: dev
`
	if err := apply.ApplyReader(ctx, c, strings.NewReader(yaml), "demo", ""); err != nil {
		t.Fatal(err)
	}

	gvk := runtime.GroupVersionKind{Group: v1alpha1.Group, Version: v1alpha1.Version, Kind: v1alpha1.KindMKSCluster}
	items, err := c.List(ctx, gvk, "demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 clusters in the project, got %d", len(items))
	}
	items, err = c.List(ctx, gvk, "demo", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GetName() != "c2" {
		t.Fatalf("expected the workspace cluster c2, got %v", items)
	}
}

func TestApplyReaderRejectsUnknownKind(t *testing.T) {
	c := client.New(client.Options{UseMemory: true})
	yaml := `apiVersion: gpupaas.ai/v1alpha1
kind: Workspace
metadata:
  name: dev
  project: demo
`
	if err := apply.ApplyReader(context.Background(), c, strings.NewReader(yaml), "demo", ""); err == nil {
		t.Fatal("a kind the SDK does not manage must be rejected")
	}
}

func TestDeleteReaderIgnoreNotFound(t *testing.T) {
	c := client.New(client.Options{UseMemory: true})
	ctx := context.Background()

	yaml := `apiVersion: gpupaas.ai/v1alpha1
kind: MKSCluster
metadata:
  name: missing
  project: demo
`
	if err := apply.DeleteReader(ctx, c, strings.NewReader(yaml), "demo", "", true); err != nil {
		t.Fatal(err)
	}
	if err := apply.DeleteReader(ctx, c, strings.NewReader(yaml), "demo", "", false); err == nil {
		t.Fatal("deleting a missing cluster must fail unless not found is ignored")
	}
}
