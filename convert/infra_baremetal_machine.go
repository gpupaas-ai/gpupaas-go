package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	InfraBaremetalMachineKind     = "BaremetalMachine"
	InfraBaremetalMachineListKind = "BaremetalMachineList"
)

// InfraProjectScope identifies a project on infra.k8smgmt.io.
// Baremetal resources are project-scoped only (no workspace).
type InfraProjectScope struct {
	Project string
}

// BaremetalMachinePaths returns path builders for baremetal machine endpoints.
func BaremetalMachinePaths(scope InfraProjectScope) (
	collection func() string,
	item func(name string) string,
	subroute func(name, action string) string,
) {
	collection = func() string {
		return fmt.Sprintf(InfraBaremetalMachinesPath, url.PathEscape(scope.Project))
	}
	item = func(name string) string {
		return fmt.Sprintf(InfraBaremetalMachinePath, url.PathEscape(scope.Project), url.PathEscape(name))
	}
	subroute = func(name, action string) string {
		base := fmt.Sprintf(InfraBaremetalMachinePath, url.PathEscape(scope.Project), url.PathEscape(name))
		return base + "/" + action
	}
	return
}

// InfraBaremetalImage is the wire format for the deployment image and the
// ReinstallOS request body. Both endpoints accept the same shape per spec.
type InfraBaremetalImage struct {
	Checksum     string `json:"checksum,omitempty"`
	ChecksumType string `json:"checksumType,omitempty"`
	Format       string `json:"format,omitempty"`
	URL          string `json:"url,omitempty"`
}

// InfraBaremetalRootDeviceHints is the wire format for root device hints.
type InfraBaremetalRootDeviceHints struct {
	DeviceName         string `json:"deviceName,omitempty"`
	HCTL               string `json:"hctl,omitempty"`
	MinSizeGigabytes   int64  `json:"minSizeGigabytes,omitempty"`
	Model              string `json:"model,omitempty"`
	Rotational         *bool  `json:"rotational,omitempty"`
	SerialNumber       string `json:"serialNumber,omitempty"`
	Vendor             string `json:"vendor,omitempty"`
	WWN                string `json:"wwn,omitempty"`
	WWNVendorExtension string `json:"wwnVendorExtension,omitempty"`
	WWNWithExtension   string `json:"wwnWithExtension,omitempty"`
}

// InfraBaremetalHardwareRAIDVolume is the wire format for a hardware RAID volume.
type InfraBaremetalHardwareRAIDVolume struct {
	Controller            string   `json:"controller,omitempty"`
	Level                 string   `json:"level,omitempty"`
	Name                  string   `json:"name,omitempty"`
	NumberOfPhysicalDisks int64    `json:"numberOfPhysicalDisks,omitempty"`
	PhysicalDisks         []string `json:"physicalDisks,omitempty"`
	Rotational            *bool    `json:"rotational,omitempty"`
	SizeGibibytes         int64    `json:"sizeGibibytes,omitempty"`
}

// InfraBaremetalSoftwareRAIDVolume is the wire format for a software RAID volume.
type InfraBaremetalSoftwareRAIDVolume struct {
	Level         string                          `json:"level,omitempty"`
	PhysicalDisks []InfraBaremetalRootDeviceHints `json:"physicalDisks,omitempty"`
	SizeGibibytes int64                           `json:"sizeGibibytes,omitempty"`
}

// InfraBaremetalRaid is the wire format for RAID configuration.
type InfraBaremetalRaid struct {
	HardwareRAIDVolumes []InfraBaremetalHardwareRAIDVolume `json:"hardwareRAIDVolumes,omitempty"`
	SoftwareRAIDVolumes []InfraBaremetalSoftwareRAIDVolume `json:"softwareRAIDVolumes,omitempty"`
}

