package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	DevSecurityGroupKind     = "SecurityGroup"
	DevSecurityGroupListKind = "SecurityGroupList"
)

// SecurityGroupPaths returns collection and item path builders for the given scope.
func SecurityGroupPaths(scope DevScope) (collection func() string, item func(name string) string) {
	if scope.Workspace != "" {
		return func() string {
				return fmt.Sprintf(DevWorkspaceSecurityGroupsPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
			},
			func(name string) string {
				return fmt.Sprintf(DevWorkspaceSecurityGroupPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
			}
	}
	return func() string {
			return fmt.Sprintf(DevProjectSecurityGroupsPath, url.PathEscape(scope.Project))
		},
		func(name string) string {
			return fmt.Sprintf(DevProjectSecurityGroupPath, url.PathEscape(scope.Project), url.PathEscape(name))
		}
}

// DevIpRule is an IP rule on the wire.
type DevIpRule struct {
	SourceCIDR  string `json:"source_cidr,omitempty"`
	Application string `json:"application,omitempty"`
	Action      string `json:"action,omitempty"`
}

// DevPortForwardRule is a port-forward rule on the wire.
type DevPortForwardRule struct {
	SourceCIDR      string `json:"source_cidr,omitempty"`
	Application     string `json:"application,omitempty"`
	ApplicationPort string `json:"application_port,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
}

// DevRule is a security group rule on the wire.
type DevRule struct {
	SourceCIDR      string `json:"source_cidr,omitempty"`
	Application     string `json:"application,omitempty"`
	ApplicationPort string `json:"application_port,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	Action          string `json:"action,omitempty"`
}

// DevSecurityGroupSpec is the wire spec for a security group.
type DevSecurityGroupSpec struct {
	SecurityGroup    DevResourceRef      `json:"security_group"`
	Type             string              `json:"type,omitempty"`
	IPRules          []DevIpRule         `json:"ip_rules,omitempty"`
	PortForwardRules []DevPortForwardRule `json:"port_forward_rules,omitempty"`
	Rules            []DevRule           `json:"rules,omitempty"`
	Sharing          *DevSharingSpec     `json:"sharing,omitempty"`
}

// DevSecurityGroupStatus is runtime status on the wire.
type DevSecurityGroupStatus struct {
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
	Action string `json:"action,omitempty"`
}

// DevSecurityGroup is the wire format for security group CRUD.
type DevSecurityGroup struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   DevMetadata            `json:"metadata"`
	Spec       DevSecurityGroupSpec   `json:"spec,omitempty"`
	Status     DevSecurityGroupStatus `json:"status,omitempty"`
}

// DevSecurityGroupList is the wire format for security group list responses.
type DevSecurityGroupList struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   PaaSListMetadata   `json:"metadata,omitempty"`
	Items      []DevSecurityGroup `json:"items"`
}

// ToDevSecurityGroup converts SDK security group to wire format.
func ToDevSecurityGroup(sg *apiv1.SecurityGroup, project, workspace string) *DevSecurityGroup {
	if sg == nil {
		return nil
	}
	return &DevSecurityGroup{
		APIVersion: DevAPIVersion,
		Kind:       DevSecurityGroupKind,
		Metadata:   devMetadataToWire(sg.Metadata, project, workspace),
		Spec:       toDevSecurityGroupSpec(sg.Spec),
		Status:     DevSecurityGroupStatus{},
	}
}

