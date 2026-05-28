package v1alpha1

import (
	"context"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
)

// ProjectInterface manages Project resources.
type ProjectInterface interface {
	Create(ctx context.Context, obj *apiv1.Project, opts gpupaas.CreateOptions) (*apiv1.Project, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.Project, error)
	Update(ctx context.Context, obj *apiv1.Project, opts gpupaas.UpdateOptions) (*apiv1.Project, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.ProjectList, error)
}

// WorkspaceInterface manages Workspace resources within a project.
type WorkspaceInterface interface {
	Create(ctx context.Context, obj *apiv1.Workspace, opts gpupaas.CreateOptions) (*apiv1.Workspace, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.Workspace, error)
	Update(ctx context.Context, obj *apiv1.Workspace, opts gpupaas.UpdateOptions) (*apiv1.Workspace, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.WorkspaceList, error)
	Collaborators(workspace string) WorkspaceCollaboratorInterface
	VirtualMachines(workspace string) VirtualMachineInterface
	Storages(workspace string) StorageInterface
	SecurityGroups(workspace string) SecurityGroupInterface
	SshKeys(workspace string) SshKeyInterface
}

// WorkspaceCollaboratorInterface manages collaborators on a workspace.
type WorkspaceCollaboratorInterface interface {
	Create(ctx context.Context, obj *apiv1.WorkspaceCollaborator, opts gpupaas.CreateOptions) (*apiv1.WorkspaceCollaborator, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.WorkspaceCollaborator, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.WorkspaceCollaboratorList, error)
}

