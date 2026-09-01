package authz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sweetrpg/common.go/logging"
)

func TestMain(m *testing.M) {
	logging.Init()
	os.Exit(m.Run())
}

func TestResolveUserID(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantUserID string
	}{
		{name: "provisioned user returns canonical id", status: http.StatusOK, body: `{"user_id":"11111111-2222-3333-4444-555555555555"}`, wantUserID: "11111111-2222-3333-4444-555555555555"},
		{name: "unprovisioned subject (404) resolves to anonymous", status: http.StatusNotFound, body: `{"error":"not found"}`, wantUserID: ""},
		{name: "rejected token (401) resolves to anonymous", status: http.StatusUnauthorized, body: ``, wantUserID: ""},
		{name: "users-api error (500) resolves to anonymous", status: http.StatusInternalServerError, body: ``, wantUserID: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAuth, gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotPath = r.URL.Path
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := &Client{usersBaseURL: srv.URL, http: srv.Client()}
			got := c.ResolveUserID(context.Background(), "test-token")

			if got != tt.wantUserID {
				t.Errorf("ResolveUserID = %q, want %q", got, tt.wantUserID)
			}
			if gotPath != "/profile" {
				t.Errorf("requested path = %q, want /profile", gotPath)
			}
			if gotAuth != "Bearer test-token" {
				t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
			}
		})
	}
}

func TestResolveUserIDNoConfig(t *testing.T) {
	c := &Client{usersBaseURL: "", http: http.DefaultClient}
	if got := c.ResolveUserID(context.Background(), "test-token"); got != "" {
		t.Errorf("ResolveUserID with no usersBaseURL = %q, want \"\"", got)
	}

	c = &Client{usersBaseURL: "http://users-api.invalid", http: http.DefaultClient}
	if got := c.ResolveUserID(context.Background(), ""); got != "" {
		t.Errorf("ResolveUserID with no token = %q, want \"\"", got)
	}
}
