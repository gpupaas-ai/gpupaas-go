package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// AddToScheme registers gpupaas v1alpha1 types.
func AddToScheme(scheme *runtime.Scheme) {
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindProject},
		&Project{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "projects"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindWorkspace},
		&Workspace{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "workspaces"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindWorkspaceCollaborator},
		&WorkspaceCollaborator{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "workspacecollaborators"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindVirtualMachine},
		&VirtualMachine{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "virtualmachines"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindStorage},
		&Storage{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "storages"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindSecurityGroup},
		&SecurityGroup{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "securitygroups"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindSshKey},
		&SshKey{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "sshkeys"},
	)
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindBaremetalMachine},
		&BaremetalMachine{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "baremetalmachines"},
	)
}

// DefaultScheme returns a scheme with all v1alpha1 types registered.
func DefaultScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	return scheme
}

// NewSerializer returns a serializer for v1alpha1 types.
func NewSerializer(scheme *runtime.Scheme) *runtime.Serializer {
	return runtime.NewSerializer(scheme)
}

// MustRegisterDefaults registers defaults on scheme or panics.
func MustRegisterDefaults(scheme *runtime.Scheme) {
	AddToScheme(scheme)
}
