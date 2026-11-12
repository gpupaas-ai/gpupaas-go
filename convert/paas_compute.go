package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	// PaaSComputeInstanceKind and PaaSComputeInstanceListKind are the wire kinds.
	PaaSComputeInstanceKind     = "ComputeInstance"
	PaaSComputeInstanceListKind = "ComputeInstanceList"
)

// ComputeInstancePaths returns path builders for compute instance endpoints.
// Workspace-scoped builders are returned when scope.Workspace is set.
func ComputeInstancePaths(scope DevScope) (
	collection func() string,
	item func(name string) string,
) {
	if scope.Workspace != "" {
		collection = func() string {
			return fmt.Sprintf(PaaSWorkspaceComputeInstancesPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
		}
		item = func(name string) string {
			return fmt.Sprintf(PaaSWorkspaceComputeInstancePath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
		}
	} else {
		collection = func() string {
			return fmt.Sprintf(PaaSComputeInstancesPath, url.PathEscape(scope.Project))
		}
		item = func(name string) string {
			return fmt.Sprintf(PaaSComputeInstancePath, url.PathEscape(scope.Project), url.PathEscape(name))
		}
	}
	return
}

// PaaSComputeProvisionedResource is the nested provisioning reference on the wire.
type PaaSComputeProvisionedResource struct {
	Provisioner  string `json:"provisioner,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
	EaaS         *struct {
		TriggerID string `json:"triggerId,omitempty"`
	} `json:"eaas,omitempty"`
}

// PaaSComputeInstanceStatus is the wire status of a compute instance.
type PaaSComputeInstanceStatus struct {
	Status              string                          `json:"status,omitempty"`
	Reason              string                          `json:"reason,omitempty"`
	Action              string                          `json:"action,omitempty"`
	WorkflowStatus      map[string]any                  `json:"workflowStatus,omitempty"`
	Environment         *apiv1.ComputeEnvironmentRef    `json:"environment,omitempty"`
	ProvisionedResource *PaaSComputeProvisionedResource `json:"provisionedResource,omitempty"`
}

// PaaSComputeInstance is the wire format of a compute instance on paas.envmgmt.io/v1. The spec is not
// modelled: the SDK reads compute instances for their status only.
type PaaSComputeInstance struct {
	APIVersion string                     `json:"apiVersion"`
	Kind       string                     `json:"kind"`
	Metadata   DevMetadata                `json:"metadata"`
	Status     *PaaSComputeInstanceStatus `json:"status,omitempty"`
}

// PaaSComputeInstanceList is the wire format for compute instance list responses.
type PaaSComputeInstanceList struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   PaaSListMetadata      `json:"metadata,omitempty"`
	Items      []PaaSComputeInstance `json:"items"`
}

// FromPaaSComputeInstance converts a wire compute instance to the SDK type. The
// backend omits metadata.project (it returns projectID) on some responses, so the
// request scope is used as a fallback for project and workspace.
func FromPaaSComputeInstance(w *PaaSComputeInstance, project, workspace string) *apiv1.ComputeInstance {
	if w == nil {
		return nil
	}
	out := &apiv1.ComputeInstance{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindComputeInstance},
		Metadata: devMetadataFromWire(w.Metadata, workspace),
	}
	if out.Metadata.Project == "" {
		out.Metadata.Project = project
	}
	if s := w.Status; s != nil {
		out.Status = apiv1.ComputeInstanceStatus{
			Status:         s.Status,
			Reason:         s.Reason,
			Action:         s.Action,
			WorkflowStatus: s.WorkflowStatus,
			Environment:    s.Environment,
		}
		if pr := s.ProvisionedResource; pr != nil {
			out.Status.ProvisionedResource = &apiv1.ComputeProvisionedResource{
				Provisioner:  pr.Provisioner,
				ResourceID:   pr.ResourceID,
				ResourceName: pr.ResourceName,
			}
			if pr.EaaS != nil {
				out.Status.ProvisionedResource.TriggerID = pr.EaaS.TriggerID
			}
		}
	}
	return out
}

// FromPaaSComputeInstanceList converts a wire list to the SDK list.
func FromPaaSComputeInstanceList(list *PaaSComputeInstanceList, project, workspace string) *apiv1.ComputeInstanceList {
	out := &apiv1.ComputeInstanceList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: PaaSComputeInstanceListKind},
		Items:    make([]apiv1.ComputeInstance, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if c := FromPaaSComputeInstance(&list.Items[i], project, workspace); c != nil {
			out.Items = append(out.Items, *c)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}
