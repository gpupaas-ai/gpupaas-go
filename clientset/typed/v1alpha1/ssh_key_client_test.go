package v1alpha1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	"github.com/gpupaas-ai/gpupaas-go/clientset"
)

func TestSshKeyClientGetByScope(t *testing.T) {
	const (
		projectPath   = "/apis/dev.envmgmt.io/v1/projects/demo/sshkeys/my-key"
		workspacePath = "/apis/dev.envmgmt.io/v1/projects/demo/workspaces/dev/sshkeys/my-key"
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && (r.URL.Path == projectPath || r.URL.Path == workspacePath):
			meta := map[string]any{"name": "my-key", "project": "demo"}
			if r.URL.Path == workspacePath {
				meta["workspace"] = "dev"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"apiVersion": "dev.envmgmt.io/v1", "kind": "SshKey", "metadata": meta,
				"spec":   map[string]any{"public_key": "ssh-rsa AAAA"},
				"status": map[string]any{"status": "success"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		}
	}))
	defer srv.Close()

	cs, err := clientset.NewForConfig(gpupaas.NewConfig(srv.URL, "token"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	got, err := cs.V1alpha1().SshKeys("demo").Get(ctx, "my-key", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "my-key" || got.Metadata.Workspace != "" || got.Spec.PublicKey != "ssh-rsa AAAA" || got.Status.Status != "success" {
		t.Fatalf("project key = %+v", got)
	}

	got, err = cs.V1alpha1().Workspaces("demo").SshKeys("dev").Get(ctx, "my-key", gpupaas.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Workspace != "dev" {
		t.Fatalf("workspace key = %+v", got.Metadata)
	}

	if _, err := cs.V1alpha1().SshKeys("demo").Get(ctx, "missing", gpupaas.GetOptions{}); !gpupaas.IsNotFound(err) {
		t.Fatalf("missing key should be not found, got %v", err)
	}
}
