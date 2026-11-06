package validation

import (
	"fmt"

	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// Options supplies default scope values for validation.
type Options struct {
	DefaultProject   string
	DefaultWorkspace string
}

// ValidateObject checks required fields and applies scope defaults in place.
func ValidateObject(obj runtime.Object, opts Options) error {
	if obj.GetAPIVersion() == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if obj.GetKind() == "" {
		return fmt.Errorf("kind is required")
	}
	if obj.GetName() == "" {
		return fmt.Errorf("metadata.name is required")
	}

	switch obj.GetKind() {
	case v1alpha1.KindMKSCluster:
		// Clusters live in a project and, optionally, in one of its workspaces.
		project := obj.GetProject()
		if project == "" {
			project = opts.DefaultProject
		}
		if project == "" {
			return fmt.Errorf("metadata.project is required (or pass project to Apply/List)")
		}
		obj.SetProject(project)
		if opts.DefaultWorkspace != "" {
			workspace := obj.GetWorkspace()
			if workspace == "" {
				workspace = opts.DefaultWorkspace
			}
			obj.SetWorkspace(workspace)
		}
		return nil
	default:
		project := obj.GetProject()
		if project == "" {
			project = opts.DefaultProject
		}
		if project == "" {
			return fmt.Errorf("metadata.project is required (or pass project to Apply/List)")
		}
		obj.SetProject(project)

		workspace := obj.GetWorkspace()
		if workspace == "" {
			workspace = opts.DefaultWorkspace
		}
		if workspace == "" {
			return fmt.Errorf("metadata.workspace is required (or pass workspace to Apply/List)")
		}
		obj.SetWorkspace(workspace)
		return nil
	}
}
