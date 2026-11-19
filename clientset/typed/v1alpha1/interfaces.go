package v1alpha1

import (
	"context"
	"time"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// BareMetalInterface manages first-class bare-metal instances (dev.envmgmt.io) at project or
// workspace scope. This is the instance a developer asks for from a BmProfile (SKU).
//
// Create stores a new instance and fails with a 409 if the name is taken; Update applies a changed
// spec to an existing one. Both return as soon as the backend accepts the request, and the
// platform provisions the machine asynchronously (minutes): poll Get, or use WaitForBareMetal.
type BareMetalInterface interface {
	Create(ctx context.Context, obj *apiv1.BareMetal, opts gpupaas.CreateOptions) (*apiv1.BareMetal, error)
	Update(ctx context.Context, obj *apiv1.BareMetal, opts gpupaas.UpdateOptions) (*apiv1.BareMetal, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.BareMetal, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.BareMetalList, error)
	// Delete starts teardown; the instance stays visible (deleting) until it is gone.
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	// Action runs a lifecycle verb (start, stop, reboot, "Reinstall OS", ...). payload may be nil.
	// It returns once the backend accepts the action; poll Get for the result.
	Action(ctx context.Context, name, verb string, payload map[string]any) (*apiv1.BareMetal, error)
}

// VMInstanceInterface manages first-class virtual machines (dev.envmgmt.io) at project or workspace
// scope. This is the VM a developer asks for from a VMProfile (SKU): spec.type names the SKU, the size
// matches one of the SKU's presets, and fields are snake_case on the wire.
//
// Create stores a new VM and fails with a 409 if the name is taken; Update applies a changed spec to
// an existing one. Both return as soon as the backend accepts the request, and the platform provisions
// the VM asynchronously (minutes): poll Get, or use WaitForVMInstance.
type VMInstanceInterface interface {
	Create(ctx context.Context, obj *apiv1.VMInstance, opts gpupaas.CreateOptions) (*apiv1.VMInstance, error)
	Update(ctx context.Context, obj *apiv1.VMInstance, opts gpupaas.UpdateOptions) (*apiv1.VMInstance, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.VMInstance, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.VMInstanceList, error)
	// Delete starts teardown; the VM stays visible (deleting) until it is gone.
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	// Action runs a lifecycle verb (start, stop, reboot, ...). payload may be nil. It returns once the
	// backend accepts the action; poll Get for the result.
	Action(ctx context.Context, name, verb string, payload map[string]any) (*apiv1.VMInstance, error)
}

// VMProfileInterface reads the VM SKUs of a project.
type VMProfileInterface interface {
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.VMProfile, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.VMProfileList, error)
}

// BmProfileInterface reads the bare-metal SKUs of a project.
type BmProfileInterface interface {
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.BmProfile, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.BmProfileList, error)
}

