package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// DevProjectSharingSpec describes project-level sharing for a dev resource.
type DevProjectSharingSpec struct {
	Name       string   `json:"name" yaml:"name"`
	Workspaces []string `json:"workspaces,omitempty" yaml:"workspaces,omitempty"`
}

// DevSharingSpec holds project/workspace sharing configuration.
type DevSharingSpec struct {
	ShareMode  string                  `json:"shareMode" yaml:"shareMode"`
	Workspaces []string                `json:"workspaces,omitempty" yaml:"workspaces,omitempty"`
	Projects   []DevProjectSharingSpec `json:"projects,omitempty" yaml:"projects,omitempty"`
}

func copyDevSharingSpec(s *DevSharingSpec) *DevSharingSpec {
	if s == nil {
		return nil
	}
	sh := *s
	if len(s.Workspaces) > 0 {
		sh.Workspaces = append([]string(nil), s.Workspaces...)
	}
	if len(s.Projects) > 0 {
		sh.Projects = make([]DevProjectSharingSpec, len(s.Projects))
		for i, p := range s.Projects {
			sh.Projects[i] = p
			if len(p.Workspaces) > 0 {
				sh.Projects[i].Workspaces = append([]string(nil), p.Workspaces...)
			}
		}
	}
	return &sh
}

// StorageSpec holds desired storage state.
type StorageSpec struct {
	Storage                   ResourceRef     `json:"storage" yaml:"storage"`
	Type                      string          `json:"type,omitempty" yaml:"type,omitempty"`
	Size                      string          `json:"size,omitempty" yaml:"size,omitempty"`
	Datacenter                string          `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`
	AccessPolicy              string          `json:"accessPolicy,omitempty" yaml:"accessPolicy,omitempty"`
	ContractTerm              string          `json:"contractTerm,omitempty" yaml:"contractTerm,omitempty"`
	EnableEncryptionAtRest    string          `json:"enableEncryptionAtRest,omitempty" yaml:"enableEncryptionAtRest,omitempty"`
	EnableEncryptionInTransit string          `json:"enableEncryptionInTransit,omitempty" yaml:"enableEncryptionInTransit,omitempty"`
	Sharing                   *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`
	StorageType               string          `json:"storageType,omitempty" yaml:"storageType,omitempty"`
}

// StorageStatus holds observed storage state.
type StorageStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Action string `json:"action,omitempty" yaml:"action,omitempty"`
}

// Storage is a dev storage resource.
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
	cp.Spec = copyStorageSpec(s.Spec)
	return &cp
}

func copyStorageSpec(s StorageSpec) StorageSpec {
	cp := s
	cp.Sharing = copyDevSharingSpec(s.Sharing)
	return cp
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

// IpRule is an IP-based security group rule.
type IpRule struct {
	SourceCIDR  string `json:"sourceCidr,omitempty" yaml:"sourceCidr,omitempty"`
	Application string `json:"application,omitempty" yaml:"application,omitempty"`
	Action      string `json:"action,omitempty" yaml:"action,omitempty"`
}

// PortForwardRule is a port-forwarding security group rule.
type PortForwardRule struct {
	SourceCIDR      string `json:"sourceCidr,omitempty" yaml:"sourceCidr,omitempty"`
	Application     string `json:"application,omitempty" yaml:"application,omitempty"`
	ApplicationPort string `json:"applicationPort,omitempty" yaml:"applicationPort,omitempty"`
	Protocol        string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// Rule is a security group access rule.
type Rule struct {
	SourceCIDR      string `json:"sourceCidr,omitempty" yaml:"sourceCidr,omitempty"`
	Application     string `json:"application,omitempty" yaml:"application,omitempty"`
	ApplicationPort string `json:"applicationPort,omitempty" yaml:"applicationPort,omitempty"`
	Protocol        string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Action          string `json:"action,omitempty" yaml:"action,omitempty"`
}

// SecurityGroupSpec holds desired security group state.
type SecurityGroupSpec struct {
	SecurityGroup    ResourceRef       `json:"securityGroup" yaml:"securityGroup"`
	Type             string            `json:"type,omitempty" yaml:"type,omitempty"`
	IPRules          []IpRule          `json:"ipRules,omitempty" yaml:"ipRules,omitempty"`
	PortForwardRules []PortForwardRule `json:"portForwardRules,omitempty" yaml:"portForwardRules,omitempty"`
	Rules            []Rule            `json:"rules,omitempty" yaml:"rules,omitempty"`
	Sharing          *DevSharingSpec   `json:"sharing,omitempty" yaml:"sharing,omitempty"`
}

// SecurityGroupStatus holds observed security group state.
type SecurityGroupStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Action string `json:"action,omitempty" yaml:"action,omitempty"`
}

// SecurityGroup is a dev security group resource.
type SecurityGroup struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta          `json:"metadata" yaml:"metadata"`
	Spec     SecurityGroupSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   SecurityGroupStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (s *SecurityGroup) GetAPIVersion() string { return s.APIVersion }
func (s *SecurityGroup) GetKind() string       { return s.Kind }
func (s *SecurityGroup) GetName() string       { return s.Metadata.Name }
func (s *SecurityGroup) GetProject() string    { return s.Metadata.Project }
func (s *SecurityGroup) GetWorkspace() string  { return s.Metadata.Workspace }
func (s *SecurityGroup) SetProject(val string) { s.Metadata.Project = val }
func (s *SecurityGroup) SetWorkspace(val string) {
	s.Metadata.Workspace = val
}
func (s *SecurityGroup) DeepCopyObject() runtime.Object {
	cp := *s
	cp.Metadata = copyObjectMeta(s.Metadata)
	cp.Spec = copySecurityGroupSpec(s.Spec)
	return &cp
}

func copySecurityGroupSpec(s SecurityGroupSpec) SecurityGroupSpec {
	cp := s
	if len(s.IPRules) > 0 {
		cp.IPRules = append([]IpRule(nil), s.IPRules...)
	}
	if len(s.PortForwardRules) > 0 {
		cp.PortForwardRules = append([]PortForwardRule(nil), s.PortForwardRules...)
	}
	if len(s.Rules) > 0 {
		cp.Rules = append([]Rule(nil), s.Rules...)
	}
	cp.Sharing = copyDevSharingSpec(s.Sharing)
	return cp
}

// SecurityGroupList is a list of SecurityGroup resources.
type SecurityGroupList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta        `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []SecurityGroup `json:"items" yaml:"items"`
}

