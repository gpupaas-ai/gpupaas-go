package runtime_test

import (
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

func TestSerializerDecodeJSON(t *testing.T) {
	scheme := v1alpha1.DefaultScheme()
	ser := runtime.NewSerializer(scheme)

	data := []byte(`{
		"apiVersion": "gpupaas.ai/v1alpha1",
		"kind": "Workspace",
		"metadata": {"name": "dev", "project": "demo"},
		"spec": {"description": "test"}
	}`)

	obj, err := ser.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if obj.GetName() != "dev" || obj.GetProject() != "demo" {
		t.Fatalf("unexpected object: %+v", obj)
	}
}

func TestSerializerDecodeMultiDocYAML(t *testing.T) {
	scheme := v1alpha1.DefaultScheme()
	ser := runtime.NewSerializer(scheme)

	data := []byte(`apiVersion: gpupaas.ai/v1alpha1
kind: Project
metadata:
  name: demo
spec:
  displayName: Demo
---
apiVersion: gpupaas.ai/v1alpha1
kind: Workspace
metadata:
  name: dev
  project: demo
`)

	objects, err := ser.DecodeAll(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(objects))
	}
}

func TestSerializerUnknownKind(t *testing.T) {
	scheme := v1alpha1.DefaultScheme()
	ser := runtime.NewSerializer(scheme)

	_, err := ser.Decode([]byte(`{"apiVersion":"gpupaas.ai/v1alpha1","kind":"Unknown","metadata":{"name":"x"}}`))
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
