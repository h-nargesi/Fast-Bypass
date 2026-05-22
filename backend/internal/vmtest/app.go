//go:build vm

package vmtest

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"fast-bypass/internal/app"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/server"
	"fast-bypass/internal/store"
	"fast-bypass/internal/testutil"
)

// NewApp boots panel with a real RouterOS client (no fake).
func NewApp(t *testing.T, cfg VMConfig) (*app.App, mikrotik.Client, *store.Store) {
	t.Helper()
	if reason := cfg.SkipReason(); reason != "" {
		t.Skip(reason)
	}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "panel.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	panelCfg := cfg.Config
	panelCfg.SQLitePath = dbPath
	panelCfg.MikrotikFake = false
	panelCfg.MikrotikCacheTTL = time.Second
	panelCfg.JWTSecret = "vm-test-secret"
	panelCfg.JWTAccessTTL = time.Hour
	panelCfg.AdminUsername = "admin"
	panelCfg.AdminPassword = "AdminPass1"
	panelCfg.DefaultProfile = env("DEFAULT_PROFILE", "profile-open-2M-30d")
	panelCfg.UsernamePrefixSep = "-"
	panelCfg.UsernameLocalMaxLen = 24
	panelCfg.SharedUsersMax = 20
	panelCfg.OpenVPNTemplatePath = "../config/client-template.ovpn"
	application, err := app.New(panelCfg, log, st)
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	inner, ok := unwrapRouter(application.MT)
	if !ok {
		t.Fatal("expected RouterOS client, got fake or unknown wrapper")
	}
	if err := inner.Ping(); err != nil {
		_ = st.Close()
		t.Fatalf("mikrotik ping: %v", err)
	}
	if err := application.BootstrapAdmin(context.Background()); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return application, inner, st
}

func unwrapRouter(c mikrotik.Client) (*mikrotik.RouterOS, bool) {
	if cached, ok := c.(*mikrotik.CachedClient); ok {
		if r, ok := cached.Inner(); ok {
			return r, true
		}
	}
	if r, ok := c.(*mikrotik.RouterOS); ok {
		return r, true
	}
	return nil, false
}

// Handler returns HTTP handler for the panel API.
func Handler(t *testing.T, cfg VMConfig) (*app.App, mikrotik.Client) {
	t.Helper()
	a, mt, _ := NewApp(t, cfg)
	return a, mt
}

// SeedManager creates manager and returns token (reuses testutil).
func SeedManager(t *testing.T, a *app.App, adminToken, username, slug string, quota int) string {
	t.Helper()
	h := server.New(a)
	_, tok := testutil.SeedManager(t, h, adminToken, username, slug, quota)
	return tok
}
