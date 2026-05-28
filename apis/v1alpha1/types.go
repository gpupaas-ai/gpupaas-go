// Package v1alpha1 defines gpupaas.ai API types.
package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

const (
	Group      = "gpupaas.ai"
	Version    = "v1alpha1"
	APIVersion = Group + "/" + Version

	KindProject               = "Project"
	KindWorkspace             = "Workspace"
	KindWorkspaceCollaborator = "WorkspaceCollaborator"
	KindVirtualMachine        = "VirtualMachine"
	KindStorage               = "Storage"
	KindSecurityGroup         = "SecurityGroup"
	KindSshKey                = "SshKey"
)

// Workspace collaborator roles on the paas.envmgmt.io API.
const (
	WorkspaceRoleCollaborator         = "PAAS_WORKSPACE_COLLABORATOR"           // read+write
	WorkspaceRoleCollaboratorReadOnly = "PAAS_WORKSPACE_COLLABORATOR_READ_ONLY" // read-only
)

// ValidWorkspaceCollaboratorRole reports whether role is a supported collaborator role.
func ValidWorkspaceCollaboratorRole(role string) bool {
	return role == WorkspaceRoleCollaborator || role == WorkspaceRoleCollaboratorReadOnly
}

// TypeMeta describes the API version and kind of a resource.
type TypeMeta struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

// ObjectMeta holds standard resource metadata.
type ObjectMeta struct {
	Name        string            `json:"name" yaml:"name"`
	Project     string            `json:"project,omitempty" yaml:"project,omitempty"`
	Workspace   string            `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// ListMeta holds list metadata.
type ListMeta struct {
	Continue string `json:"continue,omitempty" yaml:"continue,omitempty"`
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

func copyObjectMeta(m ObjectMeta) ObjectMeta {
	return ObjectMeta{
		Name:        m.Name,
		Project:     m.Project,
		Workspace:   m.Workspace,
		Labels:      copyStringMap(m.Labels),
		Annotations: copyStringMap(m.Annotations),
	}
}

// ProjectSpec holds desired project state.
type ProjectSpec struct {
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Default     bool   `json:"default,omitempty" yaml:"default,omitempty"`
}

// ProjectStatus holds observed project state.
type ProjectStatus struct {
	Phase string `json:"phase,omitempty" yaml:"phase,omitempty"`
}

// Project is a top-level gpupaas platform container.
type Project struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta    `json:"metadata" yaml:"metadata"`
	Spec     ProjectSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   ProjectStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (p *Project) GetAPIVersion() string { return p.APIVersion }
func (p *Project) GetKind() string       { return p.Kind }
func (p *Project) GetName() string       { return p.Metadata.Name }
func (p *Project) GetProject() string    { return p.Metadata.Project }
func (p *Project) GetWorkspace() string  { return p.Metadata.Workspace }
func (p *Project) SetProject(v string)   { p.Metadata.Project = v }
func (p *Project) SetWorkspace(v string) { p.Metadata.Workspace = v }
func (p *Project) DeepCopyObject() runtime.Object {
	cp := *p
	cp.Metadata = copyObjectMeta(p.Metadata)
	return &cp
}

// ProjectList is a list of Project resources.
type ProjectList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []Project `json:"items" yaml:"items"`
}

func (l *ProjectList) GetAPIVersion() string { return l.APIVersion }
func (l *ProjectList) GetKind() string       { return l.Kind }
func (l *ProjectList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *ProjectList) SetItems(items []runtime.Object) {
	l.Items = make([]Project, len(items))
	for i, item := range items {
		if p, ok := item.(*Project); ok {
			l.Items[i] = *p
		}
	}
}

// WorkspaceSpec holds desired workspace state.
type WorkspaceSpec struct {
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	IconURL     string `json:"iconURL,omitempty" yaml:"iconURL,omitempty"`
	Readme      string `json:"readme,omitempty" yaml:"readme,omitempty"`
}

// WorkspaceStatus holds observed workspace state.
type WorkspaceStatus struct {
	Phase string `json:"phase,omitempty" yaml:"phase,omitempty"`
}

// Workspace is a partition within a project.
type Workspace struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta      `json:"metadata" yaml:"metadata"`
	Spec     WorkspaceSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   WorkspaceStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (w *Workspace) GetAPIVersion() string { return w.APIVersion }
func (w *Workspace) GetKind() string       { return w.Kind }
func (w *Workspace) GetName() string       { return w.Metadata.Name }
func (w *Workspace) GetProject() string    { return w.Metadata.Project }
func (w *Workspace) GetWorkspace() string  { return w.Metadata.Workspace }
func (w *Workspace) SetProject(v string)   { w.Metadata.Project = v }
func (w *Workspace) SetWorkspace(v string) { w.Metadata.Workspace = v }
func (w *Workspace) DeepCopyObject() runtime.Object {
	cp := *w
	cp.Metadata = copyObjectMeta(w.Metadata)
	return &cp
}

// WorkspaceList is a list of Workspace resources.
type WorkspaceList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []Workspace `json:"items" yaml:"items"`
}

func (l *WorkspaceList) GetAPIVersion() string { return l.APIVersion }
func (l *WorkspaceList) GetKind() string       { return l.Kind }
func (l *WorkspaceList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *WorkspaceList) SetItems(items []runtime.Object) {
	l.Items = make([]Workspace, len(items))
	for i, item := range items {
		if w, ok := item.(*Workspace); ok {
			l.Items[i] = *w
		}
	}
}

// WorkspaceCollaboratorSpec holds desired collaborator state.
//
// Assign an existing Rafay user: set username (or metadata.name) and role.
// Invite a new external user: set email and role (POST .../collaborators).
//
// Supported spec.role values:
//   - PAAS_WORKSPACE_COLLABORATOR — read+write workspace access
//   - PAAS_WORKSPACE_COLLABORATOR_READ_ONLY — read-only workspace access
//
// isSSOUser is sent as the ssoUsers=true query parameter on assign/unassign (and list filter).
type WorkspaceCollaboratorSpec struct {
	Username  string `json:"username,omitempty" yaml:"username,omitempty"`
	Email     string `json:"email,omitempty" yaml:"email,omitempty"`
	FirstName string `json:"firstName,omitempty" yaml:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty" yaml:"lastName,omitempty"`
	Role      string `json:"role,omitempty" yaml:"role,omitempty"`
	UserType  string `json:"userType,omitempty" yaml:"userType,omitempty"`
	IsSSOUser bool   `json:"isSSOUser,omitempty" yaml:"isSSOUser,omitempty"`
}

