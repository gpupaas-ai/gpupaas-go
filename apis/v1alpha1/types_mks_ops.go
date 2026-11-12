package v1alpha1

import (
	"encoding/json"
	"strings"
)

// Request and response types of the MKS day-2, status, audit and discovery routes of the v3.1.41
// API reference (paas.k8smgmt.io/v3). The cluster, profile and node types live in types_mks.go.

// ---------------------------------------------------------------------------
// Day-2 requests
// ---------------------------------------------------------------------------

// MKSUpgradeRequest is the body of POST .../clusters/{cluster}/upgrade. Send only what changes. The
// Kubernetes version moves one minor version at a time and never down; the valid targets are in
// MKSClusterStatus.AvailableKubernetesVersions.
type MKSUpgradeRequest struct {
	// K8sVersion is "v1.36.4" or the minor form "1.36", which the platform resolves to its latest patch.
	K8sVersion       string  `json:"k8sVersion,omitempty"`
	Blueprint        string  `json:"blueprint,omitempty"`
	BlueprintVersion string  `json:"blueprintVersion,omitempty"`
	CNI              *MKSCNI `json:"cni,omitempty"`
}

// MKSMetadataPatch is the body of PATCH .../clusters/{cluster}/metadata, the fast path that edits
// labels, annotations and tags without redeploying the cluster. A nil map leaves that field alone. A
// non-nil map is merged into the stored one, or replaces it when the matching Replace flag is set; an
// empty non-nil map with the flag set clears the field.
type MKSMetadataPatch struct {
	Labels             map[string]string
	Annotations        map[string]string
	Tags               map[string]string
	ReplaceLabels      bool
	ReplaceAnnotations bool
	ReplaceTags        bool
}

// MarshalJSON writes a map that is present, even when empty, and leaves out one that is nil.
func (p MKSMetadataPatch) MarshalJSON() ([]byte, error) {
	out := map[string]any{}
	if p.Labels != nil {
		out["labels"] = p.Labels
	}
	if p.Annotations != nil {
		out["annotations"] = p.Annotations
	}
	if p.Tags != nil {
		out["tags"] = p.Tags
	}
	if p.ReplaceLabels {
		out["replaceLabels"] = true
	}
	if p.ReplaceAnnotations {
		out["replaceAnnotations"] = true
	}
	if p.ReplaceTags {
		out["replaceTags"] = true
	}
	return json.Marshal(out)
}

// ---------------------------------------------------------------------------
// Cluster status document (GET .../clusters/{cluster}/status)
// ---------------------------------------------------------------------------

// MKSClusterHealth is the health of the control plane, of each worker pool and overall, as one of
// MKS_HEALTH_STATE_HEALTHY, MKS_HEALTH_STATE_UNHEALTHY, MKS_HEALTH_STATE_DEGRADED or MKS_HEALTH_STATE_UNKNOWN.
type MKSClusterHealth struct {
	ControlPlane string            `json:"controlPlane,omitempty"`
	NodePools    map[string]string `json:"nodePools,omitempty"`
	Overall      string            `json:"overall,omitempty"`
}

// MKSClusterStatusView is the status document: the declared pools joined with the nodes the platform
// reports behind them. It is assembled per request and is not the cluster resource.
type MKSClusterStatusView struct {
	Metadata MKSStatusViewMetadata `json:"metadata"`
	Spec     MKSStatusViewSpec     `json:"spec"`
	Status   MKSStatusViewStatus   `json:"status"`
}

// MKSStatusViewMetadata identifies the cluster.
type MKSStatusViewMetadata struct {
	Name        string            `json:"name,omitempty"`
	Project     string            `json:"project,omitempty"`
	Workspace   string            `json:"workspace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// MKSStatusViewSpec is the cluster-shaped half of the status document.
type MKSStatusViewSpec struct {
	KubernetesVersion string            `json:"kubernetesVersion,omitempty"`
	Location          string            `json:"location,omitempty"`
	Blueprint         *MKSBlueprint     `json:"blueprint,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	ControlPlane      *MKSPoolStatus    `json:"controlPlane,omitempty"`
	WorkerPools       []MKSPoolStatus   `json:"workerPools,omitempty"`
}

