// Package authz calls auth-api's POST /authz/check to resolve a caller's identity (subject).
// Unlike catalog-api's role-gated authz, Game Room endpoints aren't role-based - every user manages
// their own library/wishlist/tables, and other users' collections are filtered by visibility
// rather than by role. This package only ever needs the caller's subject, and treats a missing
// or invalid token as an anonymous viewer rather than an error, since most Game Room reads are
// legitimately anonymous-accessible (anything public).
package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sweetrpg/common.go/logging"
)

// CheckResponse is the union of auth-api's allowed/denied /authz/check response shapes.
type CheckResponse struct {
	Allowed bool     `json:"allowed"`
	Roles   []string `json:"roles"`
	Sub     string   `json:"sub"`
	Reason  string   `json:"reason"`
}

// InvalidTokenError means auth-api rejected the bearer token itself.
type InvalidTokenError struct{}

func (InvalidTokenError) Error() string { return "authz: invalid or missing token" }

// Client calls auth-api's /authz/check endpoint.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds a Client against auth-api's base URL. An empty baseURL is accepted so the
// service can still start when AUTH_API_URL isn't configured - every Check call then fails,
// which ResolveViewer treats as anonymous.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

// Check verifies token against auth-api and returns the caller's allowed/roles/subject.
func (c *Client) Check(ctx context.Context, token, service string) (*CheckResponse, error) {
	body, err := json.Marshal(map[string]string{"service": service})
	if err != nil {
		return nil, fmt.Errorf("authz: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/authz/check", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("authz: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authz: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, InvalidTokenError{}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authz: unexpected status %d from auth-api", resp.StatusCode)
	}

	var out CheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("authz: decode response: %w", err)
	}

	logging.Logger.Debug("authz check", "allowed", out.Allowed, "sub", out.Sub)
	return &out, nil
}
