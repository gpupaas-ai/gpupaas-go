package convert

import (
	"fmt"
	"net/url"

	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

const (
	// MKSClusterKind and friends are the wire kinds of the first-class MKS API.
	MKSClusterKind            = "Cluster"
	MKSClusterListKind        = "ClusterList"
	MKSClusterProfileKind     = "ClusterProfile"
	MKSClusterProfileListKind = "ClusterProfileList"
	MKSNodeKind               = "ClusterNode"
	MKSNodeListKind           = "ClusterNodeList"
)

// MKSClusterPaths returns the collection and item path builders for clusters. Workspace
// builders are returned when scope.Workspace is set.
func MKSClusterPaths(scope DevScope) (collection func() string, item func(name string) string) {
	if scope.Workspace != "" {
		collection = func() string {
			return fmt.Sprintf(MKSWorkspaceClustersPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace))
		}
		item = func(name string) string {
			return fmt.Sprintf(MKSWorkspaceClusterPath, url.PathEscape(scope.Project), url.PathEscape(scope.Workspace), url.PathEscape(name))
		}
		return
	}
	collection = func() string { return fmt.Sprintf(MKSClustersPath, url.PathEscape(scope.Project)) }
	item = func(name string) string {
		return fmt.Sprintf(MKSClusterPath, url.PathEscape(scope.Project), url.PathEscape(name))
	}
	return
}

// MKSClusterProfilePaths returns the collection and item path builders for cluster profiles.
func MKSClusterProfilePaths(project string) (collection func() string, item func(name string) string) {
	collection = func() string { return fmt.Sprintf(MKSClusterProfilesPath, url.PathEscape(project)) }
	item = func(name string) string {
		return fmt.Sprintf(MKSClusterProfilePath, url.PathEscape(project), url.PathEscape(name))
	}
	return
}

// MKSClusterObject is the wire format of a cluster. The spec and status JSON match the SDK
// types, so they are reused directly; unmodeled spec fields round-trip through Extra.
type MKSClusterObject struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   DevMetadata             `json:"metadata"`
	Spec       apiv1.MKSClusterSpec    `json:"spec"`
	Status     *apiv1.MKSClusterStatus `json:"status,omitempty"`
	// Version is the backend revision of the cluster; it is echoed back on update.
	Version int64 `json:"version,omitempty"`
}

// MKSClusterObjectList is the wire format of a cluster list response.
type MKSClusterObjectList struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   PaaSListMetadata   `json:"metadata,omitempty"`
	Items      []MKSClusterObject `json:"items"`
}

// ToMKSCluster converts an SDK cluster to the wire format for apply. Status is dropped.
func ToMKSCluster(c *apiv1.MKSCluster, project, workspace string) *MKSClusterObject {
	if c == nil {
		return nil
	}
	cp := c.DeepCopyObject().(*apiv1.MKSCluster)
	return &MKSClusterObject{
		APIVersion: MKSAPIVersion,
		Kind:       MKSClusterKind,
		Metadata:   devMetadataToWire(cp.Metadata, project, workspace),
		Spec:       cp.Spec,
		Version:    cp.Version,
	}
}

// FromMKSCluster converts a wire cluster to the SDK type. The backend does not always echo
// metadata.project, so the request scope is the fallback for project and workspace.
func FromMKSCluster(w *MKSClusterObject, project, workspace string) *apiv1.MKSCluster {
	if w == nil {
		return nil
	}
	out := &apiv1.MKSCluster{
		TypeMeta: apiv1.TypeMeta{APIVersion: MKSAPIVersion, Kind: apiv1.KindMKSCluster},
		Metadata: devMetadataFromWire(w.Metadata, workspace),
		Spec:     w.Spec,
		Version:  w.Version,
	}
	if out.Metadata.Project == "" {
		out.Metadata.Project = project
	}
	if w.Status != nil {
		out.Status = *w.Status
	}
	return out
}

