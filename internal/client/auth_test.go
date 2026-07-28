package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenFetchAndCache(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/controller/api/oauth/access_token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls++
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Fatalf("grant_type = %q, want client_credentials", got)
		}
		if got := r.Form.Get("client_id"); got != "client@account" {
			t.Fatalf("client_id = %q, want client@account", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: "tok-1", ExpiresIn: 300})
	}))
	defer srv.Close()

	c := New(srv.URL, "client@account", "secret")

	tok1, err := c.token(context.Background())
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if tok1 != "tok-1" {
		t.Fatalf("token = %q, want tok-1", tok1)
	}

	tok2, err := c.token(context.Background())
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if tok2 != "tok-1" {
		t.Fatalf("token = %q, want cached tok-1", tok2)
	}
	if calls != 1 {
		t.Fatalf("token endpoint called %d times, want 1 (second call should hit cache)", calls)
	}
}

func TestTokenRefreshNearExpiry(t *testing.T) {
	responses := []string{"tok-a", "tok-b"}
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := responses[call]
		if call < len(responses)-1 {
			call++
		}
		w.Header().Set("Content-Type", "application/json")
		// expires_in shorter than tokenRefreshSkew forces every call to refetch.
		_ = json.NewEncoder(w).Encode(tokenResponse{AccessToken: tok, ExpiresIn: 1})
	}))
	defer srv.Close()

	c := New(srv.URL, "client@account", "secret")

	tok1, err := c.token(context.Background())
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if tok1 != "tok-a" {
		t.Fatalf("token = %q, want tok-a", tok1)
	}

	tok2, err := c.token(context.Background())
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if tok2 != "tok-b" {
		t.Fatalf("token = %q, want tok-b (should have refreshed since expires_in < skew)", tok2)
	}
}

func TestTokenErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "bad", "creds")
	_, err := c.token(context.Background())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}