// ResolvedRole returns the backend role string for this spec.
func (s WorkspaceCollaboratorSpec) ResolvedRole() string {
	return s.Role
}

// WorkspaceCollaboratorStatus holds observed collaborator state.
type WorkspaceCollaboratorStatus struct {
	Phase string `json:"phase,omitempty" yaml:"phase,omitempty"`
	Role  string `json:"role,omitempty" yaml:"role,omitempty"`
}

// WorkspaceCollaborator is a user with access to a workspace.
type WorkspaceCollaborator struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta                  `json:"metadata" yaml:"metadata"`
	Spec     WorkspaceCollaboratorSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   WorkspaceCollaboratorStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (c *WorkspaceCollaborator) GetAPIVersion() string { return c.APIVersion }
func (c *WorkspaceCollaborator) GetKind() string       { return c.Kind }
func (c *WorkspaceCollaborator) GetName() string       { return c.Metadata.Name }
func (c *WorkspaceCollaborator) GetProject() string    { return c.Metadata.Project }
func (c *WorkspaceCollaborator) GetWorkspace() string  { return c.Metadata.Workspace }
func (c *WorkspaceCollaborator) SetProject(v string)   { c.Metadata.Project = v }
func (c *WorkspaceCollaborator) SetWorkspace(v string) { c.Metadata.Workspace = v }
func (c *WorkspaceCollaborator) DeepCopyObject() runtime.Object {
	cp := *c
	cp.Metadata = copyObjectMeta(c.Metadata)
	return &cp
}

// AssignExisting reports whether Apply should call assigncollaborators (existing Rafay user).
func (c *WorkspaceCollaborator) AssignExisting() bool {
	return c.Spec.Email == "" && (c.Spec.Username != "" || c.Metadata.Name != "")
}

