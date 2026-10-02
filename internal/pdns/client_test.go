package pdns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateZone(t *testing.T) {
	srv, rec := capture(t, http.StatusCreated, "")
	c := NewClient(srv.URL, "secret", srv.Client())

	err := c.CreateZone(context.Background(), Zone{
		Name:        "example.org.",
		Kind:        "Native",
		Nameservers: []string{"ns1.example.org."},
	})
	if err != nil {
		t.Fatalf("CreateZone: %v", err)
	}

	if rec.method != http.MethodPost {
		t.Errorf("method = %q, want POST", rec.method)
	}
	if rec.path != "/api/v1/servers/localhost/zones" {
		t.Errorf("path = %q", rec.path)
	}
	if rec.apiKey != "secret" {
		t.Errorf("X-API-Key = %q, want secret", rec.apiKey)
	}
	if rec.contentType != "application/json" {
		t.Errorf("Content-Type = %q", rec.contentType)
	}
	var got Zone
	if err := json.Unmarshal(rec.body, &got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if got.Name != "example.org." || got.Kind != "Native" {
		t.Errorf("body zone = %+v", got)
	}
}

func TestCreateZoneConflictFallsBackToUpdate(t *testing.T) {
	var methods []string
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		paths = append(paths, r.URL.Path)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error": "Conflict"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", srv.Client())
	if err := c.CreateZone(context.Background(), Zone{Name: "example.org.", Kind: "Native"}); err != nil {
		t.Fatalf("CreateZone: %v", err)
	}
	if len(methods) != 2 || methods[0] != http.MethodPost || methods[1] != http.MethodPut {
		t.Errorf("methods = %v, want [POST PUT]", methods)
	}
	if len(paths) != 2 || paths[1] != "/api/v1/servers/localhost/zones/example.org." {
		t.Errorf("paths = %v", paths)
	}
}

func TestUpdateZone(t *testing.T) {
	srv, rec := capture(t, http.StatusNoContent, "")
	c := NewClient(srv.URL, "k", srv.Client())

	err := c.UpdateZone(context.Background(), "example org..Native", Zone{Name: "example.org.", Kind: "Native"})
	if err != nil {
		t.Fatalf("UpdateZone: %v", err)
	}

	if rec.method != http.MethodPut {
		t.Errorf("method = %q, want PUT", rec.method)
	}
	if rec.path != "/api/v1/servers/localhost/zones/example%20org..Native" {
		t.Errorf("path = %q (zone ID not escaped)", rec.path)
	}
}

func TestDeleteZone(t *testing.T) {
	srv, rec := capture(t, http.StatusNoContent, "")
	c := NewClient(srv.URL, "k", srv.Client())

	err := c.DeleteZone(context.Background(), "example.org.")
	if err != nil {
		t.Fatalf("DeleteZone: %v", err)
	}
	if rec.method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", rec.method)
	}
	if rec.path != "/api/v1/servers/localhost/zones/example.org." {
		t.Errorf("path = %q", rec.path)
	}
	if rec.contentType != "" {
		t.Errorf("Content-Type = %q, want empty for bodyless request", rec.contentType)
	}
}

func TestDeleteZoneNotFoundIsIdempotent(t *testing.T) {
	srv, _ := capture(t, http.StatusNotFound, `{"error": "not found"}`)
	c := NewClient(srv.URL, "k", srv.Client())

	if err := c.DeleteZone(context.Background(), "example.org."); err != nil {
		t.Fatalf("DeleteZone on 404: %v", err)
	}
}

func TestNon2xxErrorIncludesStatusAndBody(t *testing.T) {
	srv, _ := capture(t, http.StatusBadRequest, `{"error": "Domain 'example.org.' is not a valid DNS name"}`)
	c := NewClient(srv.URL, "k", srv.Client())

	err := c.CreateZone(context.Background(), Zone{Name: "example.org.", Kind: "Native"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "not a valid DNS name") {
		t.Errorf("error = %v, want status 400 and body text", err)
	}
}

func TestErrorBodyIsCapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(make([]byte, 1<<20)) // 1 MiB of zeros
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", srv.Client())
	err := c.DeleteZone(context.Background(), "example.org.")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	// Body is capped at 4 KiB; allow room for the error message prefix.
	if len(err.Error()) > 4200 {
		t.Errorf("error message too large: %d bytes", len(err.Error()))
	}
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %T is not *apiError", err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestDoClosesBodyOnceOnError(t *testing.T) {
	var closes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k", srv.Client())
	// Wrap the transport so we can count how many times the response body
	// of a non-2xx response is closed.
	c.httpClient = &http.Client{
		Transport: closeCountTransport{next: srv.Client().Transport, closes: &closes},
	}

	_, err := c.do(context.Background(), http.MethodGet, srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if closes != 1 {
		t.Errorf("body closed %d times, want 1", closes)
	}
}

type closeCountTransport struct {
	next   http.RoundTripper
	closes *int
}

func (t closeCountTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		resp.Body = &countingReadCloser{ReadCloser: resp.Body, closes: t.closes}
	}
	return resp, err
}

type countingReadCloser struct {
	io.ReadCloser
	closes *int
}

func (c *countingReadCloser) Close() error {
	*c.closes++
	return c.ReadCloser.Close()
}

// capture is a helper httptest server that records the request and replies
// with the given status and body.
func capture(t *testing.T, wantStatus int, wantBody string) (*httptest.Server, *record) {
	t.Helper()
	rec := &record{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rec.method = r.Method
		rec.path = r.URL.EscapedPath()
		rec.apiKey = r.Header.Get("X-API-Key")
		rec.contentType = r.Header.Get("Content-Type")
		rec.body = b
		w.WriteHeader(wantStatus)
		_, _ = w.Write([]byte(wantBody))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

type record struct {
	method      string
	path        string
	apiKey      string
	contentType string
	body        []byte
}
