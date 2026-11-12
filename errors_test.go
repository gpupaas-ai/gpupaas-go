package gpupaas

import (
	"fmt"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	gateway := `{"internal":"ent: workspace_compute_instance not found","code":1,"external":"Upstream misbehaving"}`
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"404", &APIError{StatusCode: 404, Body: "nope"}, true},
		{"502 wrapped not found", &APIError{StatusCode: 502, Body: gateway}, true},
		{"502 wrapped not found, other case", &APIError{StatusCode: 502, Body: `{"internal":"ENT: Workspace Not Found"}`}, true},
		{"502 gateway that only mentions not found", &APIError{StatusCode: 502, Body: `{"internal":"upstream service not found","external":"Bad gateway"}`}, false},
		{"502 other reason", &APIError{StatusCode: 502, Body: `{"internal":"cannot destroy compute instance with status destroying"}`}, false},
		{"500 mentioning not found", &APIError{StatusCode: 500, Body: gateway}, false},
		{"403", &APIError{StatusCode: 403, Body: "not found"}, false},
		{"wrapped", fmt.Errorf("read: %w", &APIError{StatusCode: 502, Body: gateway}), true},
		{"plain error", fmt.Errorf("boom"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := IsNotFound(c.err); got != c.want {
			t.Errorf("%s: IsNotFound = %v, want %v", c.name, got, c.want)
		}
	}
}
