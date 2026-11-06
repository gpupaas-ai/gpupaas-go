package v1alpha1

import (
	"strings"
	"testing"
)

func TestFilterLogLinesKeepsStructuredProblemsAndPlainOutputProblems(t *testing.T) {
	raw := strings.Join([]string{
		`time=2026-10-15 11:50:35,493 level=INFO path=x line=1 msg=all fine`,
		`time=2026-10-15 11:50:36,484 level=WARNING path=x line=2 msg=No IP found from provider`,
		`Output: mac                      error: A device with MAC 52:53:00:AA:BB:03 already exists: ajay-testing-hostname-2`,
		`Error: Warning: there were uncommitted changes in the device mode!`,
		`ci-info: | bond0 | True | up |`,
		`debconf: unable to initialize frontend: Dialog`,
		`time=2026-10-15 11:50:40,268 level=ERROR path=x line=3 msg=Failed to set MAC and image`,
		`time=2026-10-15 11:50:41,000 level=ERROR path=x line=4 msg=login failed, password=hunter2`,
		`Output: secret not found in vault`,
		`Connection refused by the head node`,
	}, "\n")
	got := filterLogLines(raw)
	want := []string{
		"WARNING: No IP found from provider",
		"OUTPUT: Output: mac                      error: A device with MAC 52:53:00:AA:BB:03 already exists: ajay-testing-hostname-2",
		"OUTPUT: Error: Warning: there were uncommitted changes in the device mode!",
		"ERROR: Failed to set MAC and image",
		"OUTPUT: Connection refused by the head node",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines: %q", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}
