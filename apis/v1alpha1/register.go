package v1alpha1

import "github.com/gpupaas-ai/gpupaas-go/runtime"

// AddToScheme registers gpupaas v1alpha1 types.
//
// Only MKSCluster is registered: it is the kind the apply/delete engine (client.Client, apply
// package) handles. VMInstance and BareMetal are used through the typed clientset.
func AddToScheme(scheme *runtime.Scheme) {
	scheme.AddKnownTypeWithName(
		runtime.GroupVersionKind{Group: Group, Version: Version, Kind: KindMKSCluster},
		&MKSCluster{},
		runtime.GroupVersionResource{Group: Group, Version: Version, Resource: "mksclusters"},
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
