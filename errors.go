package gpupaas

import (
	"errors"
	"fmt"
	"regexp"
)

// APIError represents a non-success HTTP response from the API.
type APIError struct {
	StatusCode int
	Reason     string
	Message    string
	RequestID  string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api error %d", e.StatusCode)
}

// storeNotFoundPattern matches the store-layer error the paas.envmgmt.io API wraps in a 502, such as
// "ent: workspace_compute_instance not found". A gateway 502 that merely mentions "not found" (an unreachable
// upstream, say) does not match, so it is not read as a deleted object.
var storeNotFoundPattern = regexp.MustCompile(`(?i)\bent: [^"]*\bnot found`)

// IsNotFound reports whether err means the requested object does not exist.
//
// Most endpoints answer 404. Some endpoints on the paas.envmgmt.io API (for
// example compute instances) wrap the store error and answer 502 with a body
// such as {"internal":"ent: workspace_compute_instance not found",
// "external":"Upstream misbehaving"}; those are treated as not found too.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 404 {
			return true
		}
		return apiErr.StatusCode == 502 && storeNotFoundPattern.MatchString(apiErr.Body)
	}
	return false
}

// IsConflict reports whether err is a 409 API error.
func IsConflict(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 409
	}
	return false
}

// IsUnauthorized reports whether err is a 401 API error.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsForbidden reports whether err is a 403 API error.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 403
	}
	return false
}

// IsServerError reports whether err is a 5xx API error.
func IsServerError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
	}
	return false
}
