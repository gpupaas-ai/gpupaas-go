package convert

import (
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	PaaSWorkspaceCollaboratorKind     = "WorkspaceCollaborator"
	PaaSWorkspaceCollaboratorListKind = "WorkspaceCollaboratorList"
	PaaSWorkspaceAddCollaboratorsKind = "WorkspaceAddCollaborators"
	PaaSWorkspaceDeleteCollabKind     = "WorkspaceDeleteCollaborators"
)

// PaaSWorkspaceCollaboratorSpec is the wire spec for a workspace collaborator.
type PaaSWorkspaceCollaboratorSpec struct {
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Role      string `json:"role,omitempty"`
	UserType  string `json:"userType,omitempty"`
}

// PaaSWorkspaceCollaborator is the wire format for POST/GET .../collaborators.
type PaaSWorkspaceCollaborator struct {
	APIVersion string                        `json:"apiVersion"`
	Kind       string                        `json:"kind"`
	Metadata   PaaSMetadata                  `json:"metadata"`
	Spec       PaaSWorkspaceCollaboratorSpec `json:"spec,omitempty"`
	Status     PaaSStatus                    `json:"status,omitempty"`
}

// PaaSWorkspaceCollaboratorList is the wire format for GET .../collaborators.
type PaaSWorkspaceCollaboratorList struct {
	APIVersion string                      `json:"apiVersion"`
	Kind       string                      `json:"kind"`
	Metadata   PaaSListMetadata            `json:"metadata,omitempty"`
	Items      []PaaSWorkspaceCollaborator `json:"items"`
}

// PaaSWorkspaceAddCollaboratorsSpec assigns existing Rafay users to a workspace.
type PaaSWorkspaceAddCollaboratorsSpec struct {
	Usernames []string `json:"Usernames,omitempty"`
	Role      string   `json:"role,omitempty"`
}

// PaaSWorkspaceAddCollaborators is the wire format for POST .../assigncollaborators.
type PaaSWorkspaceAddCollaborators struct {
	APIVersion string                            `json:"apiVersion"`
	Kind       string                            `json:"kind"`
	Metadata   PaaSMetadata                      `json:"metadata"`
	Spec       PaaSWorkspaceAddCollaboratorsSpec `json:"spec"`
	Status     PaaSStatus                        `json:"status,omitempty"`
}

// PaaSWorkspaceDeleteCollaboratorsSpec removes collaborators by username.
type PaaSWorkspaceDeleteCollaboratorsSpec struct {
	Usernames []string `json:"Usernames,omitempty"`
}

// PaaSWorkspaceDeleteCollaborators is the wire format for POST .../unassigncollaborators.
type PaaSWorkspaceDeleteCollaborators struct {
	APIVersion string                               `json:"apiVersion"`
	Kind       string                               `json:"kind"`
	Metadata   PaaSMetadata                         `json:"metadata"`
	Spec       PaaSWorkspaceDeleteCollaboratorsSpec `json:"spec"`
	Status     PaaSStatus                           `json:"status,omitempty"`
}

// ToPaaSWorkspaceCollaborator converts a k8s-style collaborator for POST .../collaborators.
func ToPaaSWorkspaceCollaborator(c *apiv1.WorkspaceCollaborator, project, workspace string) *PaaSWorkspaceCollaborator {
	if c == nil {
		return nil
	}
	projectName := c.Metadata.Project
	if projectName == "" {
		projectName = project
	}
	name := c.Metadata.Name
	if name == "" {
		name = c.Spec.Email
	}
	return &PaaSWorkspaceCollaborator{
		APIVersion: PaaSWorkspaceAPIVer,
		Kind:       PaaSWorkspaceCollaboratorKind,
		Metadata: PaaSMetadata{
			Name:        name,
			Project:     projectName,
			Labels:      c.Metadata.Labels,
			Annotations: c.Metadata.Annotations,
		},
		Spec: PaaSWorkspaceCollaboratorSpec{
			Email:     c.Spec.Email,
			FirstName: c.Spec.FirstName,
			LastName:  c.Spec.LastName,
			Role:      RoleForSpec(c.Spec.ResolvedRole()),
			UserType:  c.Spec.UserType,
		},
	}
}

// ToPaaSWorkspaceAddCollaborators converts a k8s-style collaborator for POST .../assigncollaborators.
func ToPaaSWorkspaceAddCollaborators(c *apiv1.WorkspaceCollaborator, project, workspace string) *PaaSWorkspaceAddCollaborators {
	if c == nil {
		return nil
	}
	projectName := c.Metadata.Project
	if projectName == "" {
		projectName = project
	}
	wsName := c.Metadata.Workspace
	if wsName == "" {
		wsName = workspace
	}
	username := c.CollaboratorUsername()
	return &PaaSWorkspaceAddCollaborators{
		APIVersion: PaaSWorkspaceAPIVer,
		Kind:       PaaSWorkspaceAddCollaboratorsKind,
		Metadata: PaaSMetadata{
			Name:    wsName,
			Project: projectName,
		},
		Spec: PaaSWorkspaceAddCollaboratorsSpec{
			Usernames: []string{username},
			Role:      RoleForSpec(c.Spec.ResolvedRole()),
		},
	}
}

// ToPaaSWorkspaceDeleteCollaborators converts a username for POST .../unassigncollaborators.
func ToPaaSWorkspaceDeleteCollaborators(project, workspace, username string) *PaaSWorkspaceDeleteCollaborators {
	return &PaaSWorkspaceDeleteCollaborators{
		APIVersion: PaaSWorkspaceAPIVer,
		Kind:       PaaSWorkspaceDeleteCollabKind,
		Metadata: PaaSMetadata{
			Name:    workspace,
			Project: project,
		},
		Spec: PaaSWorkspaceDeleteCollaboratorsSpec{
			Usernames: []string{username},
		},
	}
}

// FromPaaSWorkspaceCollaborator converts a wire collaborator to k8s style.
func FromPaaSWorkspaceCollaborator(c *PaaSWorkspaceCollaborator, workspace string) *apiv1.WorkspaceCollaborator {
	if c == nil {
		return nil
	}
	name := c.Metadata.Name
	if name == "" {
		name = c.Spec.Email
	}
	phase := c.Status.ConditionStatus
	if phase == "" {
		phase = c.Status.Reason
	}
	role := c.Spec.Role
	return &apiv1.WorkspaceCollaborator{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindWorkspaceCollaborator,
		},
		Metadata: apiv1.ObjectMeta{
			Name:        name,
			Project:     c.Metadata.Project,
			Workspace:   workspace,
			Labels:      c.Metadata.Labels,
			Annotations: c.Metadata.Annotations,
		},
		Spec: apiv1.WorkspaceCollaboratorSpec{
			Username:  name,
			Email:     c.Spec.Email,
			FirstName: c.Spec.FirstName,
			LastName:  c.Spec.LastName,
			Role:      role,
			UserType:  c.Spec.UserType,
		},
		Status: apiv1.WorkspaceCollaboratorStatus{
			Phase: phase,
			Role:  role,
		},
	}
}

// FromPaaSWorkspaceCollaboratorList converts a wire list to k8s style.
func FromPaaSWorkspaceCollaboratorList(list *PaaSWorkspaceCollaboratorList, workspace string) *apiv1.WorkspaceCollaboratorList {
	out := &apiv1.WorkspaceCollaboratorList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindWorkspaceCollaborator + "List",
		},
		Items: make([]apiv1.WorkspaceCollaborator, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if item := FromPaaSWorkspaceCollaborator(&list.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
