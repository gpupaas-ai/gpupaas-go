package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func redactHeader(key, value string) (string, bool) {
	switch strings.ToLower(key) {
	case "x-api-key", "x-rafay-api-keyid", "authorization":
		return key + ": ***", true
	default:
		return key + ": " + value, false
	}
}

func (c *Client) logRequest(method, url string, reqBody []byte, req *http.Request) {
	if !c.config.Verbose {
		return
	}
	out := c.logWriter()
	fmt.Fprintf(out, ">>> %s %s\n", method, url)
	for k, vals := range req.Header {
		for _, v := range vals {
			line, _ := redactHeader(k, v)
			fmt.Fprintln(out, line)
		}
	}
	if len(reqBody) > 0 {
		fmt.Fprintf(out, "%s\n", formatBody(reqBody))
	}
}

func (c *Client) logResponse(resp *http.Response, respBody []byte) {
	if !c.config.Verbose {
		return
	}
	out := c.logWriter()
	fmt.Fprintf(out, "<<< %s %s\n", resp.Status, resp.Request.URL.String())
	for k, vals := range resp.Header {
		for _, v := range vals {
			fmt.Fprintf(out, "%s: %s\n", k, v)
		}
	}
	if len(respBody) > 0 {
		fmt.Fprintf(out, "%s\n", formatBody(respBody))
	}
	fmt.Fprintln(out)
}

func (c *Client) logWriter() io.Writer {
	if c.config.LogOutput != nil {
		return c.config.LogOutput
	}
	return os.Stderr
}

// secretKey reports whether the value of a JSON key must not reach a log: passwords, tokens, keys and
// the user data of an instance.
func secretKey(k string) bool {
	k = strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(k))
	switch k {
	case "userdata", "privatekey", "accesskey", "secretkey", "apikey":
		return true
	}
	return strings.Contains(k, "password") || strings.Contains(k, "token") || strings.Contains(k, "secret")
}

// redact returns v with the value of every secret key replaced.
func redact(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			if _, isString := val.(string); isString && secretKey(k) && val != "" {
				out[k] = "***"
				continue
			}
			out[k] = redact(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = redact(val)
		}
		return out
	}
	return v
}

func formatBody(body []byte) string {
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body)
	}
	pretty, err := json.MarshalIndent(redact(v), "", "  ")
	if err != nil {
		return string(body)
	}
	return string(pretty)
}