// FromMKSClusterList converts a wire list to the SDK list.
func FromMKSClusterList(list *MKSClusterObjectList, project, workspace string) *apiv1.MKSClusterList {
	out := &apiv1.MKSClusterList{
		TypeMeta: apiv1.TypeMeta{APIVersion: MKSAPIVersion, Kind: MKSClusterListKind},
		Items:    make([]apiv1.MKSCluster, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if c := FromMKSCluster(&list.Items[i], project, workspace); c != nil {
			out.Items = append(out.Items, *c)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}

// MKSClusterProfileObject is the wire format of a cluster profile.
type MKSClusterProfileObject struct {
	APIVersion string                         `json:"apiVersion"`
	Kind       string                         `json:"kind"`
	Metadata   DevMetadata                    `json:"metadata"`
	Spec       apiv1.MKSClusterProfileSpec    `json:"spec"`
	Status     *apiv1.MKSClusterProfileStatus `json:"status,omitempty"`
}

// MKSClusterProfileObjectList is the wire format of a profile list response.
type MKSClusterProfileObjectList struct {
	APIVersion string                    `json:"apiVersion"`
	Kind       string                    `json:"kind"`
	Metadata   PaaSListMetadata          `json:"metadata,omitempty"`
	Items      []MKSClusterProfileObject `json:"items"`
}

// FromMKSClusterProfile converts a wire profile to the SDK type.
func FromMKSClusterProfile(w *MKSClusterProfileObject, project string) *apiv1.MKSClusterProfile {
	if w == nil {
		return nil
	}
	out := &apiv1.MKSClusterProfile{
		TypeMeta: apiv1.TypeMeta{APIVersion: MKSAPIVersion, Kind: apiv1.KindMKSClusterProfile},
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

// FromMKSClusterProfileList converts a wire profile list to the SDK list.
func FromMKSClusterProfileList(list *MKSClusterProfileObjectList, project string) *apiv1.MKSClusterProfileList {
	out := &apiv1.MKSClusterProfileList{
		TypeMeta: apiv1.TypeMeta{APIVersion: MKSAPIVersion, Kind: MKSClusterProfileListKind},
		Items:    make([]apiv1.MKSClusterProfile, 0),
	}
	if list == nil {
		return out
	}
	for i := range list.Items {
		if p := FromMKSClusterProfile(&list.Items[i], project); p != nil {
			out.Items = append(out.Items, *p)
		}
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}

// MKSNodeObject is the wire format of a cluster node.
type MKSNodeObject struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   DevMetadata       `json:"metadata"`
	Spec       apiv1.MKSNodeSpec `json:"spec"`
	// Status is absent on older controllers.
	Status apiv1.MKSNodeStatus `json:"status,omitempty"`
}

// MKSNodeObjectList is the wire format of a node list response.
type MKSNodeObjectList struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Metadata   PaaSListMetadata `json:"metadata,omitempty"`
	Items      []MKSNodeObject  `json:"items"`
}

// FromMKSNodeList converts a wire node list to the SDK list.
func FromMKSNodeList(list *MKSNodeObjectList, project string) *apiv1.MKSNodeList {
	out := &apiv1.MKSNodeList{
		TypeMeta: apiv1.TypeMeta{APIVersion: MKSAPIVersion, Kind: MKSNodeListKind},
		Items:    make([]apiv1.MKSNode, 0),
	}
	if list == nil {
		return out
	}
	for _, w := range list.Items {
		n := apiv1.MKSNode{
			TypeMeta: apiv1.TypeMeta{APIVersion: MKSAPIVersion, Kind: apiv1.KindMKSNode},
			Metadata: devMetadataFromWire(w.Metadata, ""),
			Spec:     w.Spec,
			Status:   w.Status,
		}
		if n.Metadata.Project == "" {
			n.Metadata.Project = project
		}
		out.Items = append(out.Items, n)
	}
	out.Metadata.Continue = devListContinue(list.Metadata, len(list.Items))
	return out
}
