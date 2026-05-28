package remote

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// Backend implements backend.Backend over the gpupaas REST API.
type Backend struct {
	client clientset.Interface
	scheme *runtime.Scheme
}

// New creates a remote backend.
func New(cfg gpupaas.Config, scheme *runtime.Scheme) *Backend {
	cfg.Normalize()
	cs, err := clientset.NewForConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("gpupaas-go: remote backend: %v", err))
	}
	return &Backend{client: cs, scheme: scheme}
}

func (b *Backend) Apply(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	switch o := obj.(type) {
	case *v1alpha1.Project:
		existing, err := b.client.V1alpha1().Projects().Get(ctx, o.Metadata.Name, gpupaas.GetOptions{})
		if gpupaas.IsNotFound(err) {
			return b.client.V1alpha1().Projects().Create(ctx, o, gpupaas.CreateOptions{})
		}
		if err != nil {
			return nil, err
		}
		o.Status = existing.Status
		return b.client.V1alpha1().Projects().Update(ctx, o, gpupaas.UpdateOptions{})
	case *v1alpha1.Workspace:
		ws := b.client.V1alpha1().Workspaces(o.Metadata.Project)
		return ws.Create(ctx, o, gpupaas.CreateOptions{})
	case *v1alpha1.WorkspaceCollaborator:
		collab := b.client.V1alpha1().Workspaces(o.Metadata.Project).Collaborators(o.Metadata.Workspace)
		return collab.Create(ctx, o, gpupaas.CreateOptions{})
	case *v1alpha1.VirtualMachine:
		if o.Metadata.Workspace != "" {
			vms := b.client.V1alpha1().Workspaces(o.Metadata.Project).VirtualMachines(o.Metadata.Workspace)
			return vms.Create(ctx, o, gpupaas.CreateOptions{})
		}
		vms := b.client.V1alpha1().VirtualMachines(o.Metadata.Project)
		return vms.Create(ctx, o, gpupaas.CreateOptions{})
	case *v1alpha1.Storage:
		if o.Metadata.Workspace != "" {
			st := b.client.V1alpha1().Workspaces(o.Metadata.Project).Storages(o.Metadata.Workspace)
			return st.Create(ctx, o, gpupaas.CreateOptions{})
		}
		st := b.client.V1alpha1().Storages(o.Metadata.Project)
		return st.Create(ctx, o, gpupaas.CreateOptions{})
	case *v1alpha1.SecurityGroup:
		if o.Metadata.Workspace != "" {
			sg := b.client.V1alpha1().Workspaces(o.Metadata.Project).SecurityGroups(o.Metadata.Workspace)
			return sg.Create(ctx, o, gpupaas.CreateOptions{})
		}
		sg := b.client.V1alpha1().SecurityGroups(o.Metadata.Project)
		return sg.Create(ctx, o, gpupaas.CreateOptions{})
	case *v1alpha1.SshKey:
		if o.Metadata.Workspace != "" {
			keys := b.client.V1alpha1().Workspaces(o.Metadata.Project).SshKeys(o.Metadata.Workspace)
			return keys.Create(ctx, o, gpupaas.CreateOptions{})
		}
		keys := b.client.V1alpha1().SshKeys(o.Metadata.Project)
		return keys.Create(ctx, o, gpupaas.CreateOptions{})
	default:
		return nil, fmt.Errorf("unsupported kind %q for remote apply", obj.GetKind())
	}
}

func (b *Backend) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error) {
	switch gvk.Kind {
	case v1alpha1.KindProject:
		return b.client.V1alpha1().Projects().Get(ctx, name, gpupaas.GetOptions{})
	case v1alpha1.KindWorkspace:
		return b.client.V1alpha1().Workspaces(project).Get(ctx, name, gpupaas.GetOptions{})
	case v1alpha1.KindWorkspaceCollaborator:
		return b.client.V1alpha1().Workspaces(project).Collaborators(workspace).Get(ctx, name, gpupaas.GetOptions{})
	case v1alpha1.KindVirtualMachine:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).VirtualMachines(workspace).Get(ctx, name, gpupaas.GetOptions{})
		}
		return b.client.V1alpha1().VirtualMachines(project).Get(ctx, name, gpupaas.GetOptions{})
	case v1alpha1.KindStorage:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).Storages(workspace).Get(ctx, name, gpupaas.GetOptions{})
		}
		return b.client.V1alpha1().Storages(project).Get(ctx, name, gpupaas.GetOptions{})
	case v1alpha1.KindSecurityGroup:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).SecurityGroups(workspace).Get(ctx, name, gpupaas.GetOptions{})
		}
		return b.client.V1alpha1().SecurityGroups(project).Get(ctx, name, gpupaas.GetOptions{})
	case v1alpha1.KindSshKey:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).SshKeys(workspace).Get(ctx, name, gpupaas.GetOptions{})
		}
		return b.client.V1alpha1().SshKeys(project).Get(ctx, name, gpupaas.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported kind %q for remote get", gvk.Kind)
	}
}

