// Package v1alpha1 defines gpupaas.ai API types.
package v1alpha1

const (
	Group      = "gpupaas.ai"
	Version    = "v1alpha1"
	APIVersion = Group + "/" + Version

	KindWorkspace         = "Workspace"
	KindSshKey            = "SshKey"
	KindSecurityGroup     = "SecurityGroup"
	KindStorage           = "Storage"
	KindMKSCluster        = "MKSCluster"
	KindMKSNode           = "MKSNode"
	KindMKSClusterProfile = "MKSClusterProfile"
)

// TypeMeta describes the API version and kind of a resource.
type TypeMeta struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

// ObjectMeta holds standard resource metadata.
type ObjectMeta struct {
	Name        string            `json:"name" yaml:"name"`
	Project     string            `json:"project,omitempty" yaml:"project,omitempty"`
	Workspace   string            `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	DisplayName string            `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	// CreatedBy is set by the backend on read; it is ignored on write.
	CreatedBy *UserMeta `json:"createdBy,omitempty" yaml:"createdBy,omitempty"`
	// ModifiedBy is set by the backend on read; it is ignored on write.
	ModifiedBy *UserMeta `json:"modifiedBy,omitempty" yaml:"modifiedBy,omitempty"`
}

// UserMeta identifies a user that touched a resource (read-only on the wire).
type UserMeta struct {
	Username  string           `json:"username,omitempty" yaml:"username,omitempty"`
	IsSSOUser bool             `json:"isSSOUser,omitempty" yaml:"isSSOUser,omitempty"`
	Options   *UserMetaOptions `json:"options,omitempty" yaml:"options,omitempty"`
}

// UserMetaOptions carries override hints supplied alongside a user reference.
type UserMetaOptions struct {
	Description string                   `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                     `json:"required,omitempty" yaml:"required,omitempty"`
	Override    *UserMetaOverrideOptions `json:"override,omitempty" yaml:"override,omitempty"`
}

// UserMetaOverrideOptions controls how an override value may be supplied.
type UserMetaOverrideOptions struct {
	Type             string   `json:"type,omitempty" yaml:"type,omitempty"` // allowed|notallowed|restricted
	RestrictedValues []string `json:"restrictedValues,omitempty" yaml:"restrictedValues,omitempty"`
}

func copyUserMeta(u *UserMeta) *UserMeta {
	if u == nil {
		return nil
	}
	cp := *u
	if u.Options != nil {
		opts := *u.Options
		if u.Options.Override != nil {
			ov := *u.Options.Override
			if len(u.Options.Override.RestrictedValues) > 0 {
				ov.RestrictedValues = append([]string(nil), u.Options.Override.RestrictedValues...)
			}
			opts.Override = &ov
		}
		cp.Options = &opts
	}
	return &cp
}

// ListMeta holds list metadata.
type ListMeta struct {
	Continue string `json:"continue,omitempty" yaml:"continue,omitempty"`
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

func copyObjectMeta(m ObjectMeta) ObjectMeta {
	return ObjectMeta{
		Name:        m.Name,
		Project:     m.Project,
		Workspace:   m.Workspace,
		DisplayName: m.DisplayName,
		Description: m.Description,
		Labels:      copyStringMap(m.Labels),
		Annotations: copyStringMap(m.Annotations),
		CreatedBy:   copyUserMeta(m.CreatedBy),
		ModifiedBy:  copyUserMeta(m.ModifiedBy),
	}
}

// ResourceRef references another catalog resource (e.g. VM profile).
type ResourceRef struct {
	Name          string `json:"name" yaml:"name"`
	SystemCatalog bool   `json:"systemCatalog,omitempty" yaml:"systemCatalog,omitempty"`
}
