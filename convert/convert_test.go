package convert_test

import (
	"testing"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/convert"
)

func TestFindAuthProjectIDByName(t *testing.T) {
	page := &convert.AuthProjectList{
		Results: []convert.AuthProject{
			{ID: "gkjnz20", Name: "test"},
			{ID: "dk331kn", Name: "batch-d-mon-3-4-4550s"},
		},
	}
	id, ok := convert.FindAuthProjectIDByName(page, "test")
	if !ok || id != "gkjnz20" {
		t.Fatalf("FindAuthProjectIDByName: got id=%q ok=%v", id, ok)
	}
	_, ok = convert.FindAuthProjectIDByName(page, "missing")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestFromAuthProjectStoresIDAnnotation(t *testing.T) {
	p := convert.FromAuthProject(&convert.AuthProject{
		ID:   "dk331kn",
		Name: "batch-d-mon-3-4-4550s",
	})
	if p.Metadata.Annotations["gpupaas.ai/project-id"] != "dk331kn" {
		t.Fatalf("annotation: %+v", p.Metadata.Annotations)
	}
}

func TestAuthProjectRoundTrip(t *testing.T) {
	auth := &convert.AuthProject{
		Name:        "demo",
		Description: "Demo project",
		Default:     true,
	}
	proj := convert.FromAuthProject(auth)
	if proj.Metadata.Name != "demo" {
		t.Fatalf("name: got %q", proj.Metadata.Name)
	}
	if proj.Spec.Description != "Demo project" {
		t.Fatalf("description: got %q", proj.Spec.Description)
	}
	if !proj.Spec.Default {
		t.Fatal("expected default=true")
	}
	if proj.Status.Phase != "Default" {
		t.Fatalf("phase: got %q", proj.Status.Phase)
	}

	back := convert.ToAuthProject(proj)
	if back.Name != "demo" || back.Description != "Demo project" || !back.Default {
		t.Fatalf("round trip: %+v", back)
	}
}

func TestAuthProjectList(t *testing.T) {
	next := "https://api.example/next"
	page := &convert.AuthProjectList{
		Count: 1,
		Next:  &next,
		Results: []convert.AuthProject{
			{Name: "a", Description: "first"},
		},
	}
	list := convert.FromAuthProjectList(page)
	if list.Metadata.Continue != next {
		t.Fatalf("continue: got %q", list.Metadata.Continue)
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "a" {
		t.Fatalf("items: %+v", list.Items)
	}
	if list.APIVersion != apiv1.APIVersion {
		t.Fatalf("apiVersion: %q", list.APIVersion)
	}
}

func TestPaaSWorkspaceRoundTrip(t *testing.T) {
	wire := &convert.PaaSWorkspace{
		APIVersion: convert.PaaSWorkspaceAPIVer,
		Kind:       convert.PaaSWorkspaceKind,
		Metadata: convert.PaaSMetadata{
			Name:        "ws1",
			Project:     "demo",
			DisplayName: "Workspace One",
			Description: "test workspace",
			Labels:      map[string]string{"env": "dev"},
		},
		Spec: convert.PaaSWorkspaceSpec{IconURL: "https://icon", Readme: "readme"},
		Status: convert.PaaSWorkspaceStatus{
			CommonStatus: convert.PaaSStatus{ConditionStatus: "StatusOK"},
		},
	}
	ws := convert.FromPaaSWorkspace(wire)
	if ws.Metadata.Project != "demo" || ws.Metadata.Name != "ws1" {
		t.Fatalf("metadata: %+v", ws.Metadata)
	}
	if ws.Spec.Description != "test workspace" || ws.Spec.IconURL != "https://icon" {
		t.Fatalf("spec: %+v", ws.Spec)
	}
	if ws.Status.Phase != "StatusOK" {
		t.Fatalf("phase: %q", ws.Status.Phase)
	}

	back := convert.ToPaaSWorkspace(ws, "demo")
	if back.Metadata.Project != "demo" || back.Metadata.Name != "ws1" {
		t.Fatalf("wire metadata: %+v", back.Metadata)
	}
	if back.APIVersion != convert.PaaSWorkspaceAPIVer {
		t.Fatalf("apiVersion: %q", back.APIVersion)
	}
}

func TestPaaSWorkspaceList(t *testing.T) {
	list := convert.FromPaaSWorkspaceList(&convert.PaaSWorkspaceList{
		APIVersion: convert.PaaSWorkspaceAPIVer,
		Kind:       convert.PaaSWorkspaceListKind,
		Items: []convert.PaaSWorkspace{
			{
				Metadata: convert.PaaSMetadata{Name: "ws1", Project: "demo"},
			},
		},
	})
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != "ws1" {
		t.Fatalf("items: %+v", list.Items)
	}
}
