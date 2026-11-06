package v1alpha1

import (
	"encoding/json"
	"strings"

	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// First-class bare metal, served by the platform under dev.envmgmt.io/v1. A BareMetal instance is
// created from a BmProfile (the bare-metal SKU) and the platform provisions a machine from
// inventory.

// KindBareMetal and KindBmProfile are the kinds of a first-class bare-metal instance and its SKU.
const (
	KindBareMetal = "BareMetal"
	KindBmProfile = "BmProfile"
)

// ---------------------------------------------------------------------------
// BareMetal
// ---------------------------------------------------------------------------

// BareMetalSpec holds the desired state of a bare-metal instance. Fields the SDK does not model
// are kept in Extra and sent back on update.
type BareMetalSpec struct {
	// Type is the BmProfile (SKU) name. It cannot change after create.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Image is the operating system image (required on create).
	Image string `json:"image,omitempty" yaml:"image,omitempty"`
	// SSHKey names an SSH key of the instance's workspace; UserPassword is write-only and never
	// returned. At least one of them is required on create.
	SSHKey       string `json:"sshKey,omitempty" yaml:"sshKey,omitempty"`
	UserPassword string `json:"userPassword,omitempty" yaml:"userPassword,omitempty"`
	Datacenter   string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	VPC          string `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet       string `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	// SecurityGroup names a security group of the workspace.
	SecurityGroup  string `json:"securityGroup,omitempty" yaml:"securityGroup,omitempty"`
	AssignPublicIP bool   `json:"assignPublicIp,omitempty" yaml:"assignPublicIp,omitempty"`
	// HSSStorage names a high-speed (shared filesystem) storage; ObjectStorage an object storage.
	HSSStorage    string `json:"hssStorage,omitempty" yaml:"hssStorage,omitempty"`
	ObjectStorage string `json:"objectStorage,omitempty" yaml:"objectStorage,omitempty"`
	// UserData is a cloud-init script as plain text; the service encodes it.
	UserData string `json:"userData,omitempty" yaml:"userData,omitempty"`
	// ContractTerm is the contract length in months and must be one the profile offers.
	ContractTerm int32 `json:"contractTerm,omitempty" yaml:"contractTerm,omitempty"`
	// GPUName is inherited from the profile when unset.
	GPUName string          `json:"gpuName,omitempty" yaml:"gpuName,omitempty"`
	Sharing *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`

	// ComputeInstanceID and ProvisionerType are set by the platform ("bcm" or "non_bcm").
	ComputeInstanceID string `json:"computeInstanceId,omitempty" yaml:"computeInstanceId,omitempty"`
	ProvisionerType   string `json:"provisionerType,omitempty" yaml:"provisionerType,omitempty"`

	// Extra holds spec fields this SDK does not model.
	Extra map[string]json.RawMessage `json:"-" yaml:"-"`
}

type bareMetalSpecAlias BareMetalSpec

// MarshalJSON writes the modeled fields plus Extra.
func (s BareMetalSpec) MarshalJSON() ([]byte, error) {
	return marshalWithExtras(bareMetalSpecAlias(s), s.Extra)
}

// UnmarshalJSON reads the modeled fields and keeps the rest in Extra.
func (s *BareMetalSpec) UnmarshalJSON(b []byte) error {
	var a bareMetalSpecAlias
	extra, err := unmarshalWithExtras(b, &a)
	if err != nil {
		return err
	}
	*s = BareMetalSpec(a)
	s.Extra = extra
	return nil
}

// BareMetalOutput is what provisioning reports about the machine.
type BareMetalOutput struct {
	HostName  string `json:"hostName,omitempty" yaml:"hostName,omitempty"`
	PrivateIP string `json:"privateIp,omitempty" yaml:"privateIp,omitempty"`
	PublicIP  string `json:"publicIp,omitempty" yaml:"publicIp,omitempty"`
	UserName  string `json:"userName,omitempty" yaml:"userName,omitempty"`
	Platform  string `json:"platform,omitempty" yaml:"platform,omitempty"`
	GPUs      string `json:"gpus,omitempty" yaml:"gpus,omitempty"`
	VCPUs     string `json:"vcpus,omitempty" yaml:"vcpus,omitempty"`
	MemoryGiB string `json:"memoryGib,omitempty" yaml:"memoryGib,omitempty"`
	Network   string `json:"network,omitempty" yaml:"network,omitempty"`
}

// BareMetalStatus holds the observed state. Action is the last or in-flight operation (deploy,
// start, stop, reboot, destroy, ...).
type BareMetalStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	// ReasonDetail is the workflow error behind a failed Reason ("Couldn't provision instance X due
	// to an internal error"), as the platform reports it on the instance itself.
	ReasonDetail string           `json:"reasonDetail,omitempty" yaml:"reasonDetail,omitempty"`
	Action       string           `json:"action,omitempty" yaml:"action,omitempty"`
	Output       *BareMetalOutput `json:"output,omitempty" yaml:"output,omitempty"`
}

// Bare-metal phases (the status string, lowercase).
const (
	BareMetalPhasePending       = "pending"
	BareMetalPhaseInProgress    = "in_progress"
	BareMetalPhaseRunning       = "running"
	BareMetalPhaseStopped       = "stopped"
	BareMetalPhaseFailed        = "failed"
	BareMetalPhaseDeleting      = "deleting"
	BareMetalPhaseDeletePending = "delete_pending"
	BareMetalPhaseDestroyFailed = "destroyfailed"
)

// Power states derived from the phase.
const (
	BareMetalPowerRunning = "running"
	BareMetalPowerStopped = "stopped"
	BareMetalPowerUnknown = "unknown"
)

// Phase returns the status normalized to lowercase without a "status_" prefix.
func (s BareMetalStatus) Phase() string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s.Status)), "status_")
}

// Running reports whether the machine is up.
func (s BareMetalStatus) Running() bool { return s.Phase() == BareMetalPhaseRunning }

// InProgress reports provisioning or another operation in flight (an empty status counts: a row the
// platform has stored but not started yet).
func (s BareMetalStatus) InProgress() bool {
	switch s.Phase() {
	case "", BareMetalPhasePending, BareMetalPhaseInProgress:
		return true
	}
	return false
}

// Failed reports a failure: provisioning failed, a delete failed, or the delete was parked for
// the operator (end users cannot delete it again).
func (s BareMetalStatus) Failed() bool {
	switch s.Phase() {
	case BareMetalPhaseFailed, BareMetalPhaseDestroyFailed, BareMetalPhaseDeletePending:
		return true
	}
	return false
}

// Deleting reports that teardown is in flight.
func (s BareMetalStatus) Deleting() bool { return s.Phase() == BareMetalPhaseDeleting }

// PowerState is "running", "stopped" or "unknown".
func (s BareMetalStatus) PowerState() string {
	switch s.Phase() {
	case BareMetalPhaseRunning:
		return BareMetalPowerRunning
	case BareMetalPhaseStopped:
		return BareMetalPowerStopped
	}
	return BareMetalPowerUnknown
}

// BareMetal is a first-class bare-metal instance.
type BareMetal struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta      `json:"metadata" yaml:"metadata"`
	Spec     BareMetalSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   BareMetalStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (b *BareMetal) GetAPIVersion() string { return b.APIVersion }
func (b *BareMetal) GetKind() string       { return b.Kind }
func (b *BareMetal) GetName() string       { return b.Metadata.Name }
func (b *BareMetal) GetProject() string    { return b.Metadata.Project }
func (b *BareMetal) GetWorkspace() string  { return b.Metadata.Workspace }
func (b *BareMetal) SetProject(v string)   { b.Metadata.Project = v }
func (b *BareMetal) SetWorkspace(v string) { b.Metadata.Workspace = v }
func (b *BareMetal) DeepCopyObject() runtime.Object {
	cp := *b
	cp.Metadata = copyObjectMeta(b.Metadata)
	cp.Spec = jsonCopy(b.Spec)
	cp.Status = jsonCopy(b.Status)
	return &cp
}

// BareMetalList is a list of BareMetal resources.
type BareMetalList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []BareMetal `json:"items" yaml:"items"`
}

func (l *BareMetalList) GetAPIVersion() string { return l.APIVersion }
func (l *BareMetalList) GetKind() string       { return l.Kind }
func (l *BareMetalList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *BareMetalList) SetItems(items []runtime.Object) {
	l.Items = make([]BareMetal, len(items))
	for i, item := range items {
		if b, ok := item.(*BareMetal); ok {
			l.Items[i] = *b
		}
	}
}

// ---------------------------------------------------------------------------
// BmProfile (the bare-metal SKU, read-only)
// ---------------------------------------------------------------------------

// BmInventoryTag is one inventory match: devices tagged key=value are candidates for the SKU.
type BmInventoryTag struct {
	Key   string `json:"key,omitempty" yaml:"key,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

// BmProfileSpec describes a bare-metal SKU.
type BmProfileSpec struct {
	// ProvisionerType is "bcm" or "non_bcm" (Ironic).
	ProvisionerType string   `json:"provisionerType,omitempty" yaml:"provisionerType,omitempty"`
	Image           []string `json:"image,omitempty" yaml:"image,omitempty"`
	ImageDefault    string   `json:"imageDefault,omitempty" yaml:"imageDefault,omitempty"`
	Datacenter      []string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	VPC             string   `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet          string   `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	// AssignPublicIP is the policy: Disabled, Enabled or CustomerSelectable.
	AssignPublicIP   string           `json:"assignPublicIp,omitempty" yaml:"assignPublicIp,omitempty"`
	MRCContractTerm  []int32          `json:"mrcContractTerm,omitempty" yaml:"mrcContractTerm,omitempty"`
	GPUName          string           `json:"gpuName,omitempty" yaml:"gpuName,omitempty"`
	UseCase          string           `json:"useCase,omitempty" yaml:"useCase,omitempty"`
	InventoryConfig  []BmInventoryTag `json:"inventoryConfig,omitempty" yaml:"inventoryConfig,omitempty"`
	TemplateName     string           `json:"templateName,omitempty" yaml:"templateName,omitempty"`
	TemplateVersion  string           `json:"templateVersion,omitempty" yaml:"templateVersion,omitempty"`
	Sharing          *DevSharingSpec  `json:"sharing,omitempty" yaml:"sharing,omitempty"`
	SharedFilesystem bool             `json:"sharedFilesystemEnabled,omitempty" yaml:"sharedFilesystemEnabled,omitempty"`
	ObjectStorage    bool             `json:"objectStorageEnabled,omitempty" yaml:"objectStorageEnabled,omitempty"`
}

// BmProfileGlobalSettings holds what the platform computed for the SKU.
type BmProfileGlobalSettings struct {
	// Agents are the orchestration agents that run the SKU's workflows. When none is healthy,
	// provisioning fails with "no healthy agents found".
	Agents []string `json:"agents,omitempty" yaml:"agents,omitempty"`
}

// BmProfileStatus holds observed SKU state.
type BmProfileStatus struct {
	Status         string                   `json:"status,omitempty" yaml:"status,omitempty"`
	GlobalSettings *BmProfileGlobalSettings `json:"globalSettings,omitempty" yaml:"globalSettings,omitempty"`
}

// BmProfile is a bare-metal SKU. Administrators manage SKUs; the SDK reads them so callers can learn
// which images, datacenters and options an instance may use.
type BmProfile struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta      `json:"metadata" yaml:"metadata"`
	Spec     BmProfileSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   BmProfileStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (p *BmProfile) GetAPIVersion() string { return p.APIVersion }
func (p *BmProfile) GetKind() string       { return p.Kind }
func (p *BmProfile) GetName() string       { return p.Metadata.Name }
func (p *BmProfile) GetProject() string    { return p.Metadata.Project }
func (p *BmProfile) GetWorkspace() string  { return p.Metadata.Workspace }
func (p *BmProfile) SetProject(v string)   { p.Metadata.Project = v }
func (p *BmProfile) SetWorkspace(v string) { p.Metadata.Workspace = v }
func (p *BmProfile) DeepCopyObject() runtime.Object {
	cp := *p
	cp.Metadata = copyObjectMeta(p.Metadata)
	cp.Spec = jsonCopy(p.Spec)
	cp.Status = jsonCopy(p.Status)
	return &cp
}

// BmProfileList is a list of BmProfile resources.
type BmProfileList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []BmProfile `json:"items" yaml:"items"`
}

func (l *BmProfileList) GetAPIVersion() string { return l.APIVersion }
func (l *BmProfileList) GetKind() string       { return l.Kind }
func (l *BmProfileList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *BmProfileList) SetItems(items []runtime.Object) {
	l.Items = make([]BmProfile, len(items))
	for i, item := range items {
		if p, ok := item.(*BmProfile); ok {
			l.Items[i] = *p
		}
	}
}