// CollaboratorUsername returns the Rafay username used for assign/unassign.
func (c *WorkspaceCollaborator) CollaboratorUsername() string {
	if c.Spec.Username != "" {
		return c.Spec.Username
	}
	return c.Metadata.Name
}

// WorkspaceCollaboratorList is a list of WorkspaceCollaborator resources.
type WorkspaceCollaboratorList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta                `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []WorkspaceCollaborator `json:"items" yaml:"items"`
}

func (l *WorkspaceCollaboratorList) GetAPIVersion() string { return l.APIVersion }
func (l *WorkspaceCollaboratorList) GetKind() string       { return l.Kind }
func (l *WorkspaceCollaboratorList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *WorkspaceCollaboratorList) SetItems(items []runtime.Object) {
	l.Items = make([]WorkspaceCollaborator, len(items))
	for i, item := range items {
		if c, ok := item.(*WorkspaceCollaborator); ok {
			l.Items[i] = *c
		}
	}
}

// ResourceRef references another catalog resource (e.g. VM profile).
type ResourceRef struct {
	Name          string `json:"name" yaml:"name"`
	SystemCatalog bool   `json:"systemCatalog,omitempty" yaml:"systemCatalog,omitempty"`
}

// VirtualMachineProjectSharingSpec describes project-level sharing for a VM.
type VirtualMachineProjectSharingSpec struct {
	Name       string   `json:"name" yaml:"name"`
	Workspaces []string `json:"workspaces,omitempty" yaml:"workspaces,omitempty"`
}

// VirtualMachineSharingSpec holds VM sharing configuration.
type VirtualMachineSharingSpec struct {
	ShareMode   string                             `json:"shareMode" yaml:"shareMode"`
	Workspaces  []string                           `json:"workspaces,omitempty" yaml:"workspaces,omitempty"`
	Projects    []VirtualMachineProjectSharingSpec `json:"projects,omitempty" yaml:"projects,omitempty"`
}

// VirtualMachineSpec holds desired virtual machine state.
type VirtualMachineSpec struct {
	VirtualMachine        ResourceRef                `json:"virtualMachine" yaml:"virtualMachine"`
	Type                  string                     `json:"type,omitempty" yaml:"type,omitempty"`
	Name                  string                     `json:"name,omitempty" yaml:"name,omitempty"`
	CPUCount              string                     `json:"cpuCount,omitempty" yaml:"cpuCount,omitempty"`
	Memory                string                     `json:"memory,omitempty" yaml:"memory,omitempty"`
	SecurityGroup         string                     `json:"securityGroup,omitempty" yaml:"securityGroup,omitempty"`
	SSHKey                string                     `json:"sshKey,omitempty" yaml:"sshKey,omitempty"`
	VPC                   string                     `json:"vpc,omitempty" yaml:"vpc,omitempty"`
	Subnet                string                     `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	AssignPublicIP        bool                       `json:"assignPublicIp,omitempty" yaml:"assignPublicIp,omitempty"`
	Sharing               *VirtualMachineSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`
	Datacenter            string                     `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	GuestPassword         string                     `json:"guestPassword,omitempty" yaml:"guestPassword,omitempty"`
	DNSServers            []string                   `json:"dnsServers,omitempty" yaml:"dnsServers,omitempty"`
	UserData              string                     `json:"userData,omitempty" yaml:"userData,omitempty"`
	Timezone              string                     `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	SharedStorage         string                     `json:"sharedStorage,omitempty" yaml:"sharedStorage,omitempty"`
	BlockStorageType      string                     `json:"blockStorageType,omitempty" yaml:"blockStorageType,omitempty"`
	Image                 string                     `json:"image,omitempty" yaml:"image,omitempty"`
	BootDiskSize          int32                      `json:"bootDiskSize,omitempty" yaml:"bootDiskSize,omitempty"`
	CreateAdditionalBlock bool                       `json:"createAdditionalBlock,omitempty" yaml:"createAdditionalBlock,omitempty"`
	AdditionalBlockSize   int32                      `json:"additionalBlockSize,omitempty" yaml:"additionalBlockSize,omitempty"`
}

// VirtualMachineOutput holds provisioning output from the backend.
type VirtualMachineOutput struct {
	HostName      string `json:"hostName,omitempty" yaml:"hostName,omitempty"`
	OSName        string `json:"osName,omitempty" yaml:"osName,omitempty"`
	PrivateIP     string `json:"privateIp,omitempty" yaml:"privateIp,omitempty"`
	PublicIP      string `json:"publicIp,omitempty" yaml:"publicIp,omitempty"`
	ServerHost    string `json:"serverHost,omitempty" yaml:"serverHost,omitempty"`
	UserName      string `json:"userName,omitempty" yaml:"userName,omitempty"`
	DiskMountPath string `json:"diskMountPath,omitempty" yaml:"diskMountPath,omitempty"`
}

// VirtualMachineStatus holds observed virtual machine state.
type VirtualMachineStatus struct {
	Status          string                `json:"status,omitempty" yaml:"status,omitempty"`
	Reason          string                `json:"reason,omitempty" yaml:"reason,omitempty"`
	Action          string                `json:"action,omitempty" yaml:"action,omitempty"`
	Output          *VirtualMachineOutput `json:"output,omitempty" yaml:"output,omitempty"`
	ProvisionedAt   string                `json:"provisionedAt,omitempty" yaml:"provisionedAt,omitempty"`
	LastConnectedAt string                `json:"lastConnectedAt,omitempty" yaml:"lastConnectedAt,omitempty"`
}

// VirtualMachine is a dev virtual machine instance.
type VirtualMachine struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta           `json:"metadata" yaml:"metadata"`
	Spec     VirtualMachineSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   VirtualMachineStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (v *VirtualMachine) GetAPIVersion() string { return v.APIVersion }
func (v *VirtualMachine) GetKind() string       { return v.Kind }
func (v *VirtualMachine) GetName() string       { return v.Metadata.Name }
func (v *VirtualMachine) GetProject() string    { return v.Metadata.Project }
func (v *VirtualMachine) GetWorkspace() string  { return v.Metadata.Workspace }
func (v *VirtualMachine) SetProject(val string) { v.Metadata.Project = val }
func (v *VirtualMachine) SetWorkspace(val string) {
	v.Metadata.Workspace = val
}
func (v *VirtualMachine) DeepCopyObject() runtime.Object {
	cp := *v
	cp.Metadata = copyObjectMeta(v.Metadata)
	cp.Spec = copyVirtualMachineSpec(v.Spec)
	if v.Status.Output != nil {
		out := *v.Status.Output
		cp.Status.Output = &out
	}
	return &cp
}

func copyVirtualMachineSpec(s VirtualMachineSpec) VirtualMachineSpec {
	cp := s
	if s.Sharing != nil {
		sh := *s.Sharing
		if len(s.Sharing.Workspaces) > 0 {
			sh.Workspaces = append([]string(nil), s.Sharing.Workspaces...)
		}
		if len(s.Sharing.Projects) > 0 {
			sh.Projects = make([]VirtualMachineProjectSharingSpec, len(s.Sharing.Projects))
			for i, p := range s.Sharing.Projects {
				sh.Projects[i] = p
				if len(p.Workspaces) > 0 {
					sh.Projects[i].Workspaces = append([]string(nil), p.Workspaces...)
				}
			}
		}
		cp.Sharing = &sh
	}
	if len(s.DNSServers) > 0 {
		cp.DNSServers = append([]string(nil), s.DNSServers...)
	}
	return cp
}

// VirtualMachineList is a list of VirtualMachine resources.
type VirtualMachineList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta         `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []VirtualMachine `json:"items" yaml:"items"`
}

func (l *VirtualMachineList) GetAPIVersion() string { return l.APIVersion }
func (l *VirtualMachineList) GetKind() string       { return l.Kind }
func (l *VirtualMachineList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *VirtualMachineList) SetItems(items []runtime.Object) {
	l.Items = make([]VirtualMachine, len(items))
	for i, item := range items {
		if v, ok := item.(*VirtualMachine); ok {
			l.Items[i] = *v
		}
	}
}
