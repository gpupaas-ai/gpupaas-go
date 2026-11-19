package v1alpha1

import (
	"encoding/json"
	"strings"

	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// First-class virtual machines, served by the platform under dev.envmgmt.io/v1. A VMInstance is
// created from a VMProfile (the VM SKU) and the platform provisions the VM on a host of the
// profile's datacenter. The contract: spec.type names the SKU, the size must match one preset row
// of the SKU, and the field names are snake_case on the wire.

// KindVMInstance and KindVMProfile are the wire kinds of a first-class VM and its SKU.
const (
	KindVMInstance = "VirtualMachine"
	KindVMProfile  = "VmProfile"
)

// ---------------------------------------------------------------------------
// VMInstance
// ---------------------------------------------------------------------------

// VMInstanceSpec holds the desired state of a first-class VM. Fields the SDK does not model are
// kept in Extra and sent back on update.
type VMInstanceSpec struct {
	// Type is the VmProfile (SKU) name. It cannot change after create.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// CPUCount and Memory must together match one preset row of the SKU; GPUCount is required
	// when the SKU has GPUs and must match a GPU preset row. They are strings on the wire.
	CPUCount string `json:"cpu_count,omitempty" yaml:"cpu_count,omitempty"`
	Memory   string `json:"memory,omitempty" yaml:"memory,omitempty"`
	GPUCount string `json:"gpu_count,omitempty" yaml:"gpu_count,omitempty"`
	// GPUModel and GPUVendor are inherited from the SKU when unset.
	GPUModel  string `json:"gpu_model,omitempty" yaml:"gpu_model,omitempty"`
	GPUVendor string `json:"gpu_vendor,omitempty" yaml:"gpu_vendor,omitempty"`
	// Image is the operating system image, one of the SKU's images.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`
	// SSHKey names an SSH key of the instance's workspace; GuestPassword is write-only and never
	// returned. At least one of them is required on create.
	SSHKey        string `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"`
	GuestPassword string `json:"guest_password,omitempty" yaml:"guest_password,omitempty"`
	Datacenter    string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	VPC           string `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet        string `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	// SecurityGroup names a security group of the workspace (one, unlike a compute instance).
	SecurityGroup string `json:"security_group,omitempty" yaml:"security_group,omitempty"`
	// AssignPublicIP is honored only when the SKU's public IP mode is Optional.
	AssignPublicIP bool `json:"assign_public_ip,omitempty" yaml:"assign_public_ip,omitempty"`
	// SharedStorage names a shared filesystem storage to mount.
	SharedStorage string `json:"shared_storage,omitempty" yaml:"shared_storage,omitempty"`
	// UserData is a cloud-init script as plain text.
	UserData string `json:"user_data,omitempty" yaml:"user_data,omitempty"`
	// BootDiskSize is in GB and must lie within the SKU's boot disk range.
	BootDiskSize int32           `json:"boot_disk_size,omitempty" yaml:"boot_disk_size,omitempty"`
	Sharing      *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`

	// VMID is the device id of the VM in inventory, set by the platform.
	VMID string `json:"vm_id,omitempty" yaml:"vm_id,omitempty"`

	// Extra holds spec fields this SDK does not model.
	Extra map[string]json.RawMessage `json:"-" yaml:"-"`
}

type vmInstanceSpecAlias VMInstanceSpec

// MarshalJSON writes the modeled fields plus Extra.
func (s VMInstanceSpec) MarshalJSON() ([]byte, error) {
	return marshalWithExtras(vmInstanceSpecAlias(s), s.Extra)
}

// UnmarshalJSON reads the modeled fields and keeps the rest in Extra.
func (s *VMInstanceSpec) UnmarshalJSON(b []byte) error {
	var a vmInstanceSpecAlias
	extra, err := unmarshalWithExtras(b, &a)
	if err != nil {
		return err
	}
	*s = VMInstanceSpec(a)
	s.Extra = extra
	return nil
}

// VMInstanceOutput is what provisioning reports about the VM.
type VMInstanceOutput struct {
	HostName   string `json:"host_name,omitempty" yaml:"host_name,omitempty"`
	InstanceID string `json:"instance_id,omitempty" yaml:"instance_id,omitempty"`
	OSName     string `json:"os_name,omitempty" yaml:"os_name,omitempty"`
	PrivateIP  string `json:"private_ip,omitempty" yaml:"private_ip,omitempty"`
	PublicIP   string `json:"public_ip,omitempty" yaml:"public_ip,omitempty"`
	ServerHost string `json:"server_host,omitempty" yaml:"server_host,omitempty"`
	UserName   string `json:"user_name,omitempty" yaml:"user_name,omitempty"`
}

// VMInstanceStatus holds the observed state. Action is the last or in-flight operation (deploy,
// start, stop, reboot, destroy, ...).
type VMInstanceStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	// ReasonDetail is the workflow error behind a failed Reason, as the platform reports it on the
	// instance itself.
	ReasonDetail string            `json:"reasonDetail,omitempty" yaml:"reasonDetail,omitempty"`
	Action       string            `json:"action,omitempty" yaml:"action,omitempty"`
	Output       *VMInstanceOutput `json:"output,omitempty" yaml:"output,omitempty"`
}

type vmInstanceStatusAlias VMInstanceStatus

// UnmarshalJSON reads the status. The VM wire is snake_case everywhere else, so a failure detail is accepted
// as reasonDetail (what bare metal reports) or as reason_detail.
func (s *VMInstanceStatus) UnmarshalJSON(b []byte) error {
	aux := struct {
		vmInstanceStatusAlias
		ReasonDetailSnake string `json:"reason_detail"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*s = VMInstanceStatus(aux.vmInstanceStatusAlias)
	if s.ReasonDetail == "" {
		s.ReasonDetail = aux.ReasonDetailSnake
	}
	return nil
}

// VM phases (the status string, lowercase).
const (
	VMPhasePending       = "pending"
	VMPhaseInProgress    = "in_progress"
	VMPhaseRunning       = "running"
	VMPhaseStopped       = "stopped"
	VMPhaseFailed        = "failed"
	VMPhaseDeleting      = "deleting"
	VMPhaseDeletePending = "delete_pending"
	VMPhaseDestroyFailed = "destroyfailed"
)

// Power states derived from the phase.
const (
	VMPowerRunning = "running"
	VMPowerStopped = "stopped"
	VMPowerUnknown = "unknown"
)

// Phase returns the status normalized to lowercase without a "status_" prefix.
func (s VMInstanceStatus) Phase() string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s.Status)), "status_")
}

