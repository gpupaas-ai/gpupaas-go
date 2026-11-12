package gpupaas

import (
	"fmt"
	"regexp"
	"testing"
)

var semver = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.-]+)?$`)

func TestVersionIsSemver(t *testing.T) {
	if !semver.MatchString(Version) {
		t.Fatalf("Version %q is not a semantic version without the v prefix", Version)
	}
}

// ControllerAPI must be the line of Version, so that bumping one without the other fails here.
func TestControllerAPIIsTheLineOfVersion(t *testing.T) {
	m := semver.FindStringSubmatch(Version)
	if m == nil {
		t.Skip("covered by TestVersionIsSemver")
	}
	if want := fmt.Sprintf("v%s.%s", m[1], m[2]); ControllerAPI != want {
		t.Fatalf("ControllerAPI is %q, want %q (the line of Version %s)", ControllerAPI, want, Version)
	}
}
