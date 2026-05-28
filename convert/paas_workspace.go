package convert

import (
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// PaaSMetadata is workspace metadata on the paas.envmgmt.io API.
type PaaSMetadata struct {
	Name        string            `json:"name"`
	Project     string            `json:"project"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PaaSWorkspaceSpec is the workspace spec on the paas.envmgmt.io API.
type PaaSWorkspaceSpec struct {
	IconURL string `json:"iconURL,omitempty"`
	Readme  string `json:"readme,omitempty"`
}

// PaaSStatus is a subset of the common status object returned by the backend.
type PaaSStatus struct {
	ConditionStatus string `json:"conditionStatus,omitempty"`
	ConditionType   string `json:"conditionType,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

// PaaSWorkspaceStatus is workspace status on the paas.envmgmt.io API.
type PaaSWorkspaceStatus struct {
	CommonStatus PaaSStatus `json:"commonStatus,omitempty"`
}

// PaaSWorkspace is the wire format for workspace apply/get on paas.envmgmt.io/v1.
type PaaSWorkspace struct {
	APIVersion string              `json:"apiVersion"`
	Kind       string              `json:"kind"`
	Metadata   PaaSMetadata        `json:"metadata"`
	Spec       PaaSWorkspaceSpec   `json:"spec,omitempty"`
	Status     PaaSWorkspaceStatus `json:"status,omitempty"`
}

// PaaSListMetadata is list metadata on paas.envmgmt.io list responses.
type PaaSListMetadata struct {
	Count  int64 `json:"count,omitempty"`
	Limit  int64 `json:"limit,omitempty"`
	Offset int64 `json:"offset,omitempty"`
}

// PaaSWorkspaceList is the wire format for workspace list responses.
type PaaSWorkspaceList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []PaaSWorkspace  `json:"items"`
}

// ToPaaSWorkspace converts a Kubernetes-style Workspace to the paas.envmgmt.io wire format.
func ToPaaSWorkspace(w *apiv1.Workspace, project string) *PaaSWorkspace {
	if w == nil {
		return nil
	}
	projectName := w.Metadata.Project
	if projectName == "" {
		projectName = project
	}
	description := w.Spec.Description
	displayName := w.Spec.DisplayName
	if displayName == "" {
		displayName = w.Metadata.Name
	}
	return &PaaSWorkspace{
		APIVersion: PaaSWorkspaceAPIVer,
		Kind:       PaaSWorkspaceKind,
		Metadata: PaaSMetadata{
			Name:        w.Metadata.Name,
			Project:     projectName,
			DisplayName: displayName,
			Description: description,
			Labels:      w.Metadata.Labels,
			Annotations: w.Metadata.Annotations,
		},
		Spec: PaaSWorkspaceSpec{
			IconURL: w.Spec.IconURL,
			Readme:  w.Spec.Readme,
		},
	}
}

// FromPaaSWorkspace converts a paas.envmgmt.io workspace to a Kubernetes-style Workspace.
func FromPaaSWorkspace(w *PaaSWorkspace) *apiv1.Workspace {
	if w == nil {
		return nil
	}
	phase := w.Status.CommonStatus.ConditionStatus
	if phase == "" {
		phase = w.Status.CommonStatus.Reason
	}
	return &apiv1.Workspace{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindWorkspace,
		},
		Metadata: apiv1.ObjectMeta{
			Name:        w.Metadata.Name,
			Project:     w.Metadata.Project,
			Labels:      w.Metadata.Labels,
			Annotations: w.Metadata.Annotations,
		},
		Spec: apiv1.WorkspaceSpec{
			DisplayName: w.Metadata.DisplayName,
			Description: w.Metadata.Description,
			IconURL:     w.Spec.IconURL,
			Readme:      w.Spec.Readme,
		},
		Status: apiv1.WorkspaceStatus{Phase: phase},
	}
}

// FromPaaSWorkspaceList converts a paas.envmgmt.io workspace list to a WorkspaceList.
func FromPaaSWorkspaceList(list *PaaSWorkspaceList) *apiv1.WorkspaceList {
	out := &apiv1.WorkspaceList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindWorkspace + "List",
		},
		Items: make([]apiv1.Workspace, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if ws := FromPaaSWorkspace(&list.Items[i]); ws != nil {
			out.Items = append(out.Items, *ws)
		}
	}
	return out
}
