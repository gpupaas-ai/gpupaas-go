package convert

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	DevStorageKind     = "Storage"
	DevStorageListKind = "StorageList"
)

// StoragePaths returns the collection and item path builders for the given scope.
func StoragePaths(scope DevScope) (collection func() string, item func(name string) string) {
	if scope.Workspace != "" {
		return func() string {
				return fmt.Sprintf(DevWorkspaceStoragesPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
			},
			func(name string) string {
				return fmt.Sprintf(DevWorkspaceStoragePath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
			}
	}
	return func() string {
			return fmt.Sprintf(DevProjectStoragesPath, url.PathEscape(scope.Project))
		},
		func(name string) string {
			return fmt.Sprintf(DevProjectStoragePath, url.PathEscape(scope.Project), url.PathEscape(name))
		}
}

// DevStorageSpec is the wire spec for storage.
type DevStorageSpec struct {
	// Storage names the profile again, as the platform's own CLI sends it. A controller reads the profile from
	// Type; the reference is written for compatibility and never read back.
	Storage                   *DevResourceRef `json:"storage,omitempty"`
	Type                      string          `json:"type,omitempty"`
	Size                      string          `json:"size,omitempty"`
	Datacenter                string          `json:"datacenter,omitempty"`
	StorageType               string          `json:"storage_type,omitempty"`
	AccessPolicy              string          `json:"access_policy,omitempty"`
	ContractTerm              string          `json:"contract_term,omitempty"`
	EnableEncryptionAtRest    string          `json:"enable_encryption_at_rest,omitempty"`
	EnableEncryptionInTransit string          `json:"enable_encryption_in_transit,omitempty"`
	Sharing                   *DevSharingSpec `json:"sharing,omitempty"`
}

// DevStorageStatus is runtime status on the wire. Output holds values of mixed types (strings and numbers), kept as
// raw JSON so that no number is rounded on the way through a float64.
type DevStorageStatus struct {
	Status string                     `json:"status,omitempty"`
	Reason string                     `json:"reason,omitempty"`
	Action string                     `json:"action,omitempty"`
	Output map[string]json.RawMessage `json:"output,omitempty"`
}

// DevStorage is the wire format for storage CRUD.
type DevStorage struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   DevMetadata      `json:"metadata"`
	Spec       DevStorageSpec   `json:"spec,omitempty"`
	Status     DevStorageStatus `json:"status,omitempty"`
}

// DevStorageList is the wire format for storage list responses.
type DevStorageList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []DevStorage     `json:"items"`
}

// ToDevStorage converts an SDK storage to the wire format.
func ToDevStorage(s *apiv1.Storage, project, workspace string) *DevStorage {
	if s == nil {
		return nil
	}
	spec := DevStorageSpec{
		Type:                      s.Spec.Type,
		Size:                      s.Spec.Size,
		Datacenter:                s.Spec.Datacenter,
		StorageType:               s.Spec.StorageType,
		AccessPolicy:              s.Spec.AccessPolicy,
		ContractTerm:              s.Spec.ContractTerm,
		EnableEncryptionAtRest:    s.Spec.EnableEncryptionAtRest,
		EnableEncryptionInTransit: s.Spec.EnableEncryptionInTransit,
	}
	if s.Spec.Type != "" {
		spec.Storage = &DevResourceRef{Name: s.Spec.Type}
	}
	if s.Spec.Sharing != nil {
		spec.Sharing = toDevSharing(*s.Spec.Sharing)
	}
	return &DevStorage{
		APIVersion: DevAPIVersion,
		Kind:       DevStorageKind,
		Metadata:   devMetadataToWire(s.Metadata, project, workspace),
		Spec:       spec,
	}
}

// FromDevStorage converts wire format to an SDK storage.
func FromDevStorage(wire *DevStorage, workspace string) *apiv1.Storage {
	if wire == nil {
		return nil
	}
	out := &apiv1.Storage{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindStorage},
		Metadata: devMetadataFromWire(wire.Metadata, workspace),
		Spec: apiv1.StorageSpec{
			Type:                      wire.Spec.Type,
			Size:                      wire.Spec.Size,
			Datacenter:                wire.Spec.Datacenter,
			StorageType:               wire.Spec.StorageType,
			AccessPolicy:              wire.Spec.AccessPolicy,
			ContractTerm:              wire.Spec.ContractTerm,
			EnableEncryptionAtRest:    wire.Spec.EnableEncryptionAtRest,
			EnableEncryptionInTransit: wire.Spec.EnableEncryptionInTransit,
		},
		Status: apiv1.StorageStatus{
			Status: wire.Status.Status,
			Reason: wire.Status.Reason,
			Action: wire.Status.Action,
			Output: outputStrings(wire.Status.Output),
		},
	}
	if wire.Spec.Sharing != nil {
		out.Spec.Sharing = fromDevSharing(*wire.Spec.Sharing)
	}
	return out
}

// outputStrings turns the mixed-type output of a storage into strings. A string stays as it is; a number keeps the
// digits the platform sent (volume_size 100000000000 stays 100000000000, whatever its size); null is empty; a bool or
// a nested value is its JSON.
func outputStrings(in map[string]json.RawMessage) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, raw := range in {
		text := strings.TrimSpace(string(raw))
		switch {
		case text == "" || text == "null":
			out[k] = ""
		case strings.HasPrefix(text, `"`):
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				out[k] = text
			} else {
				out[k] = v
			}
		default:
			out[k] = text
		}
	}
	return out
}

// FromDevStorageList converts a wire list to an SDK storage list.
func FromDevStorageList(wire *DevStorageList, workspace string) *apiv1.StorageList {
	out := &apiv1.StorageList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: DevStorageListKind},
		Items:    make([]apiv1.Storage, 0),
	}
	if wire == nil {
		return out
	}
	out.Metadata.Continue = devListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if item := FromDevStorage(&wire.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