// VirtualMachineInterface manages virtual machines at project or workspace scope.
type VirtualMachineInterface interface {
	Create(ctx context.Context, obj *apiv1.VirtualMachine, opts gpupaas.CreateOptions) (*apiv1.VirtualMachine, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.VirtualMachine, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.VirtualMachineList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
	GetStatus(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.VirtualMachine, error)
	Start(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error)
	Stop(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error)
	// Reboot executes the "reboot" action on a virtual machine.
	Reboot(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error)
	// Action executes an arbitrary action verb (e.g. "start", "stop", "reboot",
	// or backend-specific verbs) on a virtual machine. It is the generic form
	// of Start / Stop / Reboot for callers that need verbs not exposed as
	// dedicated methods.
	Action(ctx context.Context, name, action string, opts gpupaas.ActionOptions) (*apiv1.VirtualMachine, error)
}

// StorageInterface manages storage resources at project or workspace scope.
type StorageInterface interface {
	Create(ctx context.Context, obj *apiv1.Storage, opts gpupaas.CreateOptions) (*apiv1.Storage, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.Storage, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.StorageList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// SecurityGroupInterface manages security groups at project or workspace scope.
type SecurityGroupInterface interface {
	Create(ctx context.Context, obj *apiv1.SecurityGroup, opts gpupaas.CreateOptions) (*apiv1.SecurityGroup, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.SecurityGroup, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.SecurityGroupList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// SshKeyInterface manages SSH keys at project or workspace scope.
type SshKeyInterface interface {
	Create(ctx context.Context, obj *apiv1.SshKey, opts gpupaas.CreateOptions) (*apiv1.SshKey, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.SshKey, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.SshKeyList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// BaremetalMachineInterface manages BaremetalMachine resources at project scope.
//
// CRUD endpoints are project-only; baremetal machines do not have a workspace
// scope. Sub-actions (PowerOn, PowerOff, Reboot, Provision, ReinstallOS,
// CreateConsoleSession) return the updated resource where applicable, and
// GetStatusInfo returns the free-form BaremetalMachineInfo payload.
type BaremetalMachineInterface interface {
	Create(ctx context.Context, obj *apiv1.BaremetalMachine, opts gpupaas.CreateOptions) (*apiv1.BaremetalMachine, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.BaremetalMachine, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.BaremetalMachineList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error

	// PowerOn requests the machine to be powered on.
	PowerOn(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error)
	// PowerOff requests the machine to be powered off.
	PowerOff(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error)
	// Reboot requests the machine to be rebooted.
	Reboot(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error)
	// Provision triggers a deploy of the machine with its current spec.
	Provision(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error)
	// ReinstallOS reimages the machine with the supplied image.
	ReinstallOS(ctx context.Context, name string, image *apiv1.BaremetalImage, opts gpupaas.ActionOptions) (*apiv1.BaremetalMachine, error)
	// CreateConsoleSession opens a short-lived SOL console session and returns
	// a WebSocket URL for the caller to connect to.
	CreateConsoleSession(ctx context.Context, name string, req *apiv1.BaremetalConsoleSessionRequest, opts gpupaas.ActionOptions) (*apiv1.BaremetalConsoleSession, error)
	// GetStatusInfo returns runtime info for the machine (hardware, network,
	// etc.). This is distinct from the resource's embedded Status field.
	GetStatusInfo(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.BaremetalMachineInfo, error)
}

// MKSClusterInterface manages MKSCluster resources at project scope, along with
// cluster-scoped sub-resources (nodes, worker node groups, audit events) and
// imperative sub-actions (Upgrade, ScaleNodeGroup, AddNodeGroup, RemoveNodeGroup).
type MKSClusterInterface interface {
	Create(ctx context.Context, obj *apiv1.MKSCluster, opts gpupaas.CreateOptions) (*apiv1.MKSCluster, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.MKSCluster, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSClusterList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error

	// Upgrade triggers a Kubernetes (and optional platform) version upgrade.
	Upgrade(ctx context.Context, name string, req *apiv1.MKSUpgradeRequest, opts gpupaas.ActionOptions) (*apiv1.MKSCluster, error)
	// ScaleNodeGroup adjusts the size of an existing worker node group.
	ScaleNodeGroup(ctx context.Context, name string, req *apiv1.MKSScaleNodeGroupRequest, opts gpupaas.ActionOptions) (*apiv1.MKSCluster, error)
	// AddNodeGroup adds a worker node group to the cluster.
	AddNodeGroup(ctx context.Context, name string, nodeGroup *apiv1.MKSNodeGroup, opts gpupaas.ActionOptions) (*apiv1.MKSCluster, error)
	// RemoveNodeGroup removes a worker node group from the cluster.
	RemoveNodeGroup(ctx context.Context, name, nodeGroupName string, opts gpupaas.ActionOptions) (*apiv1.MKSCluster, error)

	// Nodes returns a client for nodes within the named cluster.
	Nodes(clusterName string) MKSNodeInterface
	// WorkerNodeGroups returns a client for worker node groups within the named cluster.
	WorkerNodeGroups(clusterName string) MKSWorkerNodeGroupInterface
	// AuditEvents returns a read-only client for audit events within the named cluster.
	AuditEvents(clusterName string) MKSAuditEventInterface
}

// MKSNodeInterface manages MKSNode resources within a cluster.
type MKSNodeInterface interface {
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.MKSNode, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSNodeList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error

	// Drain evicts workloads from a node.
	Drain(ctx context.Context, name string, req *apiv1.MKSDrainRequest, opts gpupaas.ActionOptions) (*apiv1.MKSNode, error)
	// Cordon marks a node unschedulable.
	Cordon(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.MKSNode, error)
	// Uncordon marks a node schedulable.
	Uncordon(ctx context.Context, name string, opts gpupaas.ActionOptions) (*apiv1.MKSNode, error)
}

// MKSWorkerNodeGroupInterface manages MKSWorkerNodeGroup resources within a cluster.
type MKSWorkerNodeGroupInterface interface {
	Create(ctx context.Context, obj *apiv1.MKSWorkerNodeGroup, opts gpupaas.CreateOptions) (*apiv1.MKSWorkerNodeGroup, error)
	Get(ctx context.Context, name string, opts gpupaas.GetOptions) (*apiv1.MKSWorkerNodeGroup, error)
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSWorkerNodeGroupList, error)
	Delete(ctx context.Context, name string, opts gpupaas.DeleteOptions) error
}

// MKSAuditEventInterface provides read-only access to MKS audit events within a cluster.
type MKSAuditEventInterface interface {
	List(ctx context.Context, opts gpupaas.ListOptions) (*apiv1.MKSAuditEventList, error)
	Get(ctx context.Context, id string, opts gpupaas.GetOptions) (*apiv1.MKSAuditEvent, error)
}
