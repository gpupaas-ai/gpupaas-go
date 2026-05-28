package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	DevVirtualMachineKind     = "VirtualMachine"
	DevVirtualMachineListKind = "VirtualMachineList"
)

// DevScope identifies project- or workspace-scoped dev API paths.
type DevScope struct {
	Project   string
	Workspace string // empty => project scope
}

// VMScope is an alias for DevScope (virtual machines).
type VMScope = DevScope

// VMPaths returns path builders for the given scope.
func VMPaths(scope VMScope) (
	collection func() string,
	item func(name string) string,
	status func(name string) string,
	action func(name, act string) string,
) {
	if scope.Workspace != "" {
		return func() string {
				return fmt.Sprintf(DevWorkspaceVMsPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
			},
			func(name string) string {
				return fmt.Sprintf(DevWorkspaceVMPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
			},
			func(name string) string {
				return fmt.Sprintf(DevWorkspaceVMStatusPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
			},
			func(name, act string) string {
				return fmt.Sprintf(DevWorkspaceVMActionPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name), url.PathEscape(act))
			}
	}
	return func() string {
			return fmt.Sprintf(DevProjectVMsPath, url.PathEscape(scope.Project))
		},
		func(name string) string {
			return fmt.Sprintf(DevProjectVMPath, url.PathEscape(scope.Project), url.PathEscape(name))
		},
		func(name string) string {
			return fmt.Sprintf(DevProjectVMStatusPath, url.PathEscape(scope.Project), url.PathEscape(name))
		},
		func(name, act string) string {
			return fmt.Sprintf(DevProjectVMActionPath, url.PathEscape(scope.Project), url.PathEscape(name), url.PathEscape(act))
		}
}

// DevResourceRef is the wire format for a resource reference.
type DevResourceRef struct {
	Name          string `json:"name"`
	SystemCatalog bool   `json:"systemCatalog,omitempty"`
}

// DevVirtualMachineProjectSharingSpec is project sharing on the wire.
type DevVirtualMachineProjectSharingSpec struct {
	Name       string   `json:"name"`
	Workspaces []string `json:"workspaces,omitempty"`
}

// DevVirtualMachineSharingSpec is sharing configuration on the wire.
type DevVirtualMachineSharingSpec struct {
	ShareMode  string                                `json:"shareMode"`
	Workspaces []string                              `json:"workspaces,omitempty"`
	Projects   []DevVirtualMachineProjectSharingSpec `json:"projects,omitempty"`
}

// DevVirtualMachineSpec is the wire spec for a virtual machine.
type DevVirtualMachineSpec struct {
	VirtualMachine        DevResourceRef                `json:"virtual_machine"`
	Type                  string                        `json:"type,omitempty"`
	Name                  string                        `json:"name,omitempty"`
	CPUCount              string                        `json:"cpu_count,omitempty"`
	Memory                string                        `json:"memory,omitempty"`
	SecurityGroup         string                        `json:"security_group,omitempty"`
	SSHKey                string                        `json:"ssh_key,omitempty"`
	VPC                   string                        `json:"vpc,omitempty"`
	Subnet                string                        `json:"subnet,omitempty"`
	AssignPublicIP        bool                          `json:"assign_public_ip,omitempty"`
	Sharing               *DevVirtualMachineSharingSpec `json:"sharing,omitempty"`
	Datacenter            string                        `json:"datacenter,omitempty"`
	GuestPassword         string                        `json:"guest_password,omitempty"`
	DNSServers            []string                      `json:"dns_servers,omitempty"`
	UserData              string                        `json:"user_data,omitempty"`
	Timezone              string                        `json:"timezone,omitempty"`
	SharedStorage         string                        `json:"shared_storage,omitempty"`
	BlockStorageType      string                        `json:"block_storage_type,omitempty"`
	Image                 string                        `json:"image,omitempty"`
	BootDiskSize          int32                         `json:"boot_disk_size,omitempty"`
	CreateAdditionalBlock bool                          `json:"create_additional_block,omitempty"`
	AdditionalBlockSize   int32                         `json:"additional_block_size,omitempty"`
}

// DevVirtualMachineOutput is provisioning output on the wire.
type DevVirtualMachineOutput struct {
	HostName      string `json:"host_name,omitempty"`
	OSName        string `json:"os_name,omitempty"`
	PrivateIP     string `json:"private_ip,omitempty"`
	PublicIP      string `json:"public_ip,omitempty"`
	ServerHost    string `json:"server_host,omitempty"`
	UserName      string `json:"user_name,omitempty"`
	DiskMountPath string `json:"disk_mount_path,omitempty"`
}

// DevVirtualMachineStatus is runtime status on the wire.
type DevVirtualMachineStatus struct {
	Status          string                   `json:"status,omitempty"`
	Reason          string                   `json:"reason,omitempty"`
	Action          string                   `json:"action,omitempty"`
	Output          *DevVirtualMachineOutput `json:"output,omitempty"`
	ProvisionedAt   string                   `json:"provisioned_at,omitempty"`
	LastConnectedAt string                   `json:"last_connected_at,omitempty"`
}

// DevMetadata is resource metadata on dev.envmgmt.io.
type DevMetadata struct {
	Name        string            `json:"name"`
	Project     string            `json:"project,omitempty"`
	Workspace   string            `json:"workspace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DevVirtualMachine is the wire format for virtual machine CRUD on dev.envmgmt.io/v1.
type DevVirtualMachine struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   DevMetadata             `json:"metadata"`
	Spec       DevVirtualMachineSpec   `json:"spec,omitempty"`
	Status     DevVirtualMachineStatus `json:"status,omitempty"`
}

// DevVirtualMachineList is the wire format for virtual machine list responses.
type DevVirtualMachineList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []DevVirtualMachine `json:"items"`
}

// DevVirtualMachineActionPayload is the optional body for POST .../action/{action}.
type DevVirtualMachineActionPayload struct {
	Variables []map[string]string `json:"variables,omitempty"`
	Envs      []map[string]string `json:"envs,omitempty"`
}

// ToDevVirtualMachine converts a k8s-style VirtualMachine to the dev.envmgmt.io wire format.
func ToDevVirtualMachine(vm *apiv1.VirtualMachine, project, workspace string) *DevVirtualMachine {
	if vm == nil {
		return nil
	}
	projectName := vm.Metadata.Project
	if projectName == "" {
		projectName = project
	}
	wsName := vm.Metadata.Workspace
	if wsName == "" {
		wsName = workspace
	}
	return &DevVirtualMachine{
		APIVersion: DevAPIVersion,
		Kind:       DevVirtualMachineKind,
		Metadata: DevMetadata{
			Name:        vm.Metadata.Name,
			Project:     projectName,
			Workspace:   wsName,
			Labels:      copyStringMap(vm.Metadata.Labels),
			Annotations: copyStringMap(vm.Metadata.Annotations),
		},
		Spec:   toDevVirtualMachineSpec(vm.Spec),
		Status: DevVirtualMachineStatus{},
	}
}

func toDevVirtualMachineSpec(s apiv1.VirtualMachineSpec) DevVirtualMachineSpec {
	out := DevVirtualMachineSpec{
		VirtualMachine: DevResourceRef{
			Name:          s.VirtualMachine.Name,
			SystemCatalog: s.VirtualMachine.SystemCatalog,
		},
		Type:                  s.Type,
		Name:                  s.Name,
		CPUCount:              s.CPUCount,
		Memory:                s.Memory,
		SecurityGroup:         s.SecurityGroup,
		SSHKey:                s.SSHKey,
		VPC:                   s.VPC,
		Subnet:                s.Subnet,
		AssignPublicIP:        s.AssignPublicIP,
		Datacenter:            s.Datacenter,
		GuestPassword:         s.GuestPassword,
		UserData:              s.UserData,
		Timezone:              s.Timezone,
		SharedStorage:         s.SharedStorage,
		BlockStorageType:      s.BlockStorageType,
		Image:                 s.Image,
		BootDiskSize:          s.BootDiskSize,
		CreateAdditionalBlock: s.CreateAdditionalBlock,
		AdditionalBlockSize:   s.AdditionalBlockSize,
	}
	if len(s.DNSServers) > 0 {
		out.DNSServers = append([]string(nil), s.DNSServers...)
	}
	if s.Sharing != nil {
		out.Sharing = toDevVirtualMachineSharing(*s.Sharing)
	}
	return out
}

func toDevVirtualMachineSharing(s apiv1.VirtualMachineSharingSpec) *DevVirtualMachineSharingSpec {
	out := &DevVirtualMachineSharingSpec{
		ShareMode: s.ShareMode,
	}
	if len(s.Workspaces) > 0 {
		out.Workspaces = append([]string(nil), s.Workspaces...)
	}
	if len(s.Projects) > 0 {
		out.Projects = make([]DevVirtualMachineProjectSharingSpec, len(s.Projects))
		for i, p := range s.Projects {
			out.Projects[i] = DevVirtualMachineProjectSharingSpec{
				Name: p.Name,
			}
			if len(p.Workspaces) > 0 {
				out.Projects[i].Workspaces = append([]string(nil), p.Workspaces...)
			}
		}
	}
	return out
}

// FromDevVirtualMachine converts wire format to gpupaas.ai/v1alpha1.
func FromDevVirtualMachine(wire *DevVirtualMachine, workspace string) *apiv1.VirtualMachine {
	if wire == nil {
		return nil
	}
	ws := wire.Metadata.Workspace
	if ws == "" {
		ws = workspace
	}
	return &apiv1.VirtualMachine{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindVirtualMachine,
		},
		Metadata: apiv1.ObjectMeta{
			Name:        wire.Metadata.Name,
			Project:     wire.Metadata.Project,
			Workspace:   ws,
			Labels:      copyStringMap(wire.Metadata.Labels),
			Annotations: copyStringMap(wire.Metadata.Annotations),
		},
		Spec:   fromDevVirtualMachineSpec(wire.Spec),
		Status: fromDevVirtualMachineStatus(wire.Status),
	}
}

func fromDevVirtualMachineSpec(s DevVirtualMachineSpec) apiv1.VirtualMachineSpec {
	out := apiv1.VirtualMachineSpec{
		VirtualMachine: apiv1.ResourceRef{
			Name:          s.VirtualMachine.Name,
			SystemCatalog: s.VirtualMachine.SystemCatalog,
		},
		Type:                  s.Type,
		Name:                  s.Name,
		CPUCount:              s.CPUCount,
		Memory:                s.Memory,
		SecurityGroup:         s.SecurityGroup,
		SSHKey:                s.SSHKey,
		VPC:                   s.VPC,
		Subnet:                s.Subnet,
		AssignPublicIP:        s.AssignPublicIP,
		Datacenter:            s.Datacenter,
		GuestPassword:         s.GuestPassword,
		UserData:              s.UserData,
		Timezone:              s.Timezone,
		SharedStorage:         s.SharedStorage,
		BlockStorageType:      s.BlockStorageType,
		Image:                 s.Image,
		BootDiskSize:          s.BootDiskSize,
		CreateAdditionalBlock: s.CreateAdditionalBlock,
		AdditionalBlockSize:   s.AdditionalBlockSize,
	}
	if len(s.DNSServers) > 0 {
		out.DNSServers = append([]string(nil), s.DNSServers...)
	}
	if s.Sharing != nil {
		out.Sharing = fromDevVirtualMachineSharing(*s.Sharing)
	}
	return out
}

func fromDevVirtualMachineSharing(s DevVirtualMachineSharingSpec) *apiv1.VirtualMachineSharingSpec {
	out := &apiv1.VirtualMachineSharingSpec{
		ShareMode: s.ShareMode,
	}
	if len(s.Workspaces) > 0 {
		out.Workspaces = append([]string(nil), s.Workspaces...)
	}
	if len(s.Projects) > 0 {
		out.Projects = make([]apiv1.VirtualMachineProjectSharingSpec, len(s.Projects))
		for i, p := range s.Projects {
			out.Projects[i] = apiv1.VirtualMachineProjectSharingSpec{
				Name: p.Name,
			}
			if len(p.Workspaces) > 0 {
				out.Projects[i].Workspaces = append([]string(nil), p.Workspaces...)
			}
		}
	}
	return out
}

func fromDevVirtualMachineStatus(s DevVirtualMachineStatus) apiv1.VirtualMachineStatus {
	out := apiv1.VirtualMachineStatus{
		Status:          s.Status,
		Reason:          s.Reason,
		Action:          s.Action,
		ProvisionedAt:   s.ProvisionedAt,
		LastConnectedAt: s.LastConnectedAt,
	}
	if s.Output != nil {
		out.Output = &apiv1.VirtualMachineOutput{
			HostName:      s.Output.HostName,
			OSName:        s.Output.OSName,
			PrivateIP:     s.Output.PrivateIP,
			PublicIP:      s.Output.PublicIP,
			ServerHost:    s.Output.ServerHost,
			UserName:      s.Output.UserName,
			DiskMountPath: s.Output.DiskMountPath,
		}
	}
	return out
}

// FromDevVirtualMachineList converts a wire list to gpupaas.ai/v1alpha1.
func FromDevVirtualMachineList(wire *DevVirtualMachineList, workspace string) *apiv1.VirtualMachineList {
	if wire == nil {
		return &apiv1.VirtualMachineList{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindVirtualMachine + "List"},
			Items:    nil,
		}
	}
	out := &apiv1.VirtualMachineList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindVirtualMachine + "List",
		},
	}
	if wire.Metadata.Count > 0 {
		out.Metadata.Continue = fmt.Sprintf("%d", wire.Metadata.Offset+int64(len(wire.Items)))
	}
	for i := range wire.Items {
		if vm := FromDevVirtualMachine(&wire.Items[i], workspace); vm != nil {
			out.Items = append(out.Items, *vm)
		}
	}
	return out
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
