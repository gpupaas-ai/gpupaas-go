package runtime

import "fmt"

// GroupVersionKind identifies an API object type.
type GroupVersionKind struct {
	Group   string
	Version string
	Kind    string
}

func (g GroupVersionKind) String() string {
	if g.Group == "" {
		return fmt.Sprintf("%s/%s", g.Version, g.Kind)
	}
	return fmt.Sprintf("%s/%s/%s", g.Group, g.Version, g.Kind)
}

// GroupVersionResource identifies a REST resource collection.
type GroupVersionResource struct {
	Group    string
	Version  string
	Resource string
}

func (g GroupVersionResource) String() string {
	if g.Group == "" {
		return fmt.Sprintf("%s/%s", g.Version, g.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", g.Group, g.Version, g.Resource)
}

// GroupVersion parses an apiVersion string.
type GroupVersion struct {
	Group   string
	Version string
}

// ParseGroupVersion splits group/version from apiVersion.
func ParseGroupVersion(apiVersion string) (GroupVersion, error) {
	for i := len(apiVersion) - 1; i >= 0; i-- {
		if apiVersion[i] == '/' {
			return GroupVersion{Group: apiVersion[:i], Version: apiVersion[i+1:]}, nil
		}
	}
	return GroupVersion{}, fmt.Errorf("invalid apiVersion %q", apiVersion)
}

// APIPathPrefix returns /apis/{group}/{version}.
func APIPathPrefix(gv GroupVersion) string {
	return fmt.Sprintf("/apis/%s/%s", gv.Group, gv.Version)
}
