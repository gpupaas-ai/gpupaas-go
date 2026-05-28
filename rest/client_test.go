package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
	apiv1 "github.com/gpupaas-ai/gpupaas-go/apis/v1alpha1"
	"github.com/gpupaas-ai/gpupaas-go/rest"
)

func TestClientGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis/gpupaas.ai/v1alpha1/projects/demo" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(apiv1.Project{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindProject},
			Metadata: apiv1.ObjectMeta{Name: "demo"},
		})
	}))
	defer srv.Close()

	client, err := rest.NewClient(rest.Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	var out apiv1.Project
	if err := client.Get(context.Background(), "/apis/gpupaas.ai/v1alpha1/projects/demo", &out); err != nil {
		t.Fatal(err)
	}
	if out.Metadata.Name != "demo" {
		t.Fatalf("unexpected name %q", out.Metadata.Name)
	}
}

func TestClientAPIKeyAuth(t *testing.T) {
	const apiKey = "test-api-key-id"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-KEY"); got != apiKey {
			t.Fatalf("X-API-KEY: got %q want %q", got, apiKey)
		}
		if got := r.Header.Get("X-RAFAY-API-KEYID"); got != apiKey {
			t.Fatalf("X-RAFAY-API-KEYID: got %q want %q", got, apiKey)
		}
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			t.Fatalf("unexpected Bearer Authorization header: %q", auth)
		}
		if auth == "" {
			t.Fatal("expected HMAC Authorization signature header")
		}
		if r.Header.Get("date") == "" || r.Header.Get("content-md5") == "" || r.Header.Get("nonce") == "" {
			t.Fatalf("missing signed headers: date=%q md5=%q nonce=%q",
				r.Header.Get("date"), r.Header.Get("content-md5"), r.Header.Get("nonce"))
		}
		_ = json.NewEncoder(w).Encode(apiv1.Project{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindProject},
			Metadata: apiv1.ObjectMeta{Name: "demo"},
		})
	}))
	defer srv.Close()

	client, err := rest.NewClient(rest.Config{
		BaseURL:   srv.URL,
		APIKey:    apiKey,
		APISecret: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	var out apiv1.Project
	if err := client.Get(context.Background(), "/apis/gpupaas.ai/v1alpha1/projects/demo", &out); err != nil {
		t.Fatal(err)
	}
}

func TestClientVerboseLogging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiv1.Project{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindProject},
			Metadata: apiv1.ObjectMeta{Name: "demo"},
		})
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	client, err := rest.NewClient(rest.Config{
		BaseURL:   srv.URL,
		APIKey:    "secret-api-key",
		APISecret: "secret-value",
		Verbose:   true,
		LogOutput: &logBuf,
	})
	if err != nil {
		t.Fatal(err)
	}
	var out apiv1.Project
	if err := client.Get(context.Background(), "/apis/gpupaas.ai/v1alpha1/projects/demo", &out); err != nil {
		t.Fatal(err)
	}
	logged := logBuf.String()
	if !strings.Contains(logged, ">>> GET") {
		t.Fatalf("expected request log, got: %s", logged)
	}
	if !strings.Contains(logged, "<<< 200") {
		t.Fatalf("expected response log, got: %s", logged)
	}
	if strings.Contains(logged, "secret-api-key") || strings.Contains(logged, "secret-value") {
		t.Fatalf("credentials must be redacted, got: %s", logged)
	}
	if !strings.Contains(logged, "Api-Key: ***") && !strings.Contains(logged, "X-API-KEY: ***") {
		t.Fatalf("expected redacted API key header, got: %s", logged)
	}
}

func TestClientNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	client, err := rest.NewClient(rest.Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	var out apiv1.Project
	err = client.Get(context.Background(), "/apis/gpupaas.ai/v1alpha1/projects/missing", &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !gpupaas.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
