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
	case v1alpha1.KindProject:
		return nil
	case v1alpha1.KindWorkspace:
		project := obj.GetProject()
		if project == "" {
			project = opts.DefaultProject
		}
		if project == "" {
			return fmt.Errorf("metadata.project is required (or pass project to Apply/List)")
		}
		obj.SetProject(project)
		return nil
	case v1alpha1.KindVirtualMachine, v1alpha1.KindStorage, v1alpha1.KindSecurityGroup, v1alpha1.KindSshKey, v1alpha1.KindBaremetalMachine, v1alpha1.KindMKSCluster:
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

// RequiresProject reports whether List/Get/Delete need a project argument for kind.
func RequiresProject(kind string) bool {
	return kind != v1alpha1.KindProject
}
