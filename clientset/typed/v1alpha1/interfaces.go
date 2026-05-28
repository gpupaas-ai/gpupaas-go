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