// MKSClusterInterface manages first-class managed Kubernetes clusters (paas.k8smgmt.io/v3) at
// project or workspace scope.
//
// Clusters are declarative. Create stores a new cluster and fails if the name is taken;
// Update applies a changed spec (Kubernetes version, worker pools and their sizes) to an
// existing one. Both return as soon as the backend accepts the request; poll Get (see
// WaitForMKSCluster) until the cluster settles, because provisioning takes many minutes.
type MKSClusterInterface interface {
	Create(ctx context.Context, obj *apiv1.MKSCluster, opts gpupaas.CreateOptions) (*apiv1.MKSCluster, error)
	Update(ctx context.Context, obj *apiv1.MKSCluster, opts gpupaas.UpdateOptions) (*apiv1.MKSCluster, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.MKSCluster, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSClusterList, error)
	// Delete starts teardown; the cluster stays visible (destroying) until it is gone.
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	// Kubeconfig downloads the user kubeconfig of a running cluster (the console's "Download
	// Kubeconfig"). It comes from the sentry service, keyed by the cluster's edge id, which the
	// cluster only has once it is registered.
	Kubeconfig(ctx context.Context, name string) ([]byte, error)

	// The calls below are the dedicated day-2 routes of the API reference. Create and Update already
	// cover most changes (they are declarative); these are the narrower alternatives. Each returns the
	// cluster as the platform answers, which is usually still in the phase the operation started, so
	// wait with WaitForMKSCluster. They take the cluster's name only and always address the project
	// route: cluster names are unique per organization.

	// AddNodePool adds one worker pool (POST .../clusters/{cluster}/nodepools). A pool that leaves
	// assignPublicIp unset takes the value of the existing worker pools.
	AddNodePool(ctx context.Context, cluster string, pool apiv1.MKSNodeGroup) (*apiv1.MKSCluster, error)
	// RemoveNodePool removes a worker pool and its machines (DELETE .../clusters/{cluster}/nodepools/{pool}).
	RemoveNodePool(ctx context.Context, cluster, pool string) (*apiv1.MKSCluster, error)
	// Upgrade changes the Kubernetes version, the blueprint or the CNI (POST .../clusters/{cluster}/upgrade).
	// Send only what changes. The version moves one minor at a time and never down.
	Upgrade(ctx context.Context, cluster string, req apiv1.MKSUpgradeRequest) (*apiv1.MKSCluster, error)
	// UpdateMetadata edits labels, annotations and tags without redeploying the cluster
	// (PATCH .../clusters/{cluster}/metadata). It is refused only while the cluster is being deleted.
	UpdateMetadata(ctx context.Context, cluster string, patch apiv1.MKSMetadataPatch) (*apiv1.MKSCluster, error)
	// Redeploy starts provisioning again (POST .../clusters/{cluster}/redeploy). The platform allows it
	// only from ERROR or DEGRADED.
	Redeploy(ctx context.Context, cluster string) (*apiv1.MKSCluster, error)
	// Status reads the status document (GET .../clusters/{cluster}/status): the declared pools joined with
	// the nodes the platform reports, with their addresses and Kubernetes conditions.
	Status(ctx context.Context, cluster string) (*apiv1.MKSClusterStatusView, error)
	// AuditEvents lists the audit trail of the cluster, newest first by default.
	AuditEvents(ctx context.Context, cluster string, filter MKSAuditEventFilter) (*apiv1.MKSAuditEventList, error)
}

// MKSAuditEventFilter selects audit events. Empty fields do not filter.
type MKSAuditEventFilter struct {
	// Action is exact, for example "cluster.upgrade" or "node.drain".
	Action string
	// Result is accepted, rejected, completed or failed.
	Result string
	// ResourceType is cluster, controlPlane, nodePool, node or sku.
	ResourceType string
	Actor        string
	// From and To bound the event time, inclusive. The zero time leaves a side open.
	From, To time.Time
	// Limit is the page size (the platform caps it at 500); Offset is how many events to skip.
	Limit, Offset int
}

// MKSClusterProfileInterface reads Kubernetes SKUs (cluster profiles) of a project.
type MKSClusterProfileInterface interface {
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.MKSClusterProfile, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSClusterProfileList, error)
	// CreateOptions returns what the SKU allows for the control plane and the workers: SKUs, images,
	// shapes and datacenters (GET .../clusterprofiles/{name}/createoptions).
	CreateOptions(ctx context.Context, name string) (*apiv1.MKSCreateOptions, error)
	// AvailableCapacity returns the capacity left for the SKU (GET .../clusterprofiles/{name}/availablecapacity).
	// The response shape is not part of the API reference, so the document is returned as received.
	AvailableCapacity(ctx context.Context, name string) (*apiv1.MKSAvailableCapacity, error)
}

// MKSMatchQuery selects what the matchmaker is asked about. Which fields a call needs is noted on
// the method; unused fields are left out of the request.
type MKSMatchQuery struct {
	// Datacenters is required by every call but KubernetesVersions.
	Datacenters []string
	// SKUKinds is bmaas, vmaas or both.
	SKUKinds []string
	// ComputeTypes is cpu, gpu or custom, one or several. Custom counts as cpu and is sent as cpu.
	ComputeTypes []string
	// Images are image resource names (from AvailableOS), for AvailableSpecs, MatchSKUs and AvailableComputeSKUs.
	Images []string
	// CPU, RAMGb and StorageGb are the shape MatchSKUs looks for; GPUType is required there when
	// ComputeTypes includes gpu, and GPUCount narrows it.
	CPU, RAMGb, StorageGb int
	GPUType               string
	GPUCount              int
}