// MKSPoolStatus is one pool: its declared shape plus the nodes the platform currently reports in it.
type MKSPoolStatus struct {
	Name           string            `json:"name,omitempty"`
	SKUName        string            `json:"skuName,omitempty"`
	SKUType        string            `json:"skuType,omitempty"`
	ComputeType    string            `json:"computeType,omitempty"`
	SKUDisplayName string            `json:"skuDisplayName,omitempty"`
	Datacenter     string            `json:"datacenter,omitempty"`
	OSImage        string            `json:"osImage,omitempty"`
	Dedicated      bool              `json:"dedicated,omitempty"`
	Scaling        *MKSScaling       `json:"scaling,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	Taints         []MKSTaint        `json:"taints,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	// Status is the pool's lifecycle: CREATING, ACTIVE, UPDATING, DELETING, CREATE_FAILED or DELETE_FAILED.
	// It is not machine health: a pool of unwell nodes is still ACTIVE.
	Status   string                `json:"status,omitempty"`
	Nodes    []MKSNodeStatusView   `json:"nodes,omitempty"`
	Platform *MKSPlatformStatusRef `json:"platform,omitempty"`
}

// MKSNodeStatusView is one node as the platform reports it.
type MKSNodeStatusView struct {
	Hostname    string            `json:"hostname,omitempty"`
	PrivateIP   string            `json:"private_ip,omitempty"`
	KubeVersion string            `json:"kube_version,omitempty"`
	OS          string            `json:"os,omitempty"`
	Arch        string            `json:"arch,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Taints      []MKSTaint        `json:"taints,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	// Roles are "Master" and "Worker".
	Roles []string `json:"roles,omitempty"`
	// Status is the platform's own word for the node, for example READY or PROVISIONING.
	Status string `json:"status,omitempty"`
	// State holds the Kubernetes node conditions (DiskPressure, MemoryPressure, PIDPressure,
	// NetworkUnavailable, ClusterNodeReady) as "true", "false" or "unknown".
	State map[string]string `json:"state,omitempty"`
}

// MKSPlatformStatusRef is the platform build installed on a pool's nodes.
type MKSPlatformStatusRef struct {
	Version string `json:"version,omitempty"`
	// Components maps a component's display name to its version.
	Components map[string]string `json:"components,omitempty"`
}

// MKSStatusViewStatus is the lifecycle state of the cluster in the status document.
type MKSStatusViewStatus struct {
	Status                      string            `json:"status,omitempty"`
	Reason                      string            `json:"reason,omitempty"`
	Action                      string            `json:"action,omitempty"`
	Health                      *MKSClusterHealth `json:"health,omitempty"`
	AvailableKubernetesVersions []string          `json:"availableKubernetesVersions,omitempty"`
}

// ---------------------------------------------------------------------------
// Audit events (GET .../clusters/{cluster}/auditevents)
// ---------------------------------------------------------------------------

// MKS audit event results.
const (
	MKSAuditAccepted  = "accepted"
	MKSAuditRejected  = "rejected"
	MKSAuditCompleted = "completed"
	MKSAuditFailed    = "failed"
)

// MKSAuditEvent is one row of a cluster's audit trail: who asked for what and how it ended.
type MKSAuditEvent struct {
	ID string `json:"id,omitempty"`
	// ResourceType is cluster, controlPlane, nodePool, node or sku.
	ResourceType string `json:"resourceType,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
	// Action is a dotted verb such as cluster.apply, cluster.upgrade, cluster.nodepool.create or node.drain.
	Action  string `json:"action,omitempty"`
	Actor   string `json:"actor,omitempty"`
	Message string `json:"message,omitempty"`
	// Timestamp is in Unix seconds. The platform writes 64-bit integers as strings, so both forms are read.
	Timestamp MKSInt64 `json:"timestamp,omitempty"`
	// Result is accepted, rejected, completed or failed.
	Result      string `json:"result,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
	ClusterID   string `json:"clusterId,omitempty"`
	// OperationID is the day-2 operation the event belongs to; empty for events that start none.
	OperationID string `json:"operationId,omitempty"`
}

// MKSAuditEventList is one page of audit events. Count is the total matching the filters, not the
// size of the page.
type MKSAuditEventList struct {
	Items []MKSAuditEvent
	Count int64
}

