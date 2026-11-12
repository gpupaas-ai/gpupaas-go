package convert

import (
	"fmt"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// DevProjectSharingSpec is project sharing on the wire.
type DevProjectSharingSpec struct {
	Name       string   `json:"name"`
	Workspaces []string `json:"workspaces,omitempty"`
}

// DevSharingSpec is sharing configuration on the wire.
type DevSharingSpec struct {
	ShareMode  string                  `json:"shareMode"`
	Workspaces []string                `json:"workspaces,omitempty"`
	Projects   []DevProjectSharingSpec `json:"projects,omitempty"`
}

func toDevSharing(s apiv1.DevSharingSpec) *DevSharingSpec {
	out := &DevSharingSpec{ShareMode: s.ShareMode}
	if len(s.Workspaces) > 0 {
		out.Workspaces = append([]string(nil), s.Workspaces...)
	}
	if len(s.Projects) > 0 {
		out.Projects = make([]DevProjectSharingSpec, len(s.Projects))
		for i, p := range s.Projects {
			out.Projects[i] = DevProjectSharingSpec{Name: p.Name}
			if len(p.Workspaces) > 0 {
				out.Projects[i].Workspaces = append([]string(nil), p.Workspaces...)
			}
		}
	}
	return out
}

func fromDevSharing(s DevSharingSpec) *apiv1.DevSharingSpec {
	out := &apiv1.DevSharingSpec{ShareMode: s.ShareMode}
	if len(s.Workspaces) > 0 {
		out.Workspaces = append([]string(nil), s.Workspaces...)
	}
	if len(s.Projects) > 0 {
		out.Projects = make([]apiv1.DevProjectSharingSpec, len(s.Projects))
		for i, p := range s.Projects {
			out.Projects[i] = apiv1.DevProjectSharingSpec{Name: p.Name}
			if len(p.Workspaces) > 0 {
				out.Projects[i].Workspaces = append([]string(nil), p.Workspaces...)
			}
		}
	}
	return out
}

func devMetadataFromWire(w DevMetadata, workspace string) apiv1.ObjectMeta {
	ws := w.Workspace
	if ws == "" {
		ws = workspace
	}
	return apiv1.ObjectMeta{
		Name:        w.Name,
		Project:     w.Project,
		Workspace:   ws,
		DisplayName: w.DisplayName,
		Description: w.Description,
		Labels:      copyStringMap(w.Labels),
		Annotations: copyStringMap(w.Annotations),
		CreatedBy:   userMetaFromWire(w.CreatedBy),
		ModifiedBy:  userMetaFromWire(w.ModifiedBy),
	}
}

func devMetadataToWire(meta apiv1.ObjectMeta, project, workspace string) DevMetadata {
	projectName := meta.Project
	if projectName == "" {
		projectName = project
	}
	wsName := meta.Workspace
	if wsName == "" {
		wsName = workspace
	}
	// CreatedBy / ModifiedBy are intentionally omitted on writes; they are
	// observed metadata populated by the backend on reads.
	return DevMetadata{
		Name:        meta.Name,
		Project:     projectName,
		Workspace:   wsName,
		DisplayName: meta.DisplayName,
		Description: meta.Description,
		Labels:      copyStringMap(meta.Labels),
		Annotations: copyStringMap(meta.Annotations),
	}
}

func userMetaFromWire(w *DevUserMeta) *apiv1.UserMeta {
	if w == nil {
		return nil
	}
	out := &apiv1.UserMeta{
		Username:  w.Username,
		IsSSOUser: w.IsSSOUser,
	}
	if w.Options != nil {
		opts := &apiv1.UserMetaOptions{
			Description: w.Options.Description,
			Required:    w.Options.Required,
		}
		if w.Options.Override != nil {
			ov := &apiv1.UserMetaOverrideOptions{
				Type: w.Options.Override.Type,
			}
			if len(w.Options.Override.RestrictedValues) > 0 {
				ov.RestrictedValues = append([]string(nil), w.Options.Override.RestrictedValues...)
			}
			opts.Override = ov
		}
		out.Options = opts
	}
	return out
}

func devListContinue(meta PaaSListMetadata, itemCount int) string {
	if meta.Count > 0 {
		return fmt.Sprintf("%d", meta.Offset+int64(itemCount))
	}
	return ""
}

// DevScope identifies project- or workspace-scoped dev API paths.
type DevScope struct {
	Project   string
	Workspace string // empty => project scope
}

// DevResourceRef is the wire format for a resource reference.
type DevResourceRef struct {
	Name          string `json:"name"`
	SystemCatalog bool   `json:"systemCatalog,omitempty"`
}

// DevMetadata is resource metadata on dev.envmgmt.io.
type DevMetadata struct {
	Name        string            `json:"name"`
	Project     string            `json:"project,omitempty"`
	Workspace   string            `json:"workspace,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	// CreatedBy / ModifiedBy are set by the backend; clients should not write them.
	CreatedBy  *DevUserMeta `json:"createdBy,omitempty"`
	ModifiedBy *DevUserMeta `json:"modifiedBy,omitempty"`
}

// DevUserMeta is the wire representation of a user reference.
type DevUserMeta struct {
	Username  string              `json:"username,omitempty"`
	IsSSOUser bool                `json:"isSSOUser,omitempty"`
	Options   *DevUserMetaOptions `json:"options,omitempty"`
}

// DevUserMetaOptions is the wire representation of user options.
type DevUserMetaOptions struct {
	Description string                      `json:"description,omitempty"`
	Required    bool                        `json:"required,omitempty"`
	Override    *DevUserMetaOverrideOptions `json:"override,omitempty"`
}

// DevUserMetaOverrideOptions is the wire representation of user override options.
type DevUserMetaOverrideOptions struct {
	Type             string   `json:"type,omitempty"`
	RestrictedValues []string `json:"restrictedValues,omitempty"`
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// PaaSListMetadata is list metadata on paas.envmgmt.io and dev.envmgmt.io list responses.
type PaaSListMetadata struct {
	Count  int64 `json:"count,omitempty"`
	Limit  int64 `json:"limit,omitempty"`
	Offset int64 `json:"offset,omitempty"`
}