// InfraBaremetalMachineSpec is the wire format for the desired state.
type InfraBaremetalMachineSpec struct {
	Architecture             string                         `json:"architecture,omitempty"`
	AutomatedCleaningMode    string                         `json:"automatedCleaningMode,omitempty"`
	BaremetalProvisionerName string                         `json:"baremetalProvisionerName,omitempty"`
	BootMode                 string                         `json:"bootMode,omitempty"`
	Datacenter               string                         `json:"datacenter,omitempty"`
	DeviceID                 string                         `json:"deviceId,omitempty"`
	Hostname                 string                         `json:"hostname,omitempty"`
	Image                    *InfraBaremetalImage           `json:"image,omitempty"`
	MACAddress               string                         `json:"macAddress,omitempty"`
	Online                   *bool                          `json:"online,omitempty"`
	Raid                     *InfraBaremetalRaid            `json:"raid,omitempty"`
	RootDeviceHints          *InfraBaremetalRootDeviceHints `json:"rootDeviceHints,omitempty"`
	SSHKey                   string                         `json:"sshKey,omitempty"`
	SystemUserData           string                         `json:"systemUserData,omitempty"`
	UserData                 string                         `json:"userData,omitempty"`
}

// InfraBaremetalMachineCondition is the wire format for an observed condition.
type InfraBaremetalMachineCondition struct {
	LastUpdated string `json:"lastUpdated,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Status      string `json:"status,omitempty"`
	Type        string `json:"type,omitempty"`
}

// InfraBaremetalMachineStatus is the wire format for observed status.
type InfraBaremetalMachineStatus struct {
	Conditions []InfraBaremetalMachineCondition `json:"conditions,omitempty"`
}

// InfraBaremetalMachine is the wire format for a single BaremetalMachine.
type InfraBaremetalMachine struct {
	APIVersion string                      `json:"apiVersion"`
	Kind       string                      `json:"kind"`
	Metadata   InfraMetadata               `json:"metadata"`
	Spec       InfraBaremetalMachineSpec   `json:"spec,omitempty"`
	Status     InfraBaremetalMachineStatus `json:"status,omitempty"`
}

// InfraBaremetalMachineList is the wire format for list responses.
type InfraBaremetalMachineList struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   PaaSListMetadata        `json:"metadata,omitempty"`
	Items      []InfraBaremetalMachine `json:"items"`
}

// InfraBaremetalMachineData is the wire format for the Info "data" envelope.
type InfraBaremetalMachineData struct {
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// InfraBaremetalMachineInfo is the response of GET .../{name}/status.
type InfraBaremetalMachineInfo struct {
	Data InfraBaremetalMachineData `json:"data,omitempty"`
}

// InfraBaremetalConsoleSessionRequest is the wire request body for POST
// .../{name}/consoleSessions (note: snake_case per spec).
type InfraBaremetalConsoleSessionRequest struct {
	ComputeID string `json:"compute_id"`
}

// InfraBaremetalConsoleSession is the wire response of POST
// .../{name}/consoleSessions (note: snake_case per spec).
type InfraBaremetalConsoleSession struct {
	AgentSessionID string `json:"agent_session_id,omitempty"`
	ConsoleURL     string `json:"console_url,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
}

// ----- Converters: SDK -> Wire -----

// ToInfraBaremetalMachine converts a k8s-style BaremetalMachine to the
// infra.k8smgmt.io wire format.
func ToInfraBaremetalMachine(bm *apiv1.BaremetalMachine, project string) *InfraBaremetalMachine {
	if bm == nil {
		return nil
	}
	return &InfraBaremetalMachine{
		APIVersion: InfraAPIVersion,
		Kind:       InfraBaremetalMachineKind,
		Metadata:   infraMetadataToWire(bm.Metadata, project),
		Spec:       toInfraBaremetalMachineSpec(bm.Spec),
	}
}