func (b *Backend) List(ctx context.Context, gvk runtime.GroupVersionKind, opts backend.ListOptions) ([]runtime.Object, error) {
	switch gvk.Kind {
	case v1alpha1.KindProject:
		list, err := b.client.V1alpha1().Projects().List(ctx, gpupaas.ListOptions{})
		if err != nil {
			return nil, err
		}
		return list.GetItems(), nil
	case v1alpha1.KindWorkspace:
		list, err := b.client.V1alpha1().Workspaces(opts.Project).List(ctx, gpupaas.ListOptions{})
		if err != nil {
			return nil, err
		}
		items := list.GetItems()
		if opts.Workspace == "" {
			return items, nil
		}
		var filtered []runtime.Object
		for _, item := range items {
			if item.GetName() == opts.Workspace || item.GetWorkspace() == opts.Workspace {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	case v1alpha1.KindWorkspaceCollaborator:
		if opts.Workspace == "" {
			return nil, fmt.Errorf("workspace is required to list collaborators")
		}
		list, err := b.client.V1alpha1().Workspaces(opts.Project).Collaborators(opts.Workspace).List(ctx, gpupaas.ListOptions{
			SSOUsers: opts.SSOUsers,
		})
		if err != nil {
			return nil, err
		}
		return list.GetItems(), nil
	case v1alpha1.KindVirtualMachine:
		var list *v1alpha1.VirtualMachineList
		var err error
		if opts.Workspace != "" {
			list, err = b.client.V1alpha1().Workspaces(opts.Project).VirtualMachines(opts.Workspace).List(ctx, gpupaas.ListOptions{})
		} else {
			list, err = b.client.V1alpha1().VirtualMachines(opts.Project).List(ctx, gpupaas.ListOptions{})
		}
		if err != nil {
			return nil, err
		}
		return list.GetItems(), nil
	case v1alpha1.KindStorage:
		var list *v1alpha1.StorageList
		var err error
		if opts.Workspace != "" {
			list, err = b.client.V1alpha1().Workspaces(opts.Project).Storages(opts.Workspace).List(ctx, gpupaas.ListOptions{})
		} else {
			list, err = b.client.V1alpha1().Storages(opts.Project).List(ctx, gpupaas.ListOptions{})
		}
		if err != nil {
			return nil, err
		}
		return list.GetItems(), nil
	case v1alpha1.KindSecurityGroup:
		var list *v1alpha1.SecurityGroupList
		var err error
		if opts.Workspace != "" {
			list, err = b.client.V1alpha1().Workspaces(opts.Project).SecurityGroups(opts.Workspace).List(ctx, gpupaas.ListOptions{})
		} else {
			list, err = b.client.V1alpha1().SecurityGroups(opts.Project).List(ctx, gpupaas.ListOptions{})
		}
		if err != nil {
			return nil, err
		}
		return list.GetItems(), nil
	case v1alpha1.KindSshKey:
		var list *v1alpha1.SshKeyList
		var err error
		if opts.Workspace != "" {
			list, err = b.client.V1alpha1().Workspaces(opts.Project).SshKeys(opts.Workspace).List(ctx, gpupaas.ListOptions{})
		} else {
			list, err = b.client.V1alpha1().SshKeys(opts.Project).List(ctx, gpupaas.ListOptions{})
		}
		if err != nil {
			return nil, err
		}
		return list.GetItems(), nil
	default:
		return nil, fmt.Errorf("unsupported kind %q for remote list", gvk.Kind)
	}
}

func (b *Backend) Delete(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error {
	switch gvk.Kind {
	case v1alpha1.KindProject:
		return b.client.V1alpha1().Projects().Delete(ctx, name, gpupaas.DeleteOptions{})
	case v1alpha1.KindWorkspace:
		return b.client.V1alpha1().Workspaces(project).Delete(ctx, name, gpupaas.DeleteOptions{})
	case v1alpha1.KindWorkspaceCollaborator:
		return b.client.V1alpha1().Workspaces(project).Collaborators(workspace).Delete(ctx, name, gpupaas.DeleteOptions{})
	case v1alpha1.KindVirtualMachine:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).VirtualMachines(workspace).Delete(ctx, name, gpupaas.DeleteOptions{})
		}
		return b.client.V1alpha1().VirtualMachines(project).Delete(ctx, name, gpupaas.DeleteOptions{})
	case v1alpha1.KindStorage:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).Storages(workspace).Delete(ctx, name, gpupaas.DeleteOptions{})
		}
		return b.client.V1alpha1().Storages(project).Delete(ctx, name, gpupaas.DeleteOptions{})
	case v1alpha1.KindSecurityGroup:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).SecurityGroups(workspace).Delete(ctx, name, gpupaas.DeleteOptions{})
		}
		return b.client.V1alpha1().SecurityGroups(project).Delete(ctx, name, gpupaas.DeleteOptions{})
	case v1alpha1.KindSshKey:
		if workspace != "" {
			return b.client.V1alpha1().Workspaces(project).SshKeys(workspace).Delete(ctx, name, gpupaas.DeleteOptions{})
		}
		return b.client.V1alpha1().SshKeys(project).Delete(ctx, name, gpupaas.DeleteOptions{})
	default:
		return fmt.Errorf("unsupported kind %q for remote delete", gvk.Kind)
	}
}
