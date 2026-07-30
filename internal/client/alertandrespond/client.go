// Package alertandrespond is a minimal HTTP client for the Splunk AppDynamics Controller
// "Alert and Respond" REST APIs (Health Rule, Policy, Actions, Schedule).
package alertandrespond

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is an authenticated AppDynamics Controller API client.
type Client struct {
	controllerURL string
	clientID      string
	clientSecret  string
	httpClient    *http.Client

	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// New returns a Client for the given SaaS controller URL and OAuth client credentials.
func New(controllerURL, clientID, clientSecret string) *Client {
	return &Client{
		controllerURL: strings.TrimRight(controllerURL, "/"),
		clientID:      clientID,
		clientSecret:  clientSecret,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError represents a non-2xx response from the Controller API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("appdynamics api error: status %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an APIError with a 404 status.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// do executes an authenticated JSON request against the Controller API.
// path is relative to the controller URL. body, if non-nil, is marshaled as the
// JSON request payload. out, if non-nil, receives the decoded JSON response body.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	token, err := c.token(ctx)
	if err != nil {
		return fmt.Errorf("fetching access token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.controllerURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response body: %w", err)
		}
	}
	return nil
}

// doForm executes an authenticated request with a URL-encoded form body
// against the legacy (non-v1) Controller REST API, returning the raw
// response body. Used by the Events API, whose POST endpoints take form
// parameters rather than a JSON body and return a plain-text response
// rather than JSON.
func (c *Client) doForm(ctx context.Context, method, path string, form url.Values) ([]byte, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.controllerURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	return respBody, nil
}
