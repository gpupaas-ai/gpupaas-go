package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	// DevBareMetalKind and friends are the wire kinds of the first-class bare-metal API.
	DevBareMetalKind       = "BareMetal"
	DevBareMetalListKind   = "BareMetalList"
	DevBmProfileKind       = "BmProfile"
	DevBmProfileListKind   = "BmProfileList"
	devBareMetalAPIVersion = DevAPIVersion
	bareMetalActionSegment = "action"
)

// BareMetalPaths returns the collection and item path builders for bare-metal instances. Workspace
// builders are returned when scope.Workspace is set.
func BareMetalPaths(scope DevScope) (collection func() string, item func(name string) string, action func(name, verb string) string) {
	if scope.Workspace != "" {
		collection = func() string {
			return fmt.Sprintf(DevWorkspaceBareMetalsPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
		}
		item = func(name string) string {
			return fmt.Sprintf(DevWorkspaceBareMetalPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
		}
	} else {
		collection = func() string { return fmt.Sprintf(DevProjectBareMetalsPath, url.PathEscape(scope.Project)) }
		item = func(name string) string {
			return fmt.Sprintf(DevProjectBareMetalPath, url.PathEscape(scope.Project), url.PathEscape(name))
		}
	}
	// Action verbs may contain spaces ("Reinstall OS"); they are escaped.
	action = func(name, verb string) string {
		return item(name) + "/" + bareMetalActionSegment + "/" + url.PathEscape(verb)
	}
	return
}

// BmProfilePaths returns the collection and item path builders for bare-metal SKUs.
func BmProfilePaths(project string) (collection func() string, item func(name string) string) {
	collection = func() string { return fmt.Sprintf(DevBmProfilesPath, url.PathEscape(project)) }
	item = func(name string) string {
		return fmt.Sprintf(DevBmProfilePath, url.PathEscape(project), url.PathEscape(name))
	}
	return
}

// BareMetalObject is the wire format of a bare-metal instance. The spec and status JSON match
// the SDK types, so they are reused directly.
type BareMetalObject struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   DevMetadata            `json:"metadata"`
	Spec       apiv1.BareMetalSpec    `json:"spec"`
	Status     *apiv1.BareMetalStatus `json:"status,omitempty"`
}

// BareMetalObjectList is the wire format of a bare-metal list response.
type BareMetalObjectList struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   PaaSListMetadata  `json:"metadata,omitempty"`
	Items      []BareMetalObject `json:"items"`
}

// ToBareMetal converts an SDK instance to the wire format for apply. Status is dropped.
func ToBareMetal(b *apiv1.BareMetal, project, workspace string) *BareMetalObject {
	if b == nil {
		return nil
	}
	cp := b.DeepCopyObject().(*apiv1.BareMetal)
	return &BareMetalObject{
		APIVersion: devBareMetalAPIVersion,
		Kind:       DevBareMetalKind,
		Metadata:   devMetadataToWire(cp.Metadata, project, workspace),
		Spec:       cp.Spec,
	}
}

// FromBareMetal converts a wire instance to the SDK type. The request scope is the fallback for
// project and workspace, because the backend does not always echo them.
func FromBareMetal(w *BareMetalObject, project, workspace string) *apiv1.BareMetal {
	if w == nil {
		return nil
	}
	out := &apiv1.BareMetal{
		TypeMeta: apiv1.TypeMeta{APIVersion: devBareMetalAPIVersion, Kind: apiv1.KindBareMetal},
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

// FromBareMetalList converts a wire list to the SDK list.
func FromBareMetalList(list *BareMetalObjectList, project, workspace string) *apiv1.BareMetalList {
	out := &apiv1.BareMetalList{
		TypeMeta: apiv1.TypeMeta{APIVersion: devBareMetalAPIVersion, Kind: DevBareMetalListKind},
		Items:    make([]apiv1.BareMetal, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if b := FromBareMetal(&list.Items[i], project, workspace); b != nil {
			out.Items = append(out.Items, *b)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}

// BmProfileObject is the wire format of a bare-metal SKU.
type BmProfileObject struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   DevMetadata            `json:"metadata"`
	Spec       apiv1.BmProfileSpec    `json:"spec"`
	Status     *apiv1.BmProfileStatus `json:"status,omitempty"`
}

// BmProfileObjectList is the wire format of a SKU list response.
type BmProfileObjectList struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   PaaSListMetadata  `json:"metadata,omitempty"`
	Items      []BmProfileObject `json:"items"`
}

// FromBmProfile converts a wire SKU to the SDK type.
func FromBmProfile(w *BmProfileObject, project string) *apiv1.BmProfile {
	if w == nil {
		return nil
	}
	out := &apiv1.BmProfile{
		TypeMeta: apiv1.TypeMeta{APIVersion: devBareMetalAPIVersion, Kind: apiv1.KindBmProfile},
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

// FromBmProfileList converts a wire SKU list to the SDK list.
func FromBmProfileList(list *BmProfileObjectList, project string) *apiv1.BmProfileList {
	out := &apiv1.BmProfileList{
		TypeMeta: apiv1.TypeMeta{APIVersion: devBareMetalAPIVersion, Kind: DevBmProfileListKind},
		Items:    make([]apiv1.BmProfile, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if p := FromBmProfile(&list.Items[i], project); p != nil {
			out.Items = append(out.Items, *p)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}
