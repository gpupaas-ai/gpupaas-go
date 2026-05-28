package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	DevSshKeyKind     = "SshKey"
	DevSshKeyListKind = "SshKeyList"
)

// SshKeyPaths returns collection and item path builders for the given scope.
func SshKeyPaths(scope DevScope) (collection func() string, item func(name string) string) {
	if scope.Workspace != "" {
		return func() string {
				return fmt.Sprintf(DevWorkspaceSshKeysPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
			},
			func(name string) string {
				return fmt.Sprintf(DevWorkspaceSshKeyPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
			}
	}
	return func() string {
			return fmt.Sprintf(DevProjectSshKeysPath, url.PathEscape(scope.Project))
		},
		func(name string) string {
			return fmt.Sprintf(DevProjectSshKeyPath, url.PathEscape(scope.Project), url.PathEscape(name))
		}
}

// DevSshKeySpec is the wire spec for an SSH key.
type DevSshKeySpec struct {
	SSHKey    DevResourceRef  `json:"ssh_key"`
	Type      string          `json:"type,omitempty"`
	Name      string          `json:"name,omitempty"`
	PublicKey string          `json:"public_key,omitempty"`
	Sharing   *DevSharingSpec `json:"sharing,omitempty"`
}

// DevSshKeyStatus is runtime status on the wire.
type DevSshKeyStatus struct {
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
	Action string `json:"action,omitempty"`
}

// DevSshKey is the wire format for SSH key CRUD.
type DevSshKey struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   DevMetadata     `json:"metadata"`
	Spec       DevSshKeySpec   `json:"spec,omitempty"`
	Status     DevSshKeyStatus `json:"status,omitempty"`
}

// DevSshKeyList is the wire format for SSH key list responses.
type DevSshKeyList struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []DevSshKey     `json:"items"`
}

// ToDevSshKey converts SDK SSH key to wire format.
func ToDevSshKey(k *apiv1.SshKey, project, workspace string) *DevSshKey {
	if k == nil {
		return nil
	}
	return &DevSshKey{
		APIVersion: DevAPIVersion,
		Kind:       DevSshKeyKind,
		Metadata:   devMetadataToWire(k.Metadata, project, workspace),
		Spec:       toDevSshKeySpec(k.Spec),
		Status:     DevSshKeyStatus{},
	}
}

func toDevSshKeySpec(s apiv1.SshKeySpec) DevSshKeySpec {
	out := DevSshKeySpec{
		SSHKey: DevResourceRef{
			Name:          s.SSHKey.Name,
			SystemCatalog: s.SSHKey.SystemCatalog,
		},
		Type:      s.Type,
		Name:      s.Name,
		PublicKey: s.PublicKey,
	}
	if s.Sharing != nil {
		out.Sharing = toDevSharing(*s.Sharing)
	}
	return out
}

// FromDevSshKey converts wire format to SDK SSH key.
func FromDevSshKey(wire *DevSshKey, workspace string) *apiv1.SshKey {
	if wire == nil {
		return nil
	}
	return &apiv1.SshKey{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindSshKey,
		},
		Metadata: devMetadataFromWire(wire.Metadata, workspace),
		Spec:     fromDevSshKeySpec(wire.Spec),
		Status:   fromDevSshKeyStatus(wire.Status),
	}
}

func fromDevSshKeySpec(s DevSshKeySpec) apiv1.SshKeySpec {
	out := apiv1.SshKeySpec{
		SSHKey: apiv1.ResourceRef{
			Name:          s.SSHKey.Name,
			SystemCatalog: s.SSHKey.SystemCatalog,
		},
		Type:      s.Type,
		Name:      s.Name,
		PublicKey: s.PublicKey,
	}
	if s.Sharing != nil {
		out.Sharing = fromDevSharing(*s.Sharing)
	}
	return out
}

func fromDevSshKeyStatus(s DevSshKeyStatus) apiv1.SshKeyStatus {
	return apiv1.SshKeyStatus{
		Status: s.Status,
		Reason: s.Reason,
		Action: s.Action,
	}
}

// FromDevSshKeyList converts a wire list to SDK SSH key list.
func FromDevSshKeyList(wire *DevSshKeyList, workspace string) *apiv1.SshKeyList {
	if wire == nil {
		return &apiv1.SshKeyList{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindSshKey + "List"},
		}
	}
	out := &apiv1.SshKeyList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindSshKey + "List",
		},
	}
	out.Metadata.Continue = devListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if item := FromDevSshKey(&wire.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
