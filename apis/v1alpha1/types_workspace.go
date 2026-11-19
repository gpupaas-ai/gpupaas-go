package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// Workspace is a partition of a project that holds instances, SSH keys and security groups. It has
// metadata only: the API documents spec.readme and spec.iconURL, but a live controller accepts them and
// never stores them, so the SDK does not offer them.
type Workspace struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ObjectMeta `json:"metadata" yaml:"metadata"`
}

func (w *Workspace) GetAPIVersion() string { return w.APIVersion }
func (w *Workspace) GetKind() string       { return w.Kind }
func (w *Workspace) GetName() string       { return w.Metadata.Name }
func (w *Workspace) GetProject() string    { return w.Metadata.Project }
func (w *Workspace) GetWorkspace() string  { return w.Metadata.Workspace }
func (w *Workspace) SetProject(v string)   { w.Metadata.Project = v }
func (w *Workspace) SetWorkspace(v string) { w.Metadata.Workspace = v }
func (w *Workspace) DeepCopyObject() runtime.Object {
	cp := *w
	cp.Metadata = copyObjectMeta(w.Metadata)
	return &cp
}

// WorkspaceList is a list of Workspace resources.
type WorkspaceList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Metadata ListMeta    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items    []Workspace `json:"items" yaml:"items"`
}

func (l *WorkspaceList) GetAPIVersion() string { return l.APIVersion }
func (l *WorkspaceList) GetKind() string       { return l.Kind }
func (l *WorkspaceList) GetItems() []runtime.Object {
	out := make([]runtime.Object, len(l.Items))
	for i := range l.Items {
		out[i] = l.Items[i].DeepCopyObject()
	}
	return out
}
func (l *WorkspaceList) SetItems(items []runtime.Object) {
	l.Items = make([]Workspace, len(items))
	for i, item := range items {
		if w, ok := item.(*Workspace); ok {
			l.Items[i] = *w
		}
	}
}
