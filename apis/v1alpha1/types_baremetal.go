package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// BaremetalImage describes the deployment image for a baremetal host.
// The same shape is reused as the request body for ReinstallOS.
type BaremetalImage struct {
	// Checksum is the checksum for the image. Required for all formats
	// except "live-iso".
	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	// ChecksumType is the checksum algorithm (md5, sha256, sha512, or "auto").
	ChecksumType string `json:"checksumType,omitempty" yaml:"checksumType,omitempty"`
	// Format is the image format (raw, qcow2, live-iso, ...).
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
	// URL is the location of the image to deploy.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
}

// BaremetalRootDeviceHints narrows the disk used for OS deployment.
// It is also used to describe physical disks inside software RAID volumes.
type BaremetalRootDeviceHints struct {
	DeviceName         string `json:"deviceName,omitempty" yaml:"deviceName,omitempty"`
	HCTL               string `json:"hctl,omitempty" yaml:"hctl,omitempty"`
	MinSizeGigabytes   int64  `json:"minSizeGigabytes,omitempty" yaml:"minSizeGigabytes,omitempty"`
	Model              string `json:"model,omitempty" yaml:"model,omitempty"`
	Rotational         *bool  `json:"rotational,omitempty" yaml:"rotational,omitempty"`
	SerialNumber       string `json:"serialNumber,omitempty" yaml:"serialNumber,omitempty"`
	Vendor             string `json:"vendor,omitempty" yaml:"vendor,omitempty"`
	WWN                string `json:"wwn,omitempty" yaml:"wwn,omitempty"`
	WWNVendorExtension string `json:"wwnVendorExtension,omitempty" yaml:"wwnVendorExtension,omitempty"`
	WWNWithExtension   string `json:"wwnWithExtension,omitempty" yaml:"wwnWithExtension,omitempty"`
}

// BaremetalHardwareRAIDVolumes describes a single hardware RAID volume.
type BaremetalHardwareRAIDVolumes struct {
	Controller            string   `json:"controller,omitempty" yaml:"controller,omitempty"`
	Level                 string   `json:"level,omitempty" yaml:"level,omitempty"`
	Name                  string   `json:"name,omitempty" yaml:"name,omitempty"`
	NumberOfPhysicalDisks int64    `json:"numberOfPhysicalDisks,omitempty" yaml:"numberOfPhysicalDisks,omitempty"`
	PhysicalDisks         []string `json:"physicalDisks,omitempty" yaml:"physicalDisks,omitempty"`
	Rotational            *bool    `json:"rotational,omitempty" yaml:"rotational,omitempty"`
	SizeGibibytes         int64    `json:"sizeGibibytes,omitempty" yaml:"sizeGibibytes,omitempty"`
}

// BaremetalSoftwareRAIDVolumes describes a single software RAID volume.
type BaremetalSoftwareRAIDVolumes struct {
	Level         string                     `json:"level,omitempty" yaml:"level,omitempty"`
	PhysicalDisks []BaremetalRootDeviceHints `json:"physicalDisks,omitempty" yaml:"physicalDisks,omitempty"`
	SizeGibibytes int64                      `json:"sizeGibibytes,omitempty" yaml:"sizeGibibytes,omitempty"`
}

// BaremetalRaid groups hardware and software RAID configuration.
type BaremetalRaid struct {
	HardwareRAIDVolumes []BaremetalHardwareRAIDVolumes `json:"hardwareRAIDVolumes,omitempty" yaml:"hardwareRAIDVolumes,omitempty"`
	SoftwareRAIDVolumes []BaremetalSoftwareRAIDVolumes `json:"softwareRAIDVolumes,omitempty" yaml:"softwareRAIDVolumes,omitempty"`
}

