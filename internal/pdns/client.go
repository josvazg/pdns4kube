// Package pdns provides a minimal client for the PowerDNS Authoritative
// v4 HTTP API, scoped to the zones endpoints of the "localhost" server.
package pdns

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
	"time"
)

// maxErrorBodySize caps how many bytes of a non-2xx response body are
// included in returned errors.
const maxErrorBodySize = 4 << 10 // 4 KiB

// Zone is a request model for creating or updating a zone.
type Zone struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Nameservers []string `json:"nameservers,omitempty"`
}

// Client talks to the PowerDNS Authoritative v4 API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient returns a Client for the API at baseURL. If httpClient is nil,
// a sensible default with a timeout is used.
func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// CreateZone creates a new zone. If the API reports the zone already
// exists (409 Conflict), it retries safely by updating the existing zone
// instead.
func (c *Client) CreateZone(ctx context.Context, z Zone) error {
	u := c.zonesURL("")
	resp, err := c.do(ctx, http.MethodPost, u, z)
	if err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return c.UpdateZone(ctx, z.Name, z)
		}
		return err
	}
	defer resp.Close()
	_, err = io.Copy(io.Discard, resp)
	return err
}

// UpdateZone replaces the configuration of the zone with the given ID.
func (c *Client) UpdateZone(ctx context.Context, zoneID string, z Zone) error {
	resp, err := c.do(ctx, http.MethodPut, c.zonesURL(zoneID), z)
	if err != nil {
		return err
	}
	defer resp.Close()
	_, err = io.Copy(io.Discard, resp)
	return err
}

// DeleteZone removes the zone with the given ID. A 404 is treated as
// idempotent success.
func (c *Client) DeleteZone(ctx context.Context, zoneID string) error {
	resp, err := c.do(ctx, http.MethodDelete, c.zonesURL(zoneID), nil)
	if err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return err
	}
	defer resp.Close()
	_, err = io.Copy(io.Discard, resp)
	return err
}

// zonesURL returns the zones collection URL, optionally addressing a
// specific zone by its (URL-escaped) ID.
func (c *Client) zonesURL(zoneID string) string {
	u := c.baseURL + "/api/v1/servers/localhost/zones"
	if zoneID != "" {
		u += "/" + url.PathEscape(zoneID)
	}
	return u
}

// apiError is returned for non-2xx responses. It includes the HTTP status
// code and up to maxErrorBodySize bytes of the response body.
type apiError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("pdns: %s %s: unexpected status %d", e.Method, e.URL, e.StatusCode)
	}
	return fmt.Sprintf("pdns: %s %s: unexpected status %d: %s", e.Method, e.URL, e.StatusCode, e.Body)
}

// do performs an HTTP request, setting the API key header and JSON content
// type when a body is sent, and returns the response body reader (or an
// *apiError populated with status and a capped response body for non-2xx).
// On success the caller owns the returned body and must close it; on error
// the response body is always closed before do returns.
func (c *Client) do(ctx context.Context, method, u string, body any) (io.ReadCloser, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("pdns: encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("pdns: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pdns: %s %s: %w", method, u, err)
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("pdns: %s %s: empty response body", method, u)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		return nil, apiErrorResponse(method, u, resp)
	}
	return resp.Body, nil
}

// apiErrorResponse converts a non-2xx response into an *apiError that
// includes the status code and up to maxErrorBodySize bytes of the
// response body. The caller is responsible for closing resp.Body.
func apiErrorResponse(method, u string, resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if err != nil {
		return &apiError{Method: method, URL: u, StatusCode: resp.StatusCode}
	}
	return &apiError{Method: method, URL: u, StatusCode: resp.StatusCode, Body: string(body)}
}
