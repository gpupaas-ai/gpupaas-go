package v1alpha1

import (
	"strings"

	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// Storage families. Both are one Storage kind on the platform; the profile (SKU) named in Spec.Type
// decides the family, and a list takes it as a filter (ListOptions.ServiceType).
const (
	StorageServiceTypeHighSpeed = "high-speed-storage"
	StorageServiceTypeObject    = "object-storage"
)

// StorageSpec holds desired storage state. Type names the storage profile (SKU) a platform administrator
// published; everything else is the user's request against it.
type StorageSpec struct {
	// Type is the storage profile (SKU) name.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Size is the requested capacity as a number in the unit the profile defines (GB unless the profile
	// says GiB). A profile with a fixed volume ignores it.
	Size       string `json:"size,omitempty" yaml:"size,omitempty"`
	Datacenter string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	// StorageType classifies the volume: block, file or object. The platform may fill it from the profile.
	StorageType  string `json:"storageType,omitempty" yaml:"storageType,omitempty"`
	AccessPolicy string `json:"accessPolicy,omitempty" yaml:"accessPolicy,omitempty"`
	ContractTerm string `json:"contractTerm,omitempty" yaml:"contractTerm,omitempty"`
	// EnableEncryptionAtRest and EnableEncryptionInTransit are "true" or "false" (strings, as the platform
	// carries them); empty leaves them to the profile.
	EnableEncryptionAtRest    string          `json:"enableEncryptionAtRest,omitempty" yaml:"enableEncryptionAtRest,omitempty"`
	EnableEncryptionInTransit string          `json:"enableEncryptionInTransit,omitempty" yaml:"enableEncryptionInTransit,omitempty"`
	Sharing                   *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`
}

// StorageStatus holds observed storage state.
type StorageStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Action string `json:"action,omitempty" yaml:"action,omitempty"`
	// Output describes the provisioned volume: a filesystem's servers and mount credentials, an object
	// store's bucket and S3 keys, the volume size in bytes. The platform returns the credentials in plain
	// text, so treat the whole map as secret. Every value is a string (numbers are formatted without an
	// exponent).
	Output map[string]string `json:"output,omitempty" yaml:"output,omitempty"`
}

// Storage phases (the status string, lowercase). They are the platform's common vocabulary.
const (
	StoragePhasePending       = "pending"
	StoragePhaseInProgress    = "in_progress"
	StoragePhaseSuccess       = "success"
	StoragePhaseFailed        = "failed"
	StoragePhaseDeleting      = "deleting"
	StoragePhaseDeletePending = "delete_pending"
	StoragePhaseDestroyFailed = "destroyfailed"
)

// Phase returns the status normalized to lowercase without a "status_" prefix.
func (s StorageStatus) Phase() string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s.Status)), "status_")
}

// Ready reports that the volume is provisioned and usable.
func (s StorageStatus) Ready() bool { return s.Phase() == StoragePhaseSuccess }

// InProgress reports provisioning or another operation in flight (an empty status counts: a record the
// platform has stored but not started yet).
func (s StorageStatus) InProgress() bool {
	switch s.Phase() {
	case "", StoragePhasePending, StoragePhaseInProgress:
		return true
	}
	return false
}

// Failed reports a failure: provisioning failed, a delete failed, or the delete was parked for the
// operator (end users cannot delete it again).
func (s StorageStatus) Failed() bool {
	switch s.Phase() {
	case StoragePhaseFailed, StoragePhaseDestroyFailed, StoragePhaseDeletePending:
		return true
	}
	return false
}

// Deleting reports that teardown is in flight.
func (s StorageStatus) Deleting() bool { return s.Phase() == StoragePhaseDeleting }

// Storage is a dev storage resource (a high-speed or object storage volume), at project or workspace scope.
type Storage struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta    `json:"metadata" yaml:"metadata"`
	Spec     StorageSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   StorageStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (s *Storage) GetAPIVersion() string { return s.APIVersion }
func (s *Storage) GetKind() string       { return s.Kind }
func (s *Storage) GetName() string       { return s.Metadata.Name }
func (s *Storage) GetProject() string    { return s.Metadata.Project }
func (s *Storage) GetWorkspace() string  { return s.Metadata.Workspace }
func (s *Storage) SetProject(val string) { s.Metadata.Project = val }
func (s *Storage) SetWorkspace(val string) {
	s.Metadata.Workspace = val
}
func (s *Storage) DeepCopyObject() runtime.Object {
	cp := *s
	cp.Metadata = copyObjectMeta(s.Metadata)
	cp.Spec.Sharing = copyDevSharingSpec(s.Spec.Sharing)
	if s.Status.Output != nil {
		cp.Status.Output = make(map[string]string, len(s.Status.Output))
		for k, v := range s.Status.Output {
			cp.Status.Output[k] = v
		}
	}
	return &cp
}

// StorageList is a list of Storage resources.
type StorageList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []Storage `json:"items" yaml:"items"`
}

func (l *StorageList) GetAPIVersion() string { return l.APIVersion }
func (l *StorageList) GetKind() string       { return l.Kind }
func (l *StorageList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *StorageList) SetItems(items []runtime.Object) {
	l.Items = make([]Storage, len(items))
	for i, item := range items {
		if s, ok := item.(*Storage); ok {
			l.Items[i] = *s
		}
	}
}
