package convert

import (
	"fmt"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// InfraUserMeta is the wire representation of a user reference on infra.k8smgmt.io.
type InfraUserMeta struct {
	Username  string                `json:"username,omitempty"`
	IsSSOUser bool                  `json:"isSSOUser,omitempty"`
	Options   *InfraUserMetaOptions `json:"options,omitempty"`
}

// InfraUserMetaOptions is the wire representation of user options on infra.k8smgmt.io.
type InfraUserMetaOptions struct {
	Description string                        `json:"description,omitempty"`
	Required    bool                          `json:"required,omitempty"`
	Override    *InfraUserMetaOverrideOptions `json:"override,omitempty"`
}

// InfraUserMetaOverrideOptions is the wire representation of override options.
type InfraUserMetaOverrideOptions struct {
	Type             string   `json:"type,omitempty"`
	RestrictedValues []string `json:"restrictedValues,omitempty"`
}

// InfraMetadata is resource metadata on infra.k8smgmt.io.
// It does not carry a workspace (infra resources are project-scoped).
type InfraMetadata struct {
	Name        string            `json:"name"`
	Project     string            `json:"project,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	// CreatedBy / ModifiedBy are populated by the backend on reads.
	CreatedBy  *InfraUserMeta `json:"createdBy,omitempty"`
	ModifiedBy *InfraUserMeta `json:"modifiedBy,omitempty"`
}

func infraMetadataToWire(meta apiv1.ObjectMeta, project string) InfraMetadata {
	projectName := meta.Project
	if projectName == "" {
		projectName = project
	}
	// CreatedBy / ModifiedBy are intentionally omitted on writes; they are
	// observed metadata populated by the backend on reads.
	return InfraMetadata{
		Name:        meta.Name,
		Project:     projectName,
		DisplayName: meta.DisplayName,
		Description: meta.Description,
		Labels:      copyStringMap(meta.Labels),
		Annotations: copyStringMap(meta.Annotations),
	}
}

func infraMetadataFromWire(w InfraMetadata) apiv1.ObjectMeta {
	return apiv1.ObjectMeta{
		Name:        w.Name,
		Project:     w.Project,
		DisplayName: w.DisplayName,
		Description: w.Description,
		Labels:      copyStringMap(w.Labels),
		Annotations: copyStringMap(w.Annotations),
		CreatedBy:   infraUserMetaFromWire(w.CreatedBy),
		ModifiedBy:  infraUserMetaFromWire(w.ModifiedBy),
	}
}

func infraUserMetaFromWire(w *InfraUserMeta) *apiv1.UserMeta {
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

func infraListContinue(meta PaaSListMetadata, itemCount int) string {
	if meta.Count > 0 {
		return fmt.Sprintf("%d", meta.Offset+int64(itemCount))
	}
	return ""
}