func (l *SecurityGroupList) GetAPIVersion() string { return l.APIVersion }
func (l *SecurityGroupList) GetKind() string       { return l.Kind }
func (l *SecurityGroupList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *SecurityGroupList) SetItems(items []runtime.Object) {
	l.Items = make([]SecurityGroup, len(items))
	for i, item := range items {
		if s, ok := item.(*SecurityGroup); ok {
			l.Items[i] = *s
		}
	}
}

// SshKeySpec holds desired SSH key state.
type SshKeySpec struct {
	SSHKey    ResourceRef     `json:"sshKey" yaml:"sshKey"`
	Type      string          `json:"type,omitempty" yaml:"type,omitempty"`
	Name      string          `json:"name,omitempty" yaml:"name,omitempty"`
	PublicKey string          `json:"publicKey,omitempty" yaml:"publicKey,omitempty"`
	Sharing   *DevSharingSpec `json:"sharing,omitempty" yaml:"sharing,omitempty"`
}

// SshKeyStatus holds observed SSH key state.
type SshKeyStatus struct {
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Action string `json:"action,omitempty" yaml:"action,omitempty"`
}

// SshKey is a dev SSH key resource.
type SshKey struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta   `json:"metadata" yaml:"metadata"`
	Spec     SshKeySpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status   SshKeyStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (s *SshKey) GetAPIVersion() string { return s.APIVersion }
func (s *SshKey) GetKind() string       { return s.Kind }
func (s *SshKey) GetName() string       { return s.Metadata.Name }
func (s *SshKey) GetProject() string    { return s.Metadata.Project }
func (s *SshKey) GetWorkspace() string  { return s.Metadata.Workspace }
func (s *SshKey) SetProject(val string) { s.Metadata.Project = val }
func (s *SshKey) SetWorkspace(val string) {
	s.Metadata.Workspace = val
}
func (s *SshKey) DeepCopyObject() runtime.Object {
	cp := *s
	cp.Metadata = copyObjectMeta(s.Metadata)
	cp.Spec = copySshKeySpec(s.Spec)
	return &cp
}

func copySshKeySpec(s SshKeySpec) SshKeySpec {
	cp := s
	cp.Sharing = copyDevSharingSpec(s.Sharing)
	return cp
}

// SshKeyList is a list of SshKey resources.
type SshKeyList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []SshKey `json:"items" yaml:"items"`
}

func (l *SshKeyList) GetAPIVersion() string { return l.APIVersion }
func (l *SshKeyList) GetKind() string       { return l.Kind }
func (l *SshKeyList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *SshKeyList) SetItems(items []runtime.Object) {
	l.Items = make([]SshKey, len(items))
	for i, item := range items {
		if s, ok := item.(*SshKey); ok {
			l.Items[i] = *s
		}
	}
}