// MKSMatchmakerInterface asks the matchmaker what a datacenter offers. The calls are read-only and
// drive the pickers of the console; they help to choose compute SKUs, images and Kubernetes
// versions for a cluster or a Kubernetes SKU.
type MKSMatchmakerInterface interface {
	// KubernetesVersions lists the Kubernetes minors the controller offers, newest first.
	KubernetesVersions(ctx context.Context) (*apiv1.MKSKubernetesVersions, error)
	// AvailableOS lists operating systems and their versions. Needs Datacenters, SKUKinds and ComputeTypes.
	AvailableOS(ctx context.Context, q MKSMatchQuery) (*apiv1.MKSAvailableOS, error)
	// AvailableImages lists the images the matching SKUs offer. Needs Datacenters, SKUKinds and ComputeTypes.
	AvailableImages(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSImageOption, error)
	// AvailableSpecs lists the machine shapes offered with the images. Also needs Images.
	AvailableSpecs(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSSpecOption, error)
	// MatchSKUs lists the compute SKUs that satisfy a shape: CPU, RAMGb, StorageGb, and GPUType for gpu.
	MatchSKUs(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSSKURef, error)
	// AvailableComputeSKUs lists the selectable compute SKUs with their shapes. Needs Datacenters,
	// SKUKinds, ComputeTypes and Images.
	AvailableComputeSKUs(ctx context.Context, q MKSMatchQuery) ([]apiv1.MKSInstanceType, error)
}

// MKSNodeInterface lists the nodes a cluster has registered.
type MKSNodeInterface interface {
	// ListForCluster returns the nodes of the named cluster.
	ListForCluster(ctx context.Context, cluster string) (*apiv1.MKSNodeList, error)
	// CordonPool marks every node of a worker pool unschedulable.
	CordonPool(ctx context.Context, cluster, pool string) (*MKSPoolOperationResult, error)
	// UncordonPool makes the nodes of a worker pool schedulable again.
	UncordonPool(ctx context.Context, cluster, pool string) (*MKSPoolOperationResult, error)
	// DrainPool evicts the workloads from every node of a worker pool.
	DrainPool(ctx context.Context, cluster, pool string, opts MKSDrainOptions) (*MKSPoolOperationResult, error)

	// The calls below act on one node, named by its hostname (the Kubernetes node name).

	// CordonNode marks a node unschedulable.
	CordonNode(ctx context.Context, cluster, hostname string) error
	// UncordonNode makes a node schedulable again.
	UncordonNode(ctx context.Context, cluster, hostname string) error
	// DrainNode evicts the workloads of a node. The platform takes no drain options for one node.
	DrainNode(ctx context.Context, cluster, hostname string) error
	// DeleteNode removes a node from the cluster.
	DeleteNode(ctx context.Context, cluster, hostname string) error
	// UpdateNodeMetadata sets and removes labels and annotations on the live Kubernetes node.
	UpdateNodeMetadata(ctx context.Context, cluster, hostname string, patch MKSNodeMetadataPatch) error
	// UpdateNodeTaints replaces all taints of a node; an empty list clears them.
	UpdateNodeTaints(ctx context.Context, cluster, hostname string, taints []apiv1.MKSTaint) error
}

// MKSNodeMetadataPatch is a per-key change of a node's labels and annotations. A key in both the set and the
// remove list of one kind is refused by the platform.
type MKSNodeMetadataPatch struct {
	SetLabels         map[string]string
	RemoveLabels      []string
	SetAnnotations    map[string]string
	RemoveAnnotations []string
}

// MKSPoolOperationResult is the platform's answer to a pool operation: how many nodes it tried and
// what each one said. The per-node flag is not reliable on its own: the platform has answered
// "succeeded" for a node whose message was a kubectl connection error, so callers should read the messages.
type MKSPoolOperationResult struct {
	ClusterName string              `json:"clusterName"`
	NodePool    string              `json:"nodePool"`
	Action      string              `json:"action"`
	Requested   int                 `json:"requested"`
	Succeeded   int                 `json:"succeeded"`
	Failed      int                 `json:"failed"`
	Results     []MKSPoolNodeResult `json:"results"`
}