// Running reports whether the VM is up.
func (s VMInstanceStatus) Running() bool { return s.Phase() == VMPhaseRunning }

// InProgress reports provisioning or another operation in flight (an empty status counts: a row the
// platform has stored but not started yet).
func (s VMInstanceStatus) InProgress() bool {
	switch s.Phase() {
	case "", VMPhasePending, VMPhaseInProgress:
		return true
	}
	return false
}

// Failed reports a failure: provisioning failed, a delete failed, or the delete was parked for the
// operator (end users cannot delete it again).
func (s VMInstanceStatus) Failed() bool {
	switch s.Phase() {
	case VMPhaseFailed, VMPhaseDestroyFailed, VMPhaseDeletePending:
		return true
	}
	return false
}

// Deleting reports that teardown is in flight.
func (s VMInstanceStatus) Deleting() bool { return s.Phase() == VMPhaseDeleting }

// PowerState is "running", "stopped" or "unknown".
func (s VMInstanceStatus) PowerState() string {
	switch s.Phase() {
	case VMPhaseRunning:
		return VMPowerRunning
	case VMPhaseStopped:
		return VMPowerStopped
	}
	return VMPowerUnknown
}

// VMInstance is a first-class virtual machine.
type VMInstance struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta       `json:"metadata" yaml:"metadata"`
	Spec     VMInstanceSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   VMInstanceStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (v *VMInstance) GetAPIVersion() string { return v.APIVersion }
func (v *VMInstance) GetKind() string       { return v.Kind }
func (v *VMInstance) GetName() string       { return v.Metadata.Name }
func (v *VMInstance) GetProject() string    { return v.Metadata.Project }
func (v *VMInstance) GetWorkspace() string  { return v.Metadata.Workspace }
func (v *VMInstance) SetProject(p string)   { v.Metadata.Project = p }
func (v *VMInstance) SetWorkspace(w string) { v.Metadata.Workspace = w }
func (v *VMInstance) DeepCopyObject() runtime.Object {
	cp := *v
	cp.Metadata = copyObjectMeta(v.Metadata)
	cp.Spec = jsonCopy(v.Spec)
	cp.Status = jsonCopy(v.Status)
	return &cp
}

// VMInstanceList is a list of VMInstance resources.
type VMInstanceList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []VMInstance `json:"items" yaml:"items"`
}

func (l *VMInstanceList) GetAPIVersion() string { return l.APIVersion }
func (l *VMInstanceList) GetKind() string       { return l.Kind }
func (l *VMInstanceList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *VMInstanceList) SetItems(items []runtime.Object) {
	l.Items = make([]VMInstance, len(items))
	for i, item := range items {
		if v, ok := item.(*VMInstance); ok {
			l.Items[i] = *v
		}
	}
}

// ---------------------------------------------------------------------------
// VMProfile (the VM SKU, read-only)
// ---------------------------------------------------------------------------

// VMPreset is one size a SKU offers: a VM must be created with exactly one of these rows.
type VMPreset struct {
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
	CPU    string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
	// GPU is set on the rows of a GPU SKU's gpu_preset.
	GPU string `json:"gpu,omitempty" yaml:"gpu,omitempty"`
}

