package remote

import (
	"context"
	"fmt"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	v1alpha1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

// Backend implements backend.Backend over the gpupaas REST API. It handles MKSCluster, the kind
// the generic client applies; first-class VMs and bare metal are used through the typed clientset.
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

func (b *Backend) clusters(project, workspace string) typed.MKSClusterInterface {
	if workspace != "" {
		return b.client.V1alpha1().Workspaces(project).MKSClusters(workspace)
	}
	return b.client.V1alpha1().MKSClusters(project)
}

func (b *Backend) Apply(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
	switch o := obj.(type) {
	case *v1alpha1.MKSCluster:
		// Apply is an upsert: the backend creates the cluster or updates the existing one.
		return b.clusters(o.Metadata.Project, o.Metadata.Workspace).Update(ctx, o, gpupaas.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported kind %q for remote apply", obj.GetKind())
	}
}

func (b *Backend) Get(ctx context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error) {
	switch gvk.Kind {
	case v1alpha1.KindMKSCluster:
		return b.clusters(project, workspace).Get(ctx, name, gpupaas.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported kind %q for remote get", gvk.Kind)
	}
}

func (b *Backend) List(ctx context.Context, gvk runtime.GroupVersionKind, opts backend.ListOptions) ([]runtime.Object, error) {
	switch gvk.Kind {
	case v1alpha1.KindMKSCluster:
		list, err := b.clusters(opts.Project, opts.Workspace).List(ctx, gpupaas.ListOptions{})
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
	case v1alpha1.KindMKSCluster:
		return b.clusters(project, workspace).Delete(ctx, name, gpupaas.DeleteOptions{})
	default:
		return fmt.Errorf("unsupported kind %q for remote delete", gvk.Kind)
	}
}