// BaremetalMachineSpec holds the desired state of a baremetal host.
type BaremetalMachineSpec struct {
	// Architecture (e.g. "x86_64", "aarch64") — usually populated by inspection.
	Architecture string `json:"architecture,omitempty" yaml:"architecture,omitempty"`
	// AutomatedCleaningMode skips cleaning when set to "disabled".
	AutomatedCleaningMode string `json:"automatedCleaningMode,omitempty" yaml:"automatedCleaningMode,omitempty"`
	// BaremetalProvisionerName picks the provisioner that owns this host.
	BaremetalProvisionerName string `json:"baremetalProvisionerName,omitempty" yaml:"baremetalProvisionerName,omitempty"`
	// BootMode is UEFI, Legacy, or UEFISecureBoot. Defaults to UEFI.
	BootMode string `json:"bootMode,omitempty" yaml:"bootMode,omitempty"`
	// Datacenter is the inventory datacenter the host lives in.
	Datacenter string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	// DeviceID is the inventory device identifier.
	DeviceID string `json:"deviceId,omitempty" yaml:"deviceId,omitempty"`
	// Hostname assigned to the baremetal machine.
	Hostname string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	// Image specifies the image to deploy on the host.
	Image *BaremetalImage `json:"image,omitempty" yaml:"image,omitempty"`
	// MACAddress of the primary NIC used for provisioning.
	MACAddress string `json:"macAddress,omitempty" yaml:"macAddress,omitempty"`
	// Online toggles the desired power state when the host is in a stable state.
	Online *bool `json:"online,omitempty" yaml:"online,omitempty"`
	// Raid holds hardware/software RAID configuration.
	Raid *BaremetalRaid `json:"raid,omitempty" yaml:"raid,omitempty"`
	// RootDeviceHints narrows the disk used for OS deployment.
	RootDeviceHints *BaremetalRootDeviceHints `json:"rootDeviceHints,omitempty" yaml:"rootDeviceHints,omitempty"`
	// SSHKey injected for first-boot login. (Field name on the wire is "sshKey".)
	SSHKey string `json:"sshKey,omitempty" yaml:"sshKey,omitempty"`
	// SystemUserData interpreted by first-boot software (cloud-init).
	SystemUserData string `json:"systemUserData,omitempty" yaml:"systemUserData,omitempty"`
	// UserData interpreted by first-boot software (cloud-init).
	UserData string `json:"userData,omitempty" yaml:"userData,omitempty"`
}

