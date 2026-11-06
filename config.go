package gpupaas

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultEndpoint = "https://api.gpupaas.ai"

// Config holds connection settings for the gpupaas.ai API.
type Config struct {
	Endpoint   string
	APIKey     string
	APISecret  string
	UserAgent  string
	HTTPClient *http.Client
	// Verbose logs HTTP requests and responses (remote backend only).
	Verbose bool
	// OrganizationID is sent as the X-Organization-ID header when set. It is the organization's id or
	// its exact, case-sensitive name. A partner-level API key (one that is not bound to an
	// organization) needs it to act inside a tenant organization; without it the platform answers 502
	// "id or partnerID/orgID/name not set". An organization-level key does not need it and the platform
	// ignores the header for such a key, even when it names another organization or none that exists.
	OrganizationID string

	// Token is deprecated; use APIKey. When APIKey is empty, Token is used.
	Token string
}

// NewConfig returns a Config with defaults applied.
// apiKey is the Rafay API key (same as paasctl api_key).
func NewConfig(endpoint, apiKey string) Config {
	cfg := Config{
		Endpoint: endpoint,
		APIKey:   apiKey,
	}
	cfg.normalize()
	return cfg
}

// ConfigFromEnv reads GPUPAAS_ENDPOINT, GPUPAAS_API_KEY, GPUPAAS_API_SECRET,
// GPUPAAS_VERBOSE, GPUPAAS_ORGANIZATION_ID, and GPUPAAS_TOKEN (deprecated alias for API key).
func ConfigFromEnv() Config {
	cfg := Config{
		Endpoint:  os.Getenv("GPUPAAS_ENDPOINT"),
		APIKey:    os.Getenv("GPUPAAS_API_KEY"),
		APISecret: os.Getenv("GPUPAAS_API_SECRET"),
		Token:     os.Getenv("GPUPAAS_TOKEN"),
		Verbose:   envBool("GPUPAAS_VERBOSE"),

		OrganizationID: strings.TrimSpace(os.Getenv("GPUPAAS_ORGANIZATION_ID")),
	}
	cfg.normalize()
	return cfg
}

func envBool(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	switch strings.ToLower(v) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func (c *Config) normalize() {
	if c.APIKey == "" && c.Token != "" {
		c.APIKey = c.Token
	}
	if c.Endpoint == "" {
		c.Endpoint = defaultEndpoint
	}
	c.Endpoint = strings.TrimRight(c.Endpoint, "/")
	if c.UserAgent == "" {
		c.UserAgent = "gpupaas-go/" + Version
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
}

// Normalize applies defaults to the configuration.
func (c *Config) Normalize() {
	c.normalize()
}
