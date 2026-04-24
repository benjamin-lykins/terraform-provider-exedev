// Package client provides an HTTP client for the exe.dev HTTPS API.
//
// The exe.dev API works by POSTing CLI command strings to https://exe.dev/exec
// with a Bearer token in the Authorization header. Responses are JSON.
//
// See: https://exe.dev/docs/https-api
package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://exe.dev"

// Client is an exe.dev HTTPS API client.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// New returns a new Client authenticated with the given bearer token.
func New(token string) *Client {
	return &Client{
		token:   token,
		baseURL: defaultBaseURL,
		http: &http.Client{
			Timeout: 35 * time.Second,
		},
	}
}

// ExecError represents a non-2xx response from the exe.dev API.
type ExecError struct {
	StatusCode int
	Body       string
}

func (e *ExecError) Error() string {
	return fmt.Sprintf("exe.dev API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// Exec runs a CLI command via the exe.dev HTTPS API and returns the raw
// response body (which is always JSON when the command succeeds).
//
// An *ExecError is returned for non-2xx HTTP responses. Callers can use
// errors.As to inspect the status code and decide whether to treat a 404-like
// response as "not found" rather than a hard error.
func (c *Client) Exec(cmd string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/exec", bytes.NewBufferString(cmd))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ExecError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	return body, nil
}

// ShellQuote returns s quoted for use as a single argument in an exe.dev
// command string. Values are wrapped in single quotes; embedded single quotes
// are escaped as '\''.
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	// Check if quoting is needed
	safe := true
	for _, r := range s {
		if !isSafeChar(r) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	// Single-quote the string, escaping embedded single quotes
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isSafeChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' || r == '/' || r == ':' || r == '@' || r == '+'
}