func toDevSecurityGroupSpec(s apiv1.SecurityGroupSpec) DevSecurityGroupSpec {
	out := DevSecurityGroupSpec{
		SecurityGroup: DevResourceRef{
			Name:          s.SecurityGroup.Name,
			SystemCatalog: s.SecurityGroup.SystemCatalog,
		},
		Type: s.Type,
	}
	if len(s.IPRules) > 0 {
		out.IPRules = make([]DevIpRule, len(s.IPRules))
		for i, r := range s.IPRules {
			out.IPRules[i] = DevIpRule{
				SourceCIDR:  r.SourceCIDR,
				Application: r.Application,
				Action:      r.Action,
			}
		}
	}
	if len(s.PortForwardRules) > 0 {
		out.PortForwardRules = make([]DevPortForwardRule, len(s.PortForwardRules))
		for i, r := range s.PortForwardRules {
			out.PortForwardRules[i] = DevPortForwardRule{
				SourceCIDR:      r.SourceCIDR,
				Application:     r.Application,
				ApplicationPort: r.ApplicationPort,
				Protocol:        r.Protocol,
			}
		}
	}
	if len(s.Rules) > 0 {
		out.Rules = make([]DevRule, len(s.Rules))
		for i, r := range s.Rules {
			out.Rules[i] = DevRule{
				SourceCIDR:      r.SourceCIDR,
				Application:     r.Application,
				ApplicationPort: r.ApplicationPort,
				Protocol:        r.Protocol,
				Action:          r.Action,
			}
		}
	}
	if s.Sharing != nil {
		out.Sharing = toDevSharing(*s.Sharing)
	}
	return out
}

// FromDevSecurityGroup converts wire format to SDK security group.
func FromDevSecurityGroup(wire *DevSecurityGroup, workspace string) *apiv1.SecurityGroup {
	if wire == nil {
		return nil
	}
	return &apiv1.SecurityGroup{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindSecurityGroup,
		},
		Metadata: devMetadataFromWire(wire.Metadata, workspace),
		Spec:     fromDevSecurityGroupSpec(wire.Spec),
		Status:   fromDevSecurityGroupStatus(wire.Status),
	}
}

func fromDevSecurityGroupSpec(s DevSecurityGroupSpec) apiv1.SecurityGroupSpec {
	out := apiv1.SecurityGroupSpec{
		SecurityGroup: apiv1.ResourceRef{
			Name:          s.SecurityGroup.Name,
			SystemCatalog: s.SecurityGroup.SystemCatalog,
		},
		Type: s.Type,
	}
	if len(s.IPRules) > 0 {
		out.IPRules = make([]apiv1.IpRule, len(s.IPRules))
		for i, r := range s.IPRules {
			out.IPRules[i] = apiv1.IpRule{
				SourceCIDR:  r.SourceCIDR,
				Application: r.Application,
				Action:      r.Action,
			}
		}
	}
	if len(s.PortForwardRules) > 0 {
		out.PortForwardRules = make([]apiv1.PortForwardRule, len(s.PortForwardRules))
		for i, r := range s.PortForwardRules {
			out.PortForwardRules[i] = apiv1.PortForwardRule{
				SourceCIDR:      r.SourceCIDR,
				Application:     r.Application,
				ApplicationPort: r.ApplicationPort,
				Protocol:        r.Protocol,
			}
		}
	}
	if len(s.Rules) > 0 {
		out.Rules = make([]apiv1.Rule, len(s.Rules))
		for i, r := range s.Rules {
			out.Rules[i] = apiv1.Rule{
				SourceCIDR:      r.SourceCIDR,
				Application:     r.Application,
				ApplicationPort: r.ApplicationPort,
				Protocol:        r.Protocol,
				Action:          r.Action,
			}
		}
	}
	if s.Sharing != nil {
		out.Sharing = fromDevSharing(*s.Sharing)
	}
	return out
}

func fromDevSecurityGroupStatus(s DevSecurityGroupStatus) apiv1.SecurityGroupStatus {
	return apiv1.SecurityGroupStatus{
		Status: s.Status,
		Reason: s.Reason,
		Action: s.Action,
	}
}

// FromDevSecurityGroupList converts a wire list to SDK security group list.
func FromDevSecurityGroupList(wire *DevSecurityGroupList, workspace string) *apiv1.SecurityGroupList {
	if wire == nil {
		return &apiv1.SecurityGroupList{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindSecurityGroup + "List"},
		}
	}
	out := &apiv1.SecurityGroupList{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindSecurityGroup + "List",
		},
	}
	out.Metadata.Continue = devListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if item := FromDevSecurityGroup(&wire.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
