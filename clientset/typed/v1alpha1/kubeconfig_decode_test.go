package v1alpha1_test

import (
	"encoding/base64"
	"testing"

	typed "github.com/gpupaas-ai/gpupaas-go/clientset/typed/v1alpha1"
)

func TestDecodeKubeconfig(t *testing.T) {
	const yaml = "apiVersion: v1\nkind: Config\nclusters:\n- name: c\n"
	envelope := `{"content_type":"application/yaml","data":"` + base64.StdEncoding.EncodeToString([]byte(yaml)) + `"}`
	cases := []struct{ name, in, want string }{
		{"the JSON envelope of a JSON request is unwrapped", envelope, yaml},
		{"surrounding whitespace is ignored", "\n " + envelope + "\n", yaml},
		{"YAML passes through unchanged", yaml, yaml},
		{"data that is not base64 is kept as it is", `{"content_type":"application/yaml","data":"clusters: [x]"}`, "clusters: [x]"},
		{"JSON without data passes through", `{"error":"rpc error"}`, `{"error":"rpc error"}`},
		{"invalid JSON passes through", `{not json`, `{not json`},
		{"empty stays empty", ``, ``},
	}
	for _, c := range cases {
		if got := string(typed.DecodeKubeconfig([]byte(c.in))); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCheckKubeconfig(t *testing.T) {
	one := "apiVersion: v1\nkind: Config\nclusters:\n- name: c\n  cluster:\n    server: https://x\n"
	two := one + "- name: other\n  cluster:\n    server: https://y\n"
	if err := typed.CheckKubeconfig([]byte(one), "c"); err != nil {
		t.Errorf("one cluster must be accepted: %v", err)
	}
	if err := typed.CheckKubeconfig([]byte(two), "c"); err == nil {
		t.Error("several clusters must be refused")
	}
	for _, bad := range []string{"", "not: a kubeconfig", "clusters: []", ": : :"} {
		if err := typed.CheckKubeconfig([]byte(bad), "c"); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
}
