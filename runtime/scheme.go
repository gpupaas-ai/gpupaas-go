package runtime

import (
	"fmt"
	"reflect"
)

// Scheme maps GVK to Go types.
type Scheme struct {
	gvkToType map[GroupVersionKind]reflect.Type
	typeToGVK map[reflect.Type]GroupVersionKind
	gvkToGVR  map[GroupVersionKind]GroupVersionResource
}

// NewScheme creates an empty scheme.
func NewScheme() *Scheme {
	return &Scheme{
		gvkToType: map[GroupVersionKind]reflect.Type{},
		typeToGVK: map[reflect.Type]GroupVersionKind{},
		gvkToGVR:  map[GroupVersionKind]GroupVersionResource{},
	}
}

// AddKnownTypeWithName registers a type for a GVK and GVR.
func (s *Scheme) AddKnownTypeWithName(gvk GroupVersionKind, obj Object, gvr GroupVersionResource) {
	t := reflect.TypeOf(obj).Elem()
	s.gvkToType[gvk] = t
	s.typeToGVK[t] = gvk
	s.gvkToGVR[gvk] = gvr
}

// New creates a new object for the given GVK.
func (s *Scheme) New(gvk GroupVersionKind) (Object, error) {
	t, ok := s.gvkToType[gvk]
	if !ok {
		return nil, fmt.Errorf("no kind registered for %s", gvk)
	}
	v := reflect.New(t).Interface()
	obj, ok := v.(Object)
	if !ok {
		return nil, fmt.Errorf("registered type for %s does not implement Object", gvk)
	}
	return obj, nil
}

// ObjectGVK returns the GVK for a registered object.
func (s *Scheme) ObjectGVK(obj Object) (GroupVersionKind, error) {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	gvk, ok := s.typeToGVK[t]
	if !ok {
		return GroupVersionKind{}, fmt.Errorf("type %T is not registered in scheme", obj)
	}
	return gvk, nil
}

// GVRForGVK returns the REST resource for a GVK.
func (s *Scheme) GVRForGVK(gvk GroupVersionKind) (GroupVersionResource, error) {
	gvr, ok := s.gvkToGVR[gvk]
	if !ok {
		return GroupVersionResource{}, fmt.Errorf("no resource registered for %s", gvk)
	}
	return gvr, nil
}

// Recognizes reports whether gvk is registered.
func (s *Scheme) Recognizes(gvk GroupVersionKind) bool {
	_, ok := s.gvkToType[gvk]
	return ok
}