func toInfraBaremetalMachineSpec(s apiv1.BaremetalMachineSpec) InfraBaremetalMachineSpec {
	out := InfraBaremetalMachineSpec{
		Architecture:             s.Architecture,
		AutomatedCleaningMode:    s.AutomatedCleaningMode,
		BaremetalProvisionerName: s.BaremetalProvisionerName,
		BootMode:                 s.BootMode,
		Datacenter:               s.Datacenter,
		DeviceID:                 s.DeviceID,
		Hostname:                 s.Hostname,
		MACAddress:               s.MACAddress,
		SSHKey:                   s.SSHKey,
		SystemUserData:           s.SystemUserData,
		UserData:                 s.UserData,
	}
	if s.Image != nil {
		out.Image = toInfraBaremetalImage(s.Image)
	}
	if s.Online != nil {
		v := *s.Online
		out.Online = &v
	}
	if s.Raid != nil {
		out.Raid = toInfraBaremetalRaid(s.Raid)
	}
	if s.RootDeviceHints != nil {
		out.RootDeviceHints = toInfraBaremetalRootDeviceHints(s.RootDeviceHints)
	}
	return out
}

func toInfraBaremetalImage(img *apiv1.BaremetalImage) *InfraBaremetalImage {
	if img == nil {
		return nil
	}
	return &InfraBaremetalImage{
		Checksum:     img.Checksum,
		ChecksumType: img.ChecksumType,
		Format:       img.Format,
		URL:          img.URL,
	}
}

func toInfraBaremetalRootDeviceHints(h *apiv1.BaremetalRootDeviceHints) *InfraBaremetalRootDeviceHints {
	if h == nil {
		return nil
	}
	out := &InfraBaremetalRootDeviceHints{
		DeviceName:         h.DeviceName,
		HCTL:               h.HCTL,
		MinSizeGigabytes:   h.MinSizeGigabytes,
		Model:              h.Model,
		SerialNumber:       h.SerialNumber,
		Vendor:             h.Vendor,
		WWN:                h.WWN,
		WWNVendorExtension: h.WWNVendorExtension,
		WWNWithExtension:   h.WWNWithExtension,
	}
	if h.Rotational != nil {
		v := *h.Rotational
		out.Rotational = &v
	}
	return out
}

func toInfraBaremetalRaid(r *apiv1.BaremetalRaid) *InfraBaremetalRaid {
	if r == nil {
		return nil
	}
	out := &InfraBaremetalRaid{}
	if len(r.HardwareRAIDVolumes) > 0 {
		out.HardwareRAIDVolumes = make([]InfraBaremetalHardwareRAIDVolume, len(r.HardwareRAIDVolumes))
		for i, v := range r.HardwareRAIDVolumes {
			vol := InfraBaremetalHardwareRAIDVolume{
				Controller:            v.Controller,
				Level:                 v.Level,
				Name:                  v.Name,
				NumberOfPhysicalDisks: v.NumberOfPhysicalDisks,
				SizeGibibytes:         v.SizeGibibytes,
			}
			if len(v.PhysicalDisks) > 0 {
				vol.PhysicalDisks = append([]string(nil), v.PhysicalDisks...)
			}
			if v.Rotational != nil {
				rv := *v.Rotational
				vol.Rotational = &rv
			}
			out.HardwareRAIDVolumes[i] = vol
		}
	}
	if len(r.SoftwareRAIDVolumes) > 0 {
		out.SoftwareRAIDVolumes = make([]InfraBaremetalSoftwareRAIDVolume, len(r.SoftwareRAIDVolumes))
		for i, v := range r.SoftwareRAIDVolumes {
			vol := InfraBaremetalSoftwareRAIDVolume{
				Level:         v.Level,
				SizeGibibytes: v.SizeGibibytes,
			}
			if len(v.PhysicalDisks) > 0 {
				vol.PhysicalDisks = make([]InfraBaremetalRootDeviceHints, len(v.PhysicalDisks))
				for j := range v.PhysicalDisks {
					if hc := toInfraBaremetalRootDeviceHints(&v.PhysicalDisks[j]); hc != nil {
						vol.PhysicalDisks[j] = *hc
					}
				}
			}
			out.SoftwareRAIDVolumes[i] = vol
		}
	}
	return out
}