// UnmarshalJSON reads the list under items, auditEvents or audit_events, whichever the platform uses,
// and the total from metadata.count.
func (l *MKSAuditEventList) UnmarshalJSON(b []byte) error {
	var raw struct {
		Items       []MKSAuditEvent `json:"items"`
		AuditEvents []MKSAuditEvent `json:"auditEvents"`
		Snake       []MKSAuditEvent `json:"audit_events"`
		Metadata    struct {
			Count MKSInt64 `json:"count"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	switch {
	case len(raw.Items) > 0:
		l.Items = raw.Items
	case len(raw.AuditEvents) > 0:
		l.Items = raw.AuditEvents
	default:
		l.Items = raw.Snake
	}
	l.Count = int64(raw.Metadata.Count)
	if l.Count == 0 {
		l.Count = int64(len(l.Items))
	}
	return nil
}

// MKSInt64 is an integer that the platform may write as a JSON number or, for 64-bit protobuf fields, as a
// quoted string. It is always written as a number.
type MKSInt64 int64

// UnmarshalJSON accepts 60 and "60".
func (n *MKSInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	i, err := json.Number(s).Int64()
	if err != nil {
		return err
	}
	*n = MKSInt64(i)
	return nil
}

// ---------------------------------------------------------------------------
// Matchmaker and SKU discovery
// ---------------------------------------------------------------------------

// MKSKubernetesVersion is one Kubernetes minor the controller offers.
type MKSKubernetesVersion struct {
	// Minor is the form accepted as spec.kubernetesVersion, for example "1.36".
	Minor string `json:"minor,omitempty"`
	// LatestPatch is what the minor resolves to, for example "v1.36.4".
	LatestPatch string `json:"latestPatch,omitempty"`
	// IsDefault marks the version the controller preselects. At most one entry has it.
	IsDefault bool `json:"isDefault,omitempty"`
}

// MKSKubernetesVersions is the answer of the kubernetesversions matchmaker call.
type MKSKubernetesVersions struct {
	// Minors are the offered minors, newest first.
	Minors   []string               `json:"minors,omitempty"`
	Versions []MKSKubernetesVersion `json:"versions,omitempty"`
}

// MKSBootDiskRange is the boot disk window of a virtual-machine SKU.
type MKSBootDiskRange struct {
	MinGb       int32  `json:"minGb,omitempty"`
	MaxGb       int32  `json:"maxGb,omitempty"`
	IncrementGb int32  `json:"incrementGb,omitempty"`
	Unit        string `json:"unit,omitempty"`
}

// MKSImageOption is an operating system image a matching SKU offers.
type MKSImageOption struct {
	Name string `json:"name,omitempty"`
	// Default is true when a candidate SKU marks the image as its default.
	Default bool `json:"default,omitempty"`
}

// MKSSpecOption is a machine shape (cpu, memory, storage and optionally a GPU) that at least one SKU offers.
type MKSSpecOption struct {
	SKUKind        string            `json:"skuKind,omitempty"`
	ComputeType    string            `json:"computeType,omitempty"`
	CPU            int32             `json:"cpu,omitempty"`
	RAMGb          int32             `json:"ramGb,omitempty"`
	StorageGb      int32             `json:"storageGb,omitempty"`
	GPUType        string            `json:"gpuType,omitempty"`
	GPUCount       int32             `json:"gpuCount,omitempty"`
	GPUMemoryGb    int32             `json:"gpuMemoryGb,omitempty"`
	BootDiskRange  *MKSBootDiskRange `json:"bootDiskRange,omitempty"`
	HardwareModels []string          `json:"hardwareModels,omitempty"`
	Datacenters    []string          `json:"datacenters,omitempty"`
	SKUCount       int32             `json:"skuCount,omitempty"`
	PresetName     string            `json:"presetName,omitempty"`
	GPUVendor      string            `json:"gpuVendor,omitempty"`
	// SKUNames are the compute SKUs that satisfy the shape.
	SKUNames    []string `json:"skuNames,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	DisplayText string   `json:"displayText,omitempty"`
}

// MKSSKURef is a compute SKU that matches a requested shape.
type MKSSKURef struct {
	Name          string            `json:"name,omitempty"`
	ID            string            `json:"id,omitempty"`
	Kind          string            `json:"kind,omitempty"`
	CPU           int32             `json:"cpu,omitempty"`
	RAMGb         int32             `json:"ramGb,omitempty"`
	StorageGb     int32             `json:"storageGb,omitempty"`
	ImageDefault  string            `json:"imageDefault,omitempty"`
	Datacenters   []string          `json:"datacenters,omitempty"`
	GPUType       string            `json:"gpuType,omitempty"`
	GPUCount      int32             `json:"gpuCount,omitempty"`
	GPUMemoryGb   int32             `json:"gpuMemoryGb,omitempty"`
	BootDiskRange *MKSBootDiskRange `json:"bootDiskRange,omitempty"`
	DisplayLabel  string            `json:"displayLabel,omitempty"`
}

// MKSOSImageVersion is one version of an operating system.
type MKSOSImageVersion struct {
	Version string `json:"version,omitempty"`
	// ImageName is the image resource name a pool's osImage takes.
	ImageName string `json:"imageName,omitempty"`
}

// MKSOSImageGroup is an operating system with its versions.
type MKSOSImageGroup struct {
	Name     string              `json:"name,omitempty"`
	Versions []MKSOSImageVersion `json:"versions,omitempty"`
}

// MKSAvailableOS is the answer of the availableos matchmaker call.
type MKSAvailableOS struct {
	OSImages []MKSOSImageGroup `json:"osImages,omitempty"`
	// GPUs are the catalog GPU names the matching bare-metal SKUs advertise (compute type gpu only).
	GPUs []string `json:"gpus,omitempty"`
}

// MKSInstanceType is a selectable compute SKU with the shapes it offers.
type MKSInstanceType struct {
	Name         string          `json:"name,omitempty"`
	Kind         string          `json:"kind,omitempty"`
	ComputeType  string          `json:"computeType,omitempty"`
	DisplayLabel string          `json:"displayLabel,omitempty"`
	Images       []string        `json:"images,omitempty"`
	Datacenters  []string        `json:"datacenters,omitempty"`
	Shapes       []MKSSpecOption `json:"shapes,omitempty"`
}

// MKSPoolOptions is what a Kubernetes SKU allows for the control plane or the workers.
type MKSPoolOptions struct {
	Name          string            `json:"name,omitempty"`
	DisplayName   string            `json:"displayName,omitempty"`
	Description   string            `json:"description,omitempty"`
	SKUKind       []string          `json:"skuKind,omitempty"`
	ComputeType   []string          `json:"computeType,omitempty"`
	Datacenters   []string          `json:"datacenters,omitempty"`
	OSImages      []MKSOSImageGroup `json:"osImages,omitempty"`
	InstanceTypes []MKSInstanceType `json:"instanceTypes,omitempty"`
	SKUNames      []string          `json:"skuNames,omitempty"`
	Images        []string          `json:"images,omitempty"`
	CPU           int32             `json:"cpu,omitempty"`
	RAMGb         int32             `json:"ramGb,omitempty"`
	StorageGb     int32             `json:"storageGb,omitempty"`
	GPUType       string            `json:"gpuType,omitempty"`
	GPUCount      int32             `json:"gpuCount,omitempty"`
	GPUMemoryGb   int32             `json:"gpuMemoryGb,omitempty"`
}

// MKSCreateOptions is the answer of GET .../clusterprofiles/{name}/createoptions: the options a
// Kubernetes SKU allows when a cluster is created from it.
type MKSCreateOptions struct {
	ControlPlane               *MKSPoolOptions `json:"controlPlane,omitempty"`
	WorkerNode                 *MKSPoolOptions `json:"workerNode,omitempty"`
	DataCenterRegions          []string        `json:"dataCenterRegions,omitempty"`
	DefaultKubernetesVersion   string          `json:"defaultKubernetesVersion,omitempty"`
	SupportedKubernetesVersion []string        `json:"supportedKubernetesVersions,omitempty"`
	EnableOIDC                 bool            `json:"enableOidc,omitempty"`
}

// MKSAvailableCapacity is the answer of GET .../clusterprofiles/{name}/availablecapacity. The response
// shape is not part of the API reference, so the document is kept as received.
type MKSAvailableCapacity struct {
	Raw json.RawMessage
}

// UnmarshalJSON keeps the document as received.
func (c *MKSAvailableCapacity) UnmarshalJSON(b []byte) error {
	c.Raw = append(c.Raw[:0], b...)
	return nil
}

// MarshalJSON writes the document as received.
func (c MKSAvailableCapacity) MarshalJSON() ([]byte, error) {
	if len(c.Raw) == 0 {
		return []byte("null"), nil
	}
	return c.Raw, nil
}
