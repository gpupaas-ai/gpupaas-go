package validation_test

import (
	"testing"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/validation"
)

func newCluster() *v1alpha1.MKSCluster {
	return &v1alpha1.MKSCluster{
		TypeMeta: v1alpha1.TypeMeta{APIVersion: v1alpha1.APIVersion, Kind: v1alpha1.KindMKSCluster},
		Metadata: v1alpha1.ObjectMeta{Name: "c1"},
	}
}

func TestValidateMKSClusterDefaultProject(t *testing.T) {
	c := newCluster()
	if err := validation.ValidateObject(c, validation.Options{DefaultProject: "demo"}); err != nil {
		t.Fatalf("a project-scoped cluster is valid: %v", err)
	}
	if c.Metadata.Project != "demo" || c.Metadata.Workspace != "" {
		t.Fatalf("scope = %+v", c.Metadata)
	}
}

func TestValidateMKSClusterDefaultWorkspace(t *testing.T) {
	c := newCluster()
	if err := validation.ValidateObject(c, validation.Options{DefaultProject: "demo", DefaultWorkspace: "dev"}); err != nil {
		t.Fatal(err)
	}
	if c.Metadata.Project != "demo" || c.Metadata.Workspace != "dev" {
		t.Fatalf("scope = %+v", c.Metadata)
	}
	own := newCluster()
	own.Metadata.Workspace = "mine"
	if err := validation.ValidateObject(own, validation.Options{DefaultProject: "demo", DefaultWorkspace: "dev"}); err != nil {
		t.Fatal(err)
	}
	if own.Metadata.Workspace != "mine" {
		t.Fatalf("an explicit workspace must win over the default, got %q", own.Metadata.Workspace)
	}
}

func TestValidateMKSClusterMissingProject(t *testing.T) {
	if err := validation.ValidateObject(newCluster(), validation.Options{}); err == nil {
		t.Fatal("project is required")
	}
}

func TestValidateRequiresIdentity(t *testing.T) {
	for name, mutate := range map[string]func(*v1alpha1.MKSCluster){
		"apiVersion": func(c *v1alpha1.MKSCluster) { c.APIVersion = "" },
		"kind":       func(c *v1alpha1.MKSCluster) { c.Kind = "" },
		"name":       func(c *v1alpha1.MKSCluster) { c.Metadata.Name = "" },
	} {
		c := newCluster()
		mutate(c)
		if err := validation.ValidateObject(c, validation.Options{DefaultProject: "demo"}); err == nil {
			t.Errorf("a cluster without %s must be rejected", name)
		}
	}
}
