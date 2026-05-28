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
