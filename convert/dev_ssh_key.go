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

// SshKeyPaths returns the collection and item path builders for the given scope.
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
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []DevSshKey      `json:"items"`
}

// ToDevSshKey converts an SDK SSH key to the wire format.
func ToDevSshKey(k *apiv1.SshKey, project, workspace string) *DevSshKey {
	if k == nil {
		return nil
	}
	out := &DevSshKey{
		APIVersion: DevAPIVersion,
		Kind:       DevSshKeyKind,
		Metadata:   devMetadataToWire(k.Metadata, project, workspace),
		Spec: DevSshKeySpec{
			Type:      k.Spec.Type,
			Name:      k.Spec.Name,
			PublicKey: k.Spec.PublicKey,
		},
	}
	if k.Spec.Sharing != nil {
		out.Spec.Sharing = toDevSharing(*k.Spec.Sharing)
	}
	return out
}

// FromDevSshKey converts wire format to SDK SSH key.
func FromDevSshKey(wire *DevSshKey, workspace string) *apiv1.SshKey {
	if wire == nil {
		return nil
	}
	out := &apiv1.SshKey{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindSshKey,
		},
		Metadata: devMetadataFromWire(wire.Metadata, workspace),
		Spec: apiv1.SshKeySpec{
			Type:      wire.Spec.Type,
			Name:      wire.Spec.Name,
			PublicKey: wire.Spec.PublicKey,
		},
		Status: apiv1.SshKeyStatus{
			Status: wire.Status.Status,
			Reason: wire.Status.Reason,
			Action: wire.Status.Action,
		},
	}
	if wire.Spec.Sharing != nil {
		out.Spec.Sharing = fromDevSharing(*wire.Spec.Sharing)
	}
	return out
}

// FromDevSshKeyList converts a wire list to an SDK SSH key list.
func FromDevSshKeyList(wire *DevSshKeyList, workspace string) *apiv1.SshKeyList {
	out := &apiv1.SshKeyList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: DevSshKeyListKind},
		Items:    make([]apiv1.SshKey, 0),
	}
	if wire == nil {
		return out
	}
	out.Metadata.Continue = devListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if item := FromDevSshKey(&wire.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
