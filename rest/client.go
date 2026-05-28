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

// Post issues a POST request.
func (c *Client) Post(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPost, path, in, out)
}

// Put issues a PUT request.
func (c *Client) Put(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPut, path, in, out)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
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

	c.logResponse(resp, respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp.StatusCode, resp.Header.Get("X-Request-Id"), respBody)
	}

	if out == nil || len(respBody) == 0 {
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
		Message string `json:"message"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Message != "" {
			msg = parsed.Message
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
	}
}
