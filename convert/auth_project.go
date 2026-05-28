package convert

import (
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// AuthProject is the auth service project payload (GET /auth/v1/projects/{id}/).
type AuthProject struct {
	ID                           string `json:"id,omitempty"`
	Name                         string `json:"name"`
	Description                  string `json:"description,omitempty"`
	Default                      bool   `json:"default,omitempty"`
	CreatedAt                    string `json:"created_at,omitempty"`
	ModifiedAt                   string `json:"modified_at,omitempty"`
	PartnerID                    string `json:"partner_id,omitempty"`
	OrganizationID               string `json:"organization_id,omitempty"`
	EnableDriftWebhook           bool   `json:"enable_drift_webhook,omitempty"`
	ClusterResourceQuota         any    `json:"cluster_resource_quota,omitempty"`
	DefaultClusterNamespaceQuota any    `json:"default_cluster_namespace_quota,omitempty"`
}

const projectIDAnnotation = "gpupaas.ai/project-id"

// AuthProjectList is the paginated list response from GET /auth/v1/projects/.
type AuthProjectList struct {
	Count    int           `json:"count"`
	Next     *string       `json:"next"`
	Previous *string       `json:"previous"`
	Results  []AuthProject `json:"results"`
}

// FromAuthProject converts a backend auth project to a Kubernetes-style Project.
func FromAuthProject(p *AuthProject) *apiv1.Project {
	if p == nil {
		return nil
	}
	name := p.Name
	if name == "" {
		name = p.ID
	}
	out := &apiv1.Project{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindProject,
		},
		Metadata: apiv1.ObjectMeta{Name: name},
		Spec: apiv1.ProjectSpec{
			DisplayName: p.Name,
			Description: p.Description,
			Default:     p.Default,
		},
	}
	if p.ID != "" {
		if out.Metadata.Annotations == nil {
			out.Metadata.Annotations = make(map[string]string)
		}
		out.Metadata.Annotations[projectIDAnnotation] = p.ID
	}
	if p.Default {
		out.Status.Phase = "Default"
	}
	return out
}

// ToAuthProject converts a Kubernetes-style Project to the auth service payload.
func ToAuthProject(p *apiv1.Project) *AuthProject {
	if p == nil {
		return nil
	}
	name := p.Metadata.Name
	return &AuthProject{
		Name:        name,
		Description: p.Spec.Description,
		Default:     p.Spec.Default,
	}
}

// FromAuthProjectList converts a paginated auth project list to a ProjectList.
func FromAuthProjectList(page *AuthProjectList) *apiv1.ProjectList {
	list := &apiv1.ProjectList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindProject + "List",
		},
		Items: make([]apiv1.Project, 0),
	}
	if page == nil {
		return list
	}
	if page.Next != nil && *page.Next != "" {
		list.Metadata.Continue = *page.Next
	}
	for i := range page.Results {
		if proj := FromAuthProject(&page.Results[i]); proj != nil {
			list.Items = append(list.Items, *proj)
		}
	}
	return list
}

// FindAuthProjectIDByName scans a list page for a project name and returns its id.
func FindAuthProjectIDByName(page *AuthProjectList, name string) (string, bool) {
	if page == nil {
		return "", false
	}
	for i := range page.Results {
		if page.Results[i].Name == name {
			return page.Results[i].ID, true
		}
	}
	return "", false
}

// ProjectIDFromObject returns the auth project id from annotations when present.
func ProjectIDFromObject(p *apiv1.Project) string {
	if p == nil || p.Metadata.Annotations == nil {
		return ""
	}
	return p.Metadata.Annotations[projectIDAnnotation]
}
