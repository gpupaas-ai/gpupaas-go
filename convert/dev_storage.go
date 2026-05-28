package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	DevStorageKind     = "Storage"
	DevStorageListKind = "StorageList"
)

// StoragePaths returns collection and item path builders for the given scope.
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
	Storage                   DevResourceRef  `json:"storage"`
	Type                      string          `json:"type,omitempty"`
	Size                      string          `json:"size,omitempty"`
	Datacenter                string          `json:"datacenter,omitempty"`
	AccessPolicy              string          `json:"access_policy,omitempty"`
	ContractTerm              string          `json:"contract_term,omitempty"`
	EnableEncryptionAtRest    string          `json:"enable_encryption_at_rest,omitempty"`
	EnableEncryptionInTransit string          `json:"enable_encryption_in_transit,omitempty"`
	Sharing                   *DevSharingSpec `json:"sharing,omitempty"`
	StorageType               string          `json:"storage_type,omitempty"`
}

// DevStorageStatus is runtime status on the wire.
type DevStorageStatus struct {
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
	Action string `json:"action,omitempty"`
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

// ToDevStorage converts SDK storage to wire format.
func ToDevStorage(s *apiv1.Storage, project, workspace string) *DevStorage {
	if s == nil {
		return nil
	}
	return &DevStorage{
		APIVersion: DevAPIVersion,
		Kind:       DevStorageKind,
		Metadata:   devMetadataToWire(s.Metadata, project, workspace),
		Spec:       toDevStorageSpec(s.Spec),
		Status:     DevStorageStatus{},
	}
}

func toDevStorageSpec(s apiv1.StorageSpec) DevStorageSpec {
	out := DevStorageSpec{
		Storage: DevResourceRef{
			Name:          s.Storage.Name,
			SystemCatalog: s.Storage.SystemCatalog,
		},
		Type:                      s.Type,
		Size:                      s.Size,
		Datacenter:                s.Datacenter,
		AccessPolicy:              s.AccessPolicy,
		ContractTerm:              s.ContractTerm,
		EnableEncryptionAtRest:    s.EnableEncryptionAtRest,
		EnableEncryptionInTransit: s.EnableEncryptionInTransit,
		StorageType:               s.StorageType,
	}
	if s.Sharing != nil {
		out.Sharing = toDevSharing(*s.Sharing)
	}
	return out
}

// FromDevStorage converts wire format to SDK storage.
func FromDevStorage(wire *DevStorage, workspace string) *apiv1.Storage {
	if wire == nil {
		return nil
	}
	return &apiv1.Storage{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindStorage,
		},
		Metadata: devMetadataFromWire(wire.Metadata, workspace),
		Spec:     fromDevStorageSpec(wire.Spec),
		Status:   fromDevStorageStatus(wire.Status),
	}
}

func fromDevStorageSpec(s DevStorageSpec) apiv1.StorageSpec {
	out := apiv1.StorageSpec{
		Storage: apiv1.ResourceRef{
			Name:          s.Storage.Name,
			SystemCatalog: s.Storage.SystemCatalog,
		},
		Type:                      s.Type,
		Size:                      s.Size,
		Datacenter:                s.Datacenter,
		AccessPolicy:              s.AccessPolicy,
		ContractTerm:              s.ContractTerm,
		EnableEncryptionAtRest:    s.EnableEncryptionAtRest,
		EnableEncryptionInTransit: s.EnableEncryptionInTransit,
		StorageType:               s.StorageType,
	}
	if s.Sharing != nil {
		out.Sharing = fromDevSharing(*s.Sharing)
	}
	return out
}

func fromDevStorageStatus(s DevStorageStatus) apiv1.StorageStatus {
	return apiv1.StorageStatus{
		Status: s.Status,
		Reason: s.Reason,
		Action: s.Action,
	}
}

// FromDevStorageList converts a wire list to SDK storage list.
func FromDevStorageList(wire *DevStorageList, workspace string) *apiv1.StorageList {
	if wire == nil {
		return &apiv1.StorageList{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindStorage + "List"},
		}
	}
	out := &apiv1.StorageList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindStorage + "List",
		},
	}
	out.Metadata.Continue = devListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if item := FromDevStorage(&wire.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