// ToInfraBaremetalImage exposes the image converter for sub-action payloads
// (e.g. ReinstallOS uses the same shape as BaremetalImage).
func ToInfraBaremetalImage(img *apiv1.BaremetalImage) *InfraBaremetalImage {
	return toInfraBaremetalImage(img)
}

// ToInfraBaremetalConsoleSessionRequest converts the SDK console-session
// request to the wire format (snake_case JSON tags).
func ToInfraBaremetalConsoleSessionRequest(req *apiv1.BaremetalConsoleSessionRequest) *InfraBaremetalConsoleSessionRequest {
	if req == nil {
		return nil
	}
	return &InfraBaremetalConsoleSessionRequest{ComputeID: req.ComputeID}
}

// ----- Converters: Wire -> SDK -----

// FromInfraBaremetalMachine converts the wire format back to a k8s-style
// BaremetalMachine.
func FromInfraBaremetalMachine(wire *InfraBaremetalMachine) *apiv1.BaremetalMachine {
	if wire == nil {
		return nil
	}
	return &apiv1.BaremetalMachine{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindBaremetalMachine,
		},
		Metadata: infraMetadataFromWire(wire.Metadata),
		Spec:     fromInfraBaremetalMachineSpec(wire.Spec),
		Status:   fromInfraBaremetalMachineStatus(wire.Status),
	}
}

func fromInfraBaremetalMachineSpec(s InfraBaremetalMachineSpec) apiv1.BaremetalMachineSpec {
	out := apiv1.BaremetalMachineSpec{
		Architecture:             s.Architecture,
		AutomatedCleaningMode:    s.AutomatedCleaningMode,
		BaremetalProvisionerName: s.BaremetalProvisionerName,
		BootMode:                 s.BootMode,
		Datacenter:               s.Datacenter,
		DeviceID:                 s.DeviceID,
		Hostname:                 s.Hostname,
		MACAddress:               s.MACAddress,
		SSHKey:                   s.SSHKey,
		SystemUserData:           s.SystemUserData,
		UserData:                 s.UserData,
	}
	if s.Image != nil {
		out.Image = fromInfraBaremetalImage(s.Image)
	}
	if s.Online != nil {
		v := *s.Online
		out.Online = &v
	}
	if s.Raid != nil {
		out.Raid = fromInfraBaremetalRaid(s.Raid)
	}
	if s.RootDeviceHints != nil {
		out.RootDeviceHints = fromInfraBaremetalRootDeviceHints(s.RootDeviceHints)
	}
	return out
}

func fromInfraBaremetalImage(img *InfraBaremetalImage) *apiv1.BaremetalImage {
	if img == nil {
		return nil
	}
	return &apiv1.BaremetalImage{
		Checksum:     img.Checksum,
		ChecksumType: img.ChecksumType,
		Format:       img.Format,
		URL:          img.URL,
	}
}

func fromInfraBaremetalRootDeviceHints(h *InfraBaremetalRootDeviceHints) *apiv1.BaremetalRootDeviceHints {
	if h == nil {
		return nil
	}
	out := &apiv1.BaremetalRootDeviceHints{
		DeviceName:         h.DeviceName,
		HCTL:               h.HCTL,
		MinSizeGigabytes:   h.MinSizeGigabytes,
		Model:              h.Model,
		SerialNumber:       h.SerialNumber,
		Vendor:             h.Vendor,
		WWN:                h.WWN,
		WWNVendorExtension: h.WWNVendorExtension,
		WWNWithExtension:   h.WWNWithExtension,
	}
	if h.Rotational != nil {
		v := *h.Rotational
		out.Rotational = &v
	}
	return out
}

