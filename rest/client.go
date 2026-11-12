package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	gpupaas "github.com/gpupaas-ai/gpupaas-go"
)

// Config configures the REST client.
type Config struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	UserAgent  string
	HTTPClient *http.Client
	Verbose    bool
	LogOutput  io.Writer
	// OrganizationID, when set, is sent as the X-Organization-ID header.
	OrganizationID string

	// Token is deprecated; use APIKey.
	Token string
}

// Client performs HTTP requests against the gpupaas API.
type Client struct {
	config Config
}

// NewClient creates a REST client.
func NewClient(config Config) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	if config.HTTPClient == nil {
		cfg := gpupaas.NewConfig(config.BaseURL, config.APIKey)
		if config.APIKey == "" {
			cfg = gpupaas.NewConfig(config.BaseURL, config.Token)
		}
		config.HTTPClient = cfg.HTTPClient
	}
	if config.APIKey == "" && config.Token != "" {
		config.APIKey = config.Token
	}
	if config.UserAgent == "" {
		config.UserAgent = gpupaas.NewConfig("", "").UserAgent
	}
	return &Client{config: config}, nil
}

// Get issues a GET request and decodes JSON into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// GetRaw issues a GET request and returns the response body without decoding it.
func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, error) {
	var raw []byte
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// Post issues a POST request.
func (c *Client) Post(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPost, path, in, out)
}

// Put issues a PUT request.
func (c *Client) Put(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPut, path, in, out)
}

// Patch issues a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPatch, path, in, out)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// DeleteInto issues a DELETE request and decodes the JSON answer into out.
func (c *Client) DeleteInto(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodDelete, path, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var reqBody []byte
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = data
		body = bytes.NewReader(data)
	}

	url := c.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.config.UserAgent != "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}
	if c.config.OrganizationID != "" {
		req.Header.Set("X-Organization-ID", c.config.OrganizationID)
	}
	if err := signRequest(req, reqBody, c.config.APIKey, c.config.APISecret); err != nil {
		return fmt.Errorf("sign request: %w", err)
	}

	c.logRequest(method, url, reqBody, req)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	logged := respBody
	if strings.HasPrefix(path, "/v2/sentry/kubeconfig") {
		logged = []byte("<kubeconfig response not logged>") // it holds client credentials
	}
	c.logResponse(resp, logged)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return withOrganizationHint(newAPIError(resp.StatusCode, resp.Header.Get("X-Request-Id"), respBody), c.config.OrganizationID)
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if raw, ok := out.(*[]byte); ok { // plain-text endpoints (for example activity logs)
		*raw = respBody
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func newAPIError(status int, requestID string, body []byte) *gpupaas.APIError {
	msg := strings.TrimSpace(string(body))
	var parsed struct {
		Message  string `json:"message"`
		Reason   string `json:"reason"`
		Internal string `json:"internal"`
		External string `json:"external"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		switch {
		case parsed.Message != "":
			msg = parsed.Message
		case parsed.Internal != "":
			// Gateway-style errors: "external" is a generic label (for example
			// "Upstream misbehaving"); "internal" carries the actionable reason.
			msg = parsed.Internal
			if parsed.Reason == "" {
				parsed.Reason = parsed.External
			}
		}
	}
	return &gpupaas.APIError{
		StatusCode: status,
		Message:    msg,
		Reason:     parsed.Reason,
		RequestID:  requestID,
		Body:       string(body),
	}
}

// The platform's wording for an API key that is not bound to an organization and sent none, and for an
// organization header that matches nothing.
const (
	organizationMissing  = "partnerID/orgID/name not set"
	organizationNotFound = "get organization by name returned error"
)

// withOrganizationHint appends the fix to those two errors and keeps the platform's own text. An
// organization-level API key never reaches either one: the platform ignores the header for such a key.
func withOrganizationHint(e *gpupaas.APIError, organization string) *gpupaas.APIError {
	switch {
	case organization == "" && strings.Contains(e.Body, organizationMissing):
		e.Message += "; this API key is not bound to an organization, so set organization_id (GPUPAAS_ORGANIZATION_ID) to " +
			"the organization's id or its exact name, or use an organization-level API key"
	case organization != "" && strings.Contains(e.Body, organizationNotFound):
		e.Message = fmt.Sprintf("organization %q was not found for this API key; organization_id takes the organization's id or its exact, case-sensitive name (%s)",
			organization, e.Message)
	}
	return e
}

// ConfigFromGPUPAAS converts root Config to rest.Config.
func ConfigFromGPUPAAS(cfg gpupaas.Config) Config {
	return Config{
		BaseURL:    cfg.Endpoint,
		APIKey:     cfg.APIKey,
		APISecret:  cfg.APISecret,
		UserAgent:  cfg.UserAgent,
		HTTPClient: cfg.HTTPClient,
		Verbose:    cfg.Verbose,
		Token:      cfg.Token,

		OrganizationID: cfg.OrganizationID,
	}
}
