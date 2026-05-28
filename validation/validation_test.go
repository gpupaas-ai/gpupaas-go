package validation_test

import (
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/validation"
)

func TestValidateWorkspaceDefaultProject(t *testing.T) {
	ws := &v1alpha1.Workspace{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
		Metadata: v1alpha1.ObjectMeta{Name: "dev"},
	}
	if err := validation.ValidateObject(ws, validation.Options{DefaultProject: "demo"}); err != nil {
		t.Fatal(err)
	}
	if ws.Metadata.Project != "demo" {
		t.Fatalf("expected project demo, got %q", ws.Metadata.Project)
	}
}

func TestValidateWorkspaceMissingProject(t *testing.T) {
	ws := &v1alpha1.Workspace{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindWorkspace},
		Metadata: v1alpha1.ObjectMeta{Name: "dev"},
	}
	if err := validation.ValidateObject(ws, validation.Options{}); err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestValidateProjectNoProjectRequired(t *testing.T) {
	p := &v1alpha1.Project{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindProject},
		Metadata: v1alpha1.ObjectMeta{Name: "demo"},
	}
	if err := validation.ValidateObject(p, validation.Options{}); err != nil {
		t.Fatal(err)
	}
}