func fromInfraBaremetalRaid(r *InfraBaremetalRaid) *apiv1.BaremetalRaid {
	if r == nil {
		return nil
	}
	out := &apiv1.BaremetalRaid{}
	if len(r.HardwareRAIDVolumes) > 0 {
		out.HardwareRAIDVolumes = make([]apiv1.BaremetalHardwareRAIDVolumes, len(r.HardwareRAIDVolumes))
		for i, v := range r.HardwareRAIDVolumes {
			vol := apiv1.BaremetalHardwareRAIDVolumes{
				Controller:            v.Controller,
				Level:                 v.Level,
				Name:                  v.Name,
				NumberOfPhysicalDisks: v.NumberOfPhysicalDisks,
				SizeGibibytes:         v.SizeGibibytes,
			}
			if len(v.PhysicalDisks) > 0 {
				vol.PhysicalDisks = append([]string(nil), v.PhysicalDisks...)
			}
			if v.Rotational != nil {
				rv := *v.Rotational
				vol.Rotational = &rv
			}
			out.HardwareRAIDVolumes[i] = vol
		}
	}
	if len(r.SoftwareRAIDVolumes) > 0 {
		out.SoftwareRAIDVolumes = make([]apiv1.BaremetalSoftwareRAIDVolumes, len(r.SoftwareRAIDVolumes))
		for i, v := range r.SoftwareRAIDVolumes {
			vol := apiv1.BaremetalSoftwareRAIDVolumes{
				Level:         v.Level,
				SizeGibibytes: v.SizeGibibytes,
			}
			if len(v.PhysicalDisks) > 0 {
				vol.PhysicalDisks = make([]apiv1.BaremetalRootDeviceHints, len(v.PhysicalDisks))
				for j := range v.PhysicalDisks {
					if hc := fromInfraBaremetalRootDeviceHints(&v.PhysicalDisks[j]); hc != nil {
						vol.PhysicalDisks[j] = *hc
					}
				}
			}
			out.SoftwareRAIDVolumes[i] = vol
		}
	}
	return out
}

func fromInfraBaremetalMachineStatus(s InfraBaremetalMachineStatus) apiv1.BaremetalMachineStatus {
	out := apiv1.BaremetalMachineStatus{}
	if len(s.Conditions) > 0 {
		out.Conditions = make([]apiv1.BaremetalMachineCondition, len(s.Conditions))
		for i, c := range s.Conditions {
			out.Conditions[i] = apiv1.BaremetalMachineCondition{
				LastUpdated: c.LastUpdated,
				Reason:      c.Reason,
				Status:      c.Status,
				Type:        c.Type,
			}
		}
	}
	return out
}

// FromInfraBaremetalMachineList converts a wire list to the SDK list shape.
func FromInfraBaremetalMachineList(wire *InfraBaremetalMachineList) *apiv1.BaremetalMachineList {
	if wire == nil {
		return &apiv1.BaremetalMachineList{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindBaremetalMachine + "List"},
		}
	}
	out := &apiv1.BaremetalMachineList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindBaremetalMachine + "List",
		},
	}
	out.Metadata.Continue = infraListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if bm := FromInfraBaremetalMachine(&wire.Items[i]); bm != nil {
			out.Items = append(out.Items, *bm)
		}
	}
	return out
}

// FromInfraBaremetalMachineInfo converts the wire info envelope to the SDK type.
func FromInfraBaremetalMachineInfo(wire *InfraBaremetalMachineInfo) *apiv1.BaremetalMachineInfo {
	if wire == nil {
		return &apiv1.BaremetalMachineInfo{}
	}
	out := &apiv1.BaremetalMachineInfo{}
	if wire.Data.Fields != nil {
		out.Data.Fields = make(map[string]interface{}, len(wire.Data.Fields))
		for k, v := range wire.Data.Fields {
			out.Data.Fields[k] = v
		}
	}
	return out
}

// FromInfraBaremetalConsoleSession converts the wire session response to SDK.
func FromInfraBaremetalConsoleSession(wire *InfraBaremetalConsoleSession) *apiv1.BaremetalConsoleSession {
	if wire == nil {
		return nil
	}
	return &apiv1.BaremetalConsoleSession{
		AgentSessionID: wire.AgentSessionID,
		ConsoleURL:     wire.ConsoleURL,
		SessionID:      wire.SessionID,
	}
}
