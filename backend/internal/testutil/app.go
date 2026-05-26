package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"fast-bypass/internal/app"
	"fast-bypass/internal/config"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/store"
)

// TestConfig returns config suitable for integration tests.
func TestConfig(dbPath string) config.Config {
	return config.Config{
		SQLitePath:          dbPath,
		TZ:                  "UTC",
		MikrotikFake:        true,
		MikrotikCacheTTL:    time.Second,
		JWTSecret:           "integration-test-secret",
		JWTAccessTTL:        time.Hour,
		JWTRefreshTTL:       24 * time.Hour,
		AdminUsername:       "admin",
		AdminPassword:       "AdminPass1",
		DefaultProfile:      "profile-open-2M-30d",
		UsernamePrefixSep:   "-",
		UsernameLocalMaxLen: 24,
		SharedUsersMax:      20,
		CORSOrigins:         []string{"http://localhost:4200"},
		OpenVPNTemplatePath: "../config/client-template.ovpn",
		OpenVPNDownloadURL:  "http://example.test/dl/",
		OpenVPNKeyPassword:  "EnvKeyPass123",
	}
}

// NewTestApp opens a temp DB, bootstraps admin, returns app and cleanup.
func NewTestApp(t *testing.T) (*app.App, *store.Store, mikrotik.Client) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "panel.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := TestConfig(dbPath)
	application, err := app.New(cfg, log, st)
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	fake := mikrotik.NewFake()
	application.UseMikrotikClient(fake)
	if err := application.BootstrapAdmin(context.Background()); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return application, st, fake
}

// SeedManager creates a manager and returns id + login token.
func SeedManager(t *testing.T, h http.Handler, adminToken, username, slug string, quota int) (int64, string) {
	t.Helper()
	w := DoJSON(t, h, http.MethodPost, "/api/v1/admin/managers", map[string]any{
		"username": username, "password": "ManagerPass1",
		"display_name": username, "slug": slug, "quota": quota,
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create manager: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	DecodeJSON(t, w, &resp)
	mgrToken := LoginToken(t, h, username, "ManagerPass1")
	return resp.ID, mgrToken
}
