package gpupaas

import "testing"

// The platform looks an organization up by exact name, so stray spaces must never reach the header, whichever way the
// config was built.
func TestOrganizationIDIsTrimmedByNormalize(t *testing.T) {
	cfg := Config{Endpoint: "https://x", APIKey: "k", OrganizationID: " acme "}
	cfg.Normalize()
	if cfg.OrganizationID != "acme" {
		t.Fatalf("got %q", cfg.OrganizationID)
	}
}
