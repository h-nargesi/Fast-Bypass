//go:build vm

package vmtest

import (
	"fmt"
	"testing"
	"time"

	"fast-bypass/internal/mikrotik"
)

func TestRouterOS_pingAndProfile(t *testing.T) {
	cfg := sharedCfg
	mt, err := mikrotik.NewFromConfig(cfg.Config)
	if err != nil {
		t.Fatal(err)
	}
	if err := mt.Ping(); err != nil {
		t.Fatal(err)
	}
}

func TestRouterOS_createUser_panelComment(t *testing.T) {
	cfg := sharedCfg
	mt, err := mikrotik.NewFromConfig(cfg.Config)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("vmtest-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = mt.RemoveUser(name) })

	if err := mt.AddUser(name, "Secret123!", "panel=vmtst", 2, false); err != nil {
		t.Fatal(err)
	}
	u, err := mt.GetUser(name)
	if err != nil {
		t.Fatal(err)
	}
	if u.Comment != "panel=vmtst" {
		t.Fatalf("comment: got %q want panel=vmtst", u.Comment)
	}
	if u.SharedUsers != 2 {
		t.Fatalf("shared-users: got %d want 2", u.SharedUsers)
	}
}

func TestRouterOS_assignProfile(t *testing.T) {
	cfg := sharedCfg
	mt, err := mikrotik.NewFromConfig(cfg.Config)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("vmprof-%d", time.Now().UnixNano())
	profile := cfg.DefaultProfile
	t.Cleanup(func() { _ = mt.RemoveUser(name) })

	if err := mt.AddUser(name, "Secret123!", "panel=vmtst", 1, false); err != nil {
		t.Fatal(err)
	}
	if err := mt.AddUserProfile(name, profile); err != nil {
		t.Fatal(err)
	}
	profs, err := mt.ListUserProfiles(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(profs) == 0 {
		t.Fatal("expected at least one user-profile")
	}
}