// MKSPoolNodeResult is the outcome of a pool operation on one node.
type MKSPoolNodeResult struct {
	Hostname  string `json:"hostname"`
	Succeeded bool   `json:"succeeded"`
	Message   string `json:"message"`
}

// MKSDrainOptions are the optional settings of a pool drain. A nil field is left out of the
// request, so the platform's own default applies.
type MKSDrainOptions struct {
	Force              *bool
	IgnoreDaemonsets   *bool
	GracePeriodSeconds *int
	DeleteEmptyDirData *bool
}

// WorkspaceInterface manages the workspaces of one project and hands out the clients of the resources
// that live in a workspace.
//
// Create stores a new workspace and fails with a 409 if the name is taken; Update applies changed
// metadata or spec to an existing one. Delete fails while the workspace still holds resources.
type WorkspaceInterface interface {
	Create(ctx context.Context, obj *apiv1.Workspace, opts gpupaas.CreateOptions) (*apiv1.Workspace, error)
	Update(ctx context.Context, obj *apiv1.Workspace, opts gpupaas.UpdateOptions) (*apiv1.Workspace, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.Workspace, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.WorkspaceList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error

	MKSClusters(workspace string) MKSClusterInterface
	BareMetals(workspace string) BareMetalInterface
	VMInstances(workspace string) VMInstanceInterface
	ComputeInstances(workspace string) ComputeInstanceInterface
	SshKeys(workspace string) SshKeyInterface
	SecurityGroups(workspace string) SecurityGroupInterface
	Storages(workspace string) StorageInterface
}

// SshKeyInterface manages SSH keys at project or workspace scope. The platform has no update for a
// key: create it, and delete it to replace it. Create stores a new key and fails with a 409 if the
// name is taken.
type SshKeyInterface interface {
	Create(ctx context.Context, obj *apiv1.SshKey, opts gpupaas.CreateOptions) (*apiv1.SshKey, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.SshKey, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.SshKeyList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// SecurityGroupInterface manages security groups at project or workspace scope. The platform has no
// update for a group: create it, and delete it to replace it. Create stores a new group and fails
// with a 409 if the name is taken.
type SecurityGroupInterface interface {
	Create(ctx context.Context, obj *apiv1.SecurityGroup, opts gpupaas.CreateOptions) (*apiv1.SecurityGroup, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.SecurityGroup, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.SecurityGroupList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// StorageInterface manages high-speed and object storage volumes at project or workspace scope (one Storage
// kind; the profile in Spec.Type decides the family, and List can filter by ListOptions.ServiceType).
// Create stores a new volume and fails with a 409 if the name is taken; Update applies a changed spec to an
// existing one. Provisioning, update and delete are asynchronous: follow them with WaitForStorage.
type StorageInterface interface {
	Create(ctx context.Context, obj *apiv1.Storage, opts gpupaas.CreateOptions) (*apiv1.Storage, error)
	Update(ctx context.Context, obj *apiv1.Storage, opts gpupaas.UpdateOptions) (*apiv1.Storage, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.Storage, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.StorageList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// ComputeInstanceInterface reads ComputeInstance resources (paas.envmgmt.io), read-only.
//
// Every first-class VM, bare-metal machine and MKS cluster or node has a compute instance of the
// same name behind it. The platform hides these instances from normal listings, so the client asks
// for them explicitly (include_first_class=true). They hold what the first-class record does not:
// the workflow error and the provisioning log of a failure, and the teardown that outlives the
// record on delete. Provisioning itself is asynchronous; read the instance to follow it.
type ComputeInstanceInterface interface {
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.ComputeInstance, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.ComputeInstanceList, error)

	// FailureLog returns the WARNING and ERROR lines (at most maxLines, newest
	// last) of the failed activities of the last run. status.reason is often only
	// "Failed to deploy"; the log holds the cause (for example an unreachable
	// host). It returns an empty log, not an error, when the run has no
	// environment or no failed activity. Log text can name hosts and addresses;
	// show it to the operator but do not store it.
	FailureLog(ctx context.Context, name string, maxLines int) (*apiv1.ComputeFailureLog, error)
}