// VMBootDisk is the boot disk range of a SKU, in Unit (GB) with sizes as strings on the wire.
type VMBootDisk struct {
	MinSize   string `json:"min_size,omitempty" yaml:"min_size,omitempty"`
	MaxSize   string `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	Increment string `json:"increment,omitempty" yaml:"increment,omitempty"`
	Unit      string `json:"unit,omitempty" yaml:"unit,omitempty"`
}

// VMProfileSpec describes a VM SKU.
type VMProfileSpec struct {
	Image        []string `json:"image,omitempty" yaml:"image,omitempty"`
	ImageDefault string   `json:"image_default,omitempty" yaml:"image_default,omitempty"`
	Datacenter   []string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	// GPUsEnabled selects gpu_preset (true) or cpu_preset (false) as the list of sizes.
	GPUsEnabled bool        `json:"gpus_enabled,omitempty" yaml:"gpus_enabled,omitempty"`
	CPUPreset   []VMPreset  `json:"cpu_preset,omitempty" yaml:"cpu_preset,omitempty"`
	GPUPreset   []VMPreset  `json:"gpu_preset,omitempty" yaml:"gpu_preset,omitempty"`
	GPUVendor   string      `json:"gpu_vendor,omitempty" yaml:"gpu_vendor,omitempty"`
	GPUModel    string      `json:"gpu_model,omitempty" yaml:"gpu_model,omitempty"`
	GPUName     string      `json:"gpu_name,omitempty" yaml:"gpu_name,omitempty"`
	BootDisk    *VMBootDisk `json:"boot_disk,omitempty" yaml:"boot_disk,omitempty"`
	// PublicIP is the policy: None, Always or Optional.
	PublicIP     string `json:"public_ip,omitempty" yaml:"public_ip,omitempty"`
	InstanceType string `json:"instance_type,omitempty" yaml:"instance_type,omitempty"`
	TemplateName string `json:"template_name,omitempty" yaml:"template_name,omitempty"`
	// TemplateVersion is the environment template version.
	TemplateVersion         string          `json:"template_version,omitempty" yaml:"template_version,omitempty"`
	SharedFilesystemEnabled bool            `json:"shared_filesystem_enabled,omitempty" yaml:"shared_filesystem_enabled,omitempty"`
	AdditionalBlockEnabled  bool            `json:"additional_block_enabled,omitempty" yaml:"additional_block_enabled,omitempty"`
	EphemeralStorageEnabled bool            `json:"ephemeral_storage_enabled,omitempty" yaml:"ephemeral_storage_enabled,omitempty"`
	Sharing                 *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`
}

// VMProfileGlobalSettings holds what the platform computed for the SKU.
type VMProfileGlobalSettings struct {
	// Agents are the orchestration agents that run the SKU's workflows. When none is healthy,
	// provisioning fails with "no healthy agents found".
	Agents []string `json:"agents,omitempty" yaml:"agents,omitempty"`
}

// VMProfileStatus holds observed SKU state.
type VMProfileStatus struct {
	Status         string                   `json:"status,omitempty" yaml:"status,omitempty"`
	GlobalSettings *VMProfileGlobalSettings `json:"globalSettings,omitempty" yaml:"globalSettings,omitempty"`
}

// VMProfile is a VM SKU. Administrators manage SKUs; the SDK reads them so callers can learn which
// images, sizes, datacenters and options a VM may use.
type VMProfile struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta      `json:"metadata" yaml:"metadata"`
	Spec     VMProfileSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   VMProfileStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (p *VMProfile) GetAPIVersion() string { return p.APIVersion }
func (p *VMProfile) GetKind() string       { return p.Kind }
func (p *VMProfile) GetName() string       { return p.Metadata.Name }
func (p *VMProfile) GetProject() string    { return p.Metadata.Project }
func (p *VMProfile) GetWorkspace() string  { return p.Metadata.Workspace }
func (p *VMProfile) SetProject(v string)   { p.Metadata.Project = v }
func (p *VMProfile) SetWorkspace(v string) { p.Metadata.Workspace = v }
func (p *VMProfile) DeepCopyObject() runtime.Object {
	cp := *p
	cp.Metadata = copyObjectMeta(p.Metadata)
	cp.Spec = jsonCopy(p.Spec)
	cp.Status = jsonCopy(p.Status)
	return &cp
}

// Sizes returns the preset rows a VM of this SKU may be created with: the GPU presets for a GPU
// SKU, the CPU presets otherwise.
func (s VMProfileSpec) Sizes() []VMPreset {
	if s.GPUsEnabled {
		return s.GPUPreset
	}
	return s.CPUPreset
}

// VMProfileList is a list of VMProfile resources.
type VMProfileList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []VMProfile `json:"items" yaml:"items"`
}

func (l *VMProfileList) GetAPIVersion() string { return l.APIVersion }
func (l *VMProfileList) GetKind() string       { return l.Kind }
func (l *VMProfileList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *VMProfileList) SetItems(items []runtime.Object) {
	l.Items = make([]VMProfile, len(items))
	for i, item := range items {
		if p, ok := item.(*VMProfile); ok {
			l.Items[i] = *p
		}
	}
}
