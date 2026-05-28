package runtime

type Object interface {
	GetAPIVersion() string
	GetKind() string
	GetName() string
	GetProject() string
	GetWorkspace() string
	SetProject(string)
	SetWorkspace(string)
	DeepCopyObject() Object
}

// ObjectList is implemented by list types.
type ObjectList interface {
	GetAPIVersion() string
	GetKind() string
	GetItems() []Object
	SetItems([]Object)
}
