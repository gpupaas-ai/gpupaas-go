package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	// DevVMInstanceKind and friends are the wire kinds of the first-class virtual machine API.
	DevVMInstanceKind       = "VirtualMachine"
	DevVMInstanceListKind   = "VirtualMachineList"
	DevVMProfileKind        = "VmProfile"
	DevVMProfileListKind    = "VmProfileList"
	devVMInstanceAPIVersion = DevAPIVersion
	vmInstanceActionSegment = "action"
)

// VMInstancePaths returns the collection and item path builders for virtual machine instances. Workspace
// builders are returned when scope.Workspace is set.
func VMInstancePaths(scope DevScope) (collection func() string, item func(name string) string, action func(name, verb string) string) {
	if scope.Workspace != "" {
		collection = func() string {
			return fmt.Sprintf(DevWorkspaceVMsPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
		}
		item = func(name string) string {
			return fmt.Sprintf(DevWorkspaceVMPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
		}
	} else {
		collection = func() string { return fmt.Sprintf(DevProjectVMsPath, url.PathEscape(scope.Project)) }
		item = func(name string) string {
			return fmt.Sprintf(DevProjectVMPath, url.PathEscape(scope.Project), url.PathEscape(name))
		}
	}
	// Action verbs may contain spaces ("Reinstall OS"); they are escaped.
	action = func(name, verb string) string {
		return item(name) + "/" + vmInstanceActionSegment + "/" + url.PathEscape(verb)
	}
	return
}

// VMProfilePaths returns the collection and item path builders for virtual machine SKUs.
func VMProfilePaths(project string) (collection func() string, item func(name string) string) {
	collection = func() string { return fmt.Sprintf(DevVMProfilesPath, url.PathEscape(project)) }
	item = func(name string) string {
		return fmt.Sprintf(DevVMProfilePath, url.PathEscape(project), url.PathEscape(name))
	}
	return
}

// VMInstanceObject is the wire format of a virtual machine instance. The spec and status JSON match
// the SDK types, so they are reused directly.
type VMInstanceObject struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   DevMetadata             `json:"metadata"`
	Spec       apiv1.VMInstanceSpec    `json:"spec"`
	Status     *apiv1.VMInstanceStatus `json:"status,omitempty"`
}

// VMInstanceObjectList is the wire format of a virtual machine list response.
type VMInstanceObjectList struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   PaaSListMetadata   `json:"metadata,omitempty"`
	Items      []VMInstanceObject `json:"items"`
}

// ToVMInstance converts an SDK instance to the wire format for apply. Status is dropped.
func ToVMInstance(b *apiv1.VMInstance, project, workspace string) *VMInstanceObject {
	if b == nil {
		return nil
	}
	cp := b.DeepCopyObject().(*apiv1.VMInstance)
	return &VMInstanceObject{
		APIVersion: devVMInstanceAPIVersion,
		Kind:       DevVMInstanceKind,
		Metadata:   devMetadataToWire(cp.Metadata, project, workspace),
		Spec:       cp.Spec,
	}
}

// FromVMInstance converts a wire instance to the SDK type. The request scope is the fallback for
// project and workspace, because the backend does not always echo them.
func FromVMInstance(w *VMInstanceObject, project, workspace string) *apiv1.VMInstance {
	if w == nil {
		return nil
	}
	out := &apiv1.VMInstance{
		TypeMeta: apiv1.TypeMeta{APIVersion: devVMInstanceAPIVersion, Kind: apiv1.KindVMInstance},
		Metadata: devMetadataFromWire(w.Metadata, workspace),
		Spec:     w.Spec,
	}
	if out.Metadata.Project == "" {
		out.Metadata.Project = project
	}
	if w.Status != nil {
		out.Status = *w.Status
	}
	return out
}

// FromVMInstanceList converts a wire list to the SDK list.
func FromVMInstanceList(list *VMInstanceObjectList, project, workspace string) *apiv1.VMInstanceList {
	out := &apiv1.VMInstanceList{
		TypeMeta: apiv1.TypeMeta{APIVersion: devVMInstanceAPIVersion, Kind: DevVMInstanceListKind},
		Items:    make([]apiv1.VMInstance, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if b := FromVMInstance(&list.Items[i], project, workspace); b != nil {
			out.Items = append(out.Items, *b)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}

// VMProfileObject is the wire format of a virtual machine SKU.
type VMProfileObject struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   DevMetadata            `json:"metadata"`
	Spec       apiv1.VMProfileSpec    `json:"spec"`
	Status     *apiv1.VMProfileStatus `json:"status,omitempty"`
}

// VMProfileObjectList is the wire format of a SKU list response.
type VMProfileObjectList struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   PaaSListMetadata  `json:"metadata,omitempty"`
	Items      []VMProfileObject `json:"items"`
}

// FromVMProfile converts a wire SKU to the SDK type.
func FromVMProfile(w *VMProfileObject, project string) *apiv1.VMProfile {
	if w == nil {
		return nil
	}
	out := &apiv1.VMProfile{
		TypeMeta: apiv1.TypeMeta{APIVersion: devVMInstanceAPIVersion, Kind: apiv1.KindVMProfile},
		Metadata: devMetadataFromWire(w.Metadata, ""),
		Spec:     w.Spec,
	}
	if out.Metadata.Project == "" {
		out.Metadata.Project = project
	}
	if w.Status != nil {
		out.Status = *w.Status
	}
	return out
}

// FromVMProfileList converts a wire SKU list to the SDK list.
func FromVMProfileList(list *VMProfileObjectList, project string) *apiv1.VMProfileList {
	out := &apiv1.VMProfileList{
		TypeMeta: apiv1.TypeMeta{APIVersion: devVMInstanceAPIVersion, Kind: DevVMProfileListKind},
		Items:    make([]apiv1.VMProfile, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if p := FromVMProfile(&list.Items[i], project); p != nil {
			out.Items = append(out.Items, *p)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}
