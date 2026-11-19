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

// SecurityGroupPaths returns the collection and item path builders for the given scope.
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
	Type    string          `json:"type,omitempty"`
	Rules   []DevRule       `json:"rules,omitempty"`
	Sharing *DevSharingSpec `json:"sharing,omitempty"`
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

// ToDevSecurityGroup converts an SDK security group to the wire format.
func ToDevSecurityGroup(sg *apiv1.SecurityGroup, project, workspace string) *DevSecurityGroup {
	if sg == nil {
		return nil
	}
	out := &DevSecurityGroup{
		APIVersion: DevAPIVersion,
		Kind:       DevSecurityGroupKind,
		Metadata:   devMetadataToWire(sg.Metadata, project, workspace),
		Spec:       DevSecurityGroupSpec{Type: sg.Spec.Type},
	}
	for _, r := range sg.Spec.Rules {
		out.Spec.Rules = append(out.Spec.Rules, DevRule{
			SourceCIDR:      r.SourceCIDR,
			Application:     r.Application,
			ApplicationPort: r.ApplicationPort,
			Protocol:        r.Protocol,
			Action:          r.Action,
		})
	}
	if sg.Spec.Sharing != nil {
		out.Spec.Sharing = toDevSharing(*sg.Spec.Sharing)
	}
	return out
}

// FromDevSecurityGroup converts wire format to an SDK security group.
func FromDevSecurityGroup(wire *DevSecurityGroup, workspace string) *apiv1.SecurityGroup {
	if wire == nil {
		return nil
	}
	out := &apiv1.SecurityGroup{
		TypeMeta: apiv1.TypeMeta{
			APIVersion: apiv1.APIVersion,
			Kind:       apiv1.KindSecurityGroup,
		},
		Metadata: devMetadataFromWire(wire.Metadata, workspace),
		Spec:     apiv1.SecurityGroupSpec{Type: wire.Spec.Type},
		Status: apiv1.SecurityGroupStatus{
			Status: wire.Status.Status,
			Reason: wire.Status.Reason,
			Action: wire.Status.Action,
		},
	}
	for _, r := range wire.Spec.Rules {
		out.Spec.Rules = append(out.Spec.Rules, apiv1.Rule{
			SourceCIDR:      r.SourceCIDR,
			Application:     r.Application,
			ApplicationPort: r.ApplicationPort,
			Protocol:        r.Protocol,
			Action:          r.Action,
		})
	}
	if wire.Spec.Sharing != nil {
		out.Spec.Sharing = fromDevSharing(*wire.Spec.Sharing)
	}
	return out
}

// FromDevSecurityGroupList converts a wire list to an SDK security group list.
func FromDevSecurityGroupList(wire *DevSecurityGroupList, workspace string) *apiv1.SecurityGroupList {
	out := &apiv1.SecurityGroupList{
		TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: DevSecurityGroupListKind},
		Items:    make([]apiv1.SecurityGroup, 0),
	}
	if wire == nil {
		return out
	}
	out.Metadata.Continue = devListContinue(wire.Metadata, len(wire.Items))
	for i := range wire.Items {
		if item := FromDevSecurityGroup(&wire.Items[i], workspace); item != nil {
			out.Items = append(out.Items, *item)
		}
	}
	return out
}
