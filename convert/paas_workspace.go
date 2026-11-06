package convert

import (
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// PaaSMetadata is workspace metadata on the paas.envmgmt.io API.
type PaaSMetadata struct {
	Name        string            `json:"name"`
	Project     string            `json:"project,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PaaSWorkspace is the wire format for workspace create/get on paas.envmgmt.io/v1.
type PaaSWorkspace struct {
	APIVersion string       `json:"apiVersion"`
	Kind       string       `json:"kind"`
	Metadata   PaaSMetadata `json:"metadata"`
}

// PaaSWorkspaceList is the wire format for workspace list responses.
type PaaSWorkspaceList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []PaaSWorkspace  `json:"items"`
}

// ToPaaSWorkspace converts an SDK Workspace to the wire format.
func ToPaaSWorkspace(w *apiv1.Workspace, project string) *PaaSWorkspace {
	if w == nil {
		return nil
	}
	projectName := w.Metadata.Project
	if projectName == "" {
		projectName = project
	}
	// The console lists workspaces by display name, so an unset one defaults to the name.
	displayName := w.Metadata.DisplayName
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
			Description: w.Metadata.Description,
			Labels:      copyStringMap(w.Metadata.Labels),
			Annotations: copyStringMap(w.Metadata.Annotations),
		},
	}
}

// FromPaaSWorkspace converts a wire workspace to an SDK Workspace.
func FromPaaSWorkspace(w *PaaSWorkspace) *apiv1.Workspace {
	if w == nil {
		return nil
	}
	return &apiv1.Workspace{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindWorkspace,
		},
		Metadata: apiv1.ObjectMeta{
			Name:        w.Metadata.Name,
			Project:     w.Metadata.Project,
			DisplayName: w.Metadata.DisplayName,
			Description: w.Metadata.Description,
			Labels:      copyStringMap(w.Metadata.Labels),
			Annotations: copyStringMap(w.Metadata.Annotations),
		},
	}
}

// FromPaaSWorkspaceList converts a wire workspace list to an SDK WorkspaceList.
func FromPaaSWorkspaceList(list *PaaSWorkspaceList) *apiv1.WorkspaceList {
	out := &apiv1.WorkspaceList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: PaaSWorkspaceListKind},
		Items:    make([]apiv1.Workspace, 0),
	}
	if list == nil {
		return out
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	for i := range list.Items {
		if ws := FromPaaSWorkspace(&list.Items[i]); ws != nil {
			out.Items = append(out.Items, *ws)
		}
	}
	return out
}
