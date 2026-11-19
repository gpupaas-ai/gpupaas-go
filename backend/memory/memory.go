package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/gpupaas-ai/gpupaas-go/backend"
	"github.com/gpupaas-ai/gpupaas-go/runtime"
)

type key struct {
	kind      string
	project   string
	workspace string
	name      string
}

// Store is an in-memory Backend for tests and examples.
type Store struct {
	mu    sync.RWMutex
	items map[key]runtime.Object
}

// New creates an empty in-memory store.
func New() *Store {
	return &Store{items: map[key]runtime.Object{}}
}

func (s *Store) Apply(_ context.Context, obj runtime.Object) (runtime.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := objectKey(obj)
	stored := obj.DeepCopyObject()
	s.items[k] = stored
	return stored.DeepCopyObject(), nil
}

func (s *Store) Get(_ context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) (runtime.Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k := lookupKey(gvk.Kind, project, workspace, name)
	obj, ok := s.items[k]
	if !ok {
		return nil, fmt.Errorf("%s/%s not found", gvk.Kind, name)
	}
	return obj.DeepCopyObject(), nil
}

func (s *Store) List(_ context.Context, gvk runtime.GroupVersionKind, opts backend.ListOptions) ([]runtime.Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []runtime.Object
	for k, obj := range s.items {
		if k.kind != gvk.Kind {
			continue
		}
		if opts.Project != "" && k.project != opts.Project {
			continue
		}
		if opts.Workspace != "" && k.workspace != opts.Workspace {
			continue
		}
		out = append(out, obj.DeepCopyObject())
	}
	return out, nil
}

func (s *Store) Delete(_ context.Context, gvk runtime.GroupVersionKind, project, workspace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := lookupKey(gvk.Kind, project, workspace, name)
	if _, ok := s.items[k]; !ok {
		return fmt.Errorf("%s/%s not found", gvk.Kind, name)
	}
	delete(s.items, k)
	return nil
}

func objectKey(obj runtime.Object) key {
	return lookupKey(obj.GetKind(), obj.GetProject(), obj.GetWorkspace(), obj.GetName())
}

func lookupKey(kind, project, workspace, name string) key {
	return key{kind: kind, project: project, workspace: workspace, name: name}
}