// BaremetalMachineCondition is a single condition reported by the platform.
type BaremetalMachineCondition struct {
	LastUpdated string `json:"lastUpdated,omitempty" yaml:"lastUpdated,omitempty"`
	Reason      string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Status      string `json:"status,omitempty" yaml:"status,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
}

// BaremetalMachineStatus holds observed conditions on the baremetal host.
type BaremetalMachineStatus struct {
	Conditions []BaremetalMachineCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// BaremetalMachine is a managed baremetal host on infra.k8smgmt.io/v3.
type BaremetalMachine struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta             `json:"metadata" yaml:"metadata"`
	Spec     BaremetalMachineSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   BaremetalMachineStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (b *BaremetalMachine) GetAPIVersion() string { return b.APIVersion }
func (b *BaremetalMachine) GetKind() string       { return b.Kind }
func (b *BaremetalMachine) GetName() string       { return b.Metadata.Name }
func (b *BaremetalMachine) GetProject() string    { return b.Metadata.Project }
func (b *BaremetalMachine) GetWorkspace() string  { return b.Metadata.Workspace }
func (b *BaremetalMachine) SetProject(val string) { b.Metadata.Project = val }
func (b *BaremetalMachine) SetWorkspace(val string) {
	b.Metadata.Workspace = val
}
func (b *BaremetalMachine) DeepCopyObject() runtime.Object {
	cp := *b
	cp.Metadata = copyObjectMeta(b.Metadata)
	cp.Spec = copyBaremetalMachineSpec(b.Spec)
	cp.Status = copyBaremetalMachineStatus(b.Status)
	return &cp
}

func copyBaremetalMachineSpec(s BaremetalMachineSpec) BaremetalMachineSpec {
	cp := s
	if s.Image != nil {
		img := *s.Image
		cp.Image = &img
	}
	if s.Online != nil {
		v := *s.Online
		cp.Online = &v
	}
	cp.Raid = copyBaremetalRaid(s.Raid)
	cp.RootDeviceHints = copyBaremetalRootDeviceHints(s.RootDeviceHints)
	return cp
}

func copyBaremetalRaid(r *BaremetalRaid) *BaremetalRaid {
	if r == nil {
		return nil
	}
	out := &BaremetalRaid{}
	if len(r.HardwareRAIDVolumes) > 0 {
		out.HardwareRAIDVolumes = make([]BaremetalHardwareRAIDVolumes, len(r.HardwareRAIDVolumes))
		for i, v := range r.HardwareRAIDVolumes {
			cp := v
			if len(v.PhysicalDisks) > 0 {
				cp.PhysicalDisks = append([]string(nil), v.PhysicalDisks...)
			}
			if v.Rotational != nil {
				rv := *v.Rotational
				cp.Rotational = &rv
			}
			out.HardwareRAIDVolumes[i] = cp
		}
	}
	if len(r.SoftwareRAIDVolumes) > 0 {
		out.SoftwareRAIDVolumes = make([]BaremetalSoftwareRAIDVolumes, len(r.SoftwareRAIDVolumes))
		for i, v := range r.SoftwareRAIDVolumes {
			cp := v
			if len(v.PhysicalDisks) > 0 {
				cp.PhysicalDisks = make([]BaremetalRootDeviceHints, len(v.PhysicalDisks))
				for j, h := range v.PhysicalDisks {
					if hc := copyBaremetalRootDeviceHints(&h); hc != nil {
						cp.PhysicalDisks[j] = *hc
					}
				}
			}
			out.SoftwareRAIDVolumes[i] = cp
		}
	}
	return out
}

func copyBaremetalRootDeviceHints(h *BaremetalRootDeviceHints) *BaremetalRootDeviceHints {
	if h == nil {
		return nil
	}
	cp := *h
	if h.Rotational != nil {
		v := *h.Rotational
		cp.Rotational = &v
	}
	return &cp
}

func copyBaremetalMachineStatus(s BaremetalMachineStatus) BaremetalMachineStatus {
	cp := s
	if len(s.Conditions) > 0 {
		cp.Conditions = append([]BaremetalMachineCondition(nil), s.Conditions...)
	}
	return cp
}

// BaremetalMachineList is a list of BaremetalMachine resources.
type BaremetalMachineList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta           `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []BaremetalMachine `json:"items" yaml:"items"`
}

func (l *BaremetalMachineList) GetAPIVersion() string { return l.APIVersion }
func (l *BaremetalMachineList) GetKind() string       { return l.Kind }
func (l *BaremetalMachineList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *BaremetalMachineList) SetItems(items []runtime.Object) {
	l.Items = make([]BaremetalMachine, len(items))
	for i, item := range items {
		if b, ok := item.(*BaremetalMachine); ok {
			l.Items[i] = *b
		}
	}
}

// BaremetalMachineData is the free-form payload returned by GET .../status.
// Fields carries platform-provided runtime information (hardware, network).
type BaremetalMachineData struct {
	Fields map[string]interface{} `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// BaremetalMachineInfo is the response of the GET .../{name}/status endpoint.
// It is intentionally distinct from BaremetalMachineStatus (which lives inside
// the resource and only carries Conditions).
type BaremetalMachineInfo struct {
	Data BaremetalMachineData `json:"data,omitempty" yaml:"data,omitempty"`
}

// BaremetalConsoleSessionRequest is the body of POST .../{name}/consoleSessions.
type BaremetalConsoleSessionRequest struct {
	// ComputeID identifies the compute instance the SOL console attaches to.
	ComputeID string `json:"computeId" yaml:"computeId"`
}

// BaremetalConsoleSession is the response of POST .../{name}/consoleSessions.
type BaremetalConsoleSession struct {
	AgentSessionID string `json:"agentSessionId,omitempty" yaml:"agentSessionId,omitempty"`
	ConsoleURL     string `json:"consoleUrl,omitempty" yaml:"consoleUrl,omitempty"`
	SessionID      string `json:"sessionId,omitempty" yaml:"sessionId,omitempty"`
}
