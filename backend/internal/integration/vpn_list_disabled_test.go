package integration_test

import (
	"net/http"
	"strconv"
	"testing"

	"fast-bypass/internal/server"
	"fast-bypass/internal/testutil"
)

func TestManager_listVPNUsers_includesDisabled(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "on", "password": "Secret123", "shared_users": 1,
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Body.String())
	}
	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "off", "password": "Secret123", "shared_users": 1, "disabled": true,
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Body.String())
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	items, _ := resp["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items: %+v", items)
	}
	byName := map[string]map[string]any{}
	for _, it := range items {
		row := it.(map[string]any)
		byName[row["mikrotik_name"].(string)] = row
	}
	if byName["ali-on"]["disabled"] != false {
		t.Fatalf("ali-on disabled: %+v", byName["ali-on"])
	}
	if byName["ali-off"]["disabled"] != true {
		t.Fatalf("ali-off disabled: %+v", byName["ali-off"])
	}
	u, _ := fake.GetUser("ali-off")
	if !u.Disabled {
		t.Fatal("fake router user should be disabled")
	}
}

func TestManager_listVPNUsers_activeOnlyExcludesDisabled(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "act", "password": "Secret123", "shared_users": 1, "assign_profile": true,
	}, mgrToken)
	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "inact", "password": "Secret123", "shared_users": 1,
		"disabled": true, "assign_profile": true,
	}, mgrToken)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users?active_only=true", nil, mgrToken)
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	items, _ := resp["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("active_only: want 1 item, got %+v", items)
	}
	row := items[0].(map[string]any)
	if row["mikrotik_name"] != "ali-act" {
		t.Fatalf("row: %+v", row)
	}
}

func TestAdmin_listVPNUsers_includesDisabled(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mid, _ := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"manager_id": mid, "local_name": "admoff", "password": "Secret123", "shared_users": 1,
		"disabled": true,
	}, adminToken)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users?manager_id="+strconv.FormatInt(mid, 10), nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	found := false
	for _, it := range resp["items"].([]any) {
		row := it.(map[string]any)
		if row["mikrotik_name"] == "ali-admoff" {
			found = true
			if row["disabled"] != true {
				t.Fatalf("disabled: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("ali-admoff not in admin list")
	}
}
