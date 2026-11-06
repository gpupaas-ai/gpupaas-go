package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
		if r.URL.Path != "/apis/paas.k8smgmt.io/v3/projects/demo/clusters/c1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(apiv1.MKSCluster{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
			Metadata: apiv1.ObjectMeta{Name: "c1"},
		})
	}))
	defer srv.Close()

	client, err := rest.NewClient(rest.Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	var out apiv1.MKSCluster
	if err := client.Get(context.Background(), "/apis/paas.k8smgmt.io/v3/projects/demo/clusters/c1", &out); err != nil {
		t.Fatal(err)
	}
	if out.Metadata.Name != "c1" {
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
		_ = json.NewEncoder(w).Encode(apiv1.MKSCluster{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
			Metadata: apiv1.ObjectMeta{Name: "c1"},
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
	var out apiv1.MKSCluster
	if err := client.Get(context.Background(), "/apis/paas.k8smgmt.io/v3/projects/demo/clusters/c1", &out); err != nil {
		t.Fatal(err)
	}
}

func TestClientVerboseLogging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiv1.MKSCluster{
			TypeMeta: apiv1.TypeMeta{APIVersion: apiv1.APIVersion, Kind: apiv1.KindMKSCluster},
			Metadata: apiv1.ObjectMeta{Name: "c1"},
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
	var out apiv1.MKSCluster
	if err := client.Get(context.Background(), "/apis/paas.k8smgmt.io/v3/projects/demo/clusters/c1", &out); err != nil {
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

// Passwords, tokens and user data in request and response bodies must not reach the log.
func TestClientVerboseLoggingRedactsSecretsInBodies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"metadata":{"name":"v1"},"status":{"output":{"mount_password":"mount-pw-123","mount_access_token":"tok-456","volume_size":50,"public_ip":"1.2.3.4"}}}`))
	}))
	defer srv.Close()
	var logBuf bytes.Buffer
	client, err := rest.NewClient(rest.Config{BaseURL: srv.URL, APIKey: "k", Verbose: true, LogOutput: &logBuf})
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"spec": map[string]any{"controlPlane": map[string]any{"userPassword": "node-pw-789", "name": "cp"}, "user_data": "#cloud-config secret", "workerPools": []any{map[string]any{"userPassword": "pool-pw-000"}}}}
	var out map[string]any
	if err := client.Post(context.Background(), "/x", body, &out); err != nil {
		t.Fatal(err)
	}
	logged := logBuf.String()
	for _, secret := range []string{"node-pw-789", "pool-pw-000", "cloud-config secret", "mount-pw-123", "tok-456"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("%q reached the log:\n%s", secret, logged)
		}
	}
	for _, kept := range []string{`"name": "cp"`, `"volume_size": 50`, `"public_ip": "1.2.3.4"`} {
		if !strings.Contains(logged, kept) {
			t.Fatalf("%s was redacted too:\n%s", kept, logged)
		}
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
	var out apiv1.MKSCluster
	err = client.Get(context.Background(), "/apis/paas.k8smgmt.io/v3/projects/demo/clusters/missing", &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !gpupaas.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGatewayErrorMessageUsesInternalReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"internal":"cannot destroy compute instance with status destroying","code":1,"external":"Upstream misbehaving"}`))
	}))
	defer srv.Close()
	c, err := rest.NewClient(rest.Config{BaseURL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Post(context.Background(), "/x", map[string]any{}, nil)
	var apiErr *gpupaas.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.StatusCode != 502 || apiErr.Message != "cannot destroy compute instance with status destroying" || apiErr.Reason != "Upstream misbehaving" {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
	if !strings.Contains(apiErr.Body, "destroying") {
		t.Fatalf("raw body should be kept: %q", apiErr.Body)
	}
}

// The bodies are the ones narayana-tb1 returned on 27 Oct 2026 for a partner-level API key.
func TestOrganizationErrorsSayHowToFixThem(t *testing.T) {
	const missing = `{"internal":"unable to hydrate options rpc error: code = Unknown desc = unable to get project rpc error: code = Unknown desc = id or partnerID/orgID/name not set","code":1,"external":"Upstream misbehaving"}`
	const unknown = `get organization by name returned error pg: no rows in result set`
	cases := []struct {
		name, org, body string
		status          int
		want            []string
		wantNot         []string
	}{
		{"no organization sent", "", missing, http.StatusBadGateway,
			[]string{"id or partnerID/orgID/name not set", "set organization_id (GPUPAAS_ORGANIZATION_ID)", "organization-level API key"}, nil},
		{"unknown organization", "nope", unknown, http.StatusInternalServerError,
			[]string{`organization "nope" was not found`, "exact, case-sensitive name", unknown}, nil},
		// The hint belongs to these two errors only.
		{"organization set, project missing", "7w2lnkp", `{"internal":"unable to get project NotFound"}`, http.StatusNotFound,
			nil, []string{"organization_id"}},
		{"no organization, unrelated error", "", `{"internal":"cannot destroy compute instance"}`, http.StatusBadGateway,
			nil, []string{"organization_id"}},
		{"organization set, the missing-organization text", "7w2lnkp", missing, http.StatusBadGateway,
			nil, []string{"set organization_id"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c, err := rest.NewClient(rest.Config{BaseURL: srv.URL, APIKey: "k", OrganizationID: tc.org})
			if err != nil {
				t.Fatal(err)
			}
			err = c.Get(context.Background(), "/x", nil)
			var apiErr *gpupaas.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tc.status {
				t.Fatalf("expected an APIError with status %d, got %v", tc.status, err)
			}
			for _, w := range tc.want {
				if !strings.Contains(apiErr.Message, w) {
					t.Errorf("message lacks %q: %s", w, apiErr.Message)
				}
			}
			for _, w := range tc.wantNot {
				if strings.Contains(apiErr.Message, w) {
					t.Errorf("message must not contain %q: %s", w, apiErr.Message)
				}
			}
			if apiErr.Body != tc.body {
				t.Errorf("the raw body must stay unchanged, got %q", apiErr.Body)
			}
		})
	}
}

func TestOrganizationHeaderOnlyWhenConfigured(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("X-Organization-ID"))
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	for _, org := range []string{"", "7w2lnkp"} {
		c, err := rest.NewClient(rest.Config{BaseURL: srv.URL, APIKey: "k", OrganizationID: org})
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if err := c.Get(context.Background(), "/x", &out); err != nil {
			t.Fatal(err)
		}
	}
	if len(got) != 2 || got[0] != "" || got[1] != "7w2lnkp" {
		t.Fatalf("X-Organization-ID values = %q, want [\"\", \"7w2lnkp\"]", got)
	}
}
