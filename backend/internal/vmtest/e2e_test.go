//go:build vm

package vmtest

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/server"
	"fast-bypass/internal/testutil"
)

func TestE2E_createVPNUser_onRouter(t *testing.T) {
	cfg := sharedCfg
	application, mt := Handler(t, cfg)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mgrToken := SeedManager(t, application, adminToken, "vmtst", "vmtst", 20)

	local := strconv.FormatInt(time.Now().UnixNano()%1e6, 10)
	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": local, "password": "Secret123!", "shared_users": 2,
		"assign_profile": true,
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	mikrotikName, _ := created["mikrotik_name"].(string)
	t.Cleanup(func() { _ = mt.RemoveUser(mikrotikName) })

	u, err := mt.GetUser(mikrotikName)
	if err != nil {
		t.Fatal(err)
	}
	if u.Comment != "panel:vmtst" {
		t.Fatalf("router comment %q", u.Comment)
	}
	profs, err := mt.ListUserProfiles(mikrotikName)
	if err != nil || len(profs) == 0 {
		t.Fatalf("profiles on router: %v", err)
	}
}

func TestE2E_managerListIsolation(t *testing.T) {
	cfg := sharedCfg
	application, mt := Handler(t, cfg)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	aliToken := SeedManager(t, application, adminToken, "vmali", "vmali", 10)
	bobToken := SeedManager(t, application, adminToken, "vmbob", "vmbob", 10)

	suffix := strconv.FormatInt(time.Now().UnixNano()%1e4, 10)
	legacy := "vmleg-" + suffix
	t.Cleanup(func() {
		_ = mt.RemoveUser("vmali-u-" + suffix)
		_ = mt.RemoveUser(legacy)
		_ = mt.RemoveUser("vmbob-only-" + suffix)
	})

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "u-" + suffix, "password": "Secret123!", "shared_users": 1,
	}, aliToken)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Body.String())
	}
	if err := mt.AddUser(legacy, "Secret123!", "panel:vmali", 1, false); err != nil {
		t.Fatal(err)
	}
	if err := mt.AddUser("vmbob-only-"+suffix, "Secret123!", "panel:vmbob", 1, false); err != nil {
		t.Fatal(err)
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, aliToken)
	var list map[string]any
	testutil.DecodeJSON(t, w, &list)
	items := list["items"].([]any)
	names := make(map[string]bool)
	for _, it := range items {
		row := it.(map[string]any)
		names[row["mikrotik_name"].(string)] = true
	}
	if !names["vmali-u-"+suffix] || !names[legacy] {
		t.Fatalf("ali should see own + legacy: %v", names)
	}
	if names["vmbob-only-"+suffix] {
		t.Fatal("ali must not see bob user")
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, bobToken)
	testutil.DecodeJSON(t, w, &list)
	items = list["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("bob list len %d", len(items))
	}
}

func TestE2E_deleteVPNUser_removesFromRouter(t *testing.T) {
	cfg := sharedCfg
	application, mt := Handler(t, cfg)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mgrToken := SeedManager(t, application, adminToken, "vmdel", "vmdel", 10)

	local := strconv.FormatInt(time.Now().UnixNano()%1e6, 10)
	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": local, "password": "Secret123!", "shared_users": 1,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))
	name := created["mikrotik_name"].(string)

	w = testutil.DoJSON(t, h, http.MethodDelete, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	if _, err := mt.GetUser(name); !errors.Is(err, mikrotik.ErrNotFound) {
		t.Fatalf("user should be removed from router, got err=%v", err)
	}
}
