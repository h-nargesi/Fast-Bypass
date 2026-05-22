package integration_test

import (
	"net/http"
	"strconv"
	"testing"

	"fast-bypass/internal/server"
	"fast-bypass/internal/testutil"
)

func assertNoLegacyVPNMetaFields(t *testing.T, m map[string]any) {
	t.Helper()
	for _, key := range []string{"local_name", "contact_phone", "contact_note"} {
		if _, ok := m[key]; ok {
			t.Fatalf("response must not include %q", key)
		}
	}
}

func TestManager_vpnUserDetail_noLegacyMetaFields(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "meta1", "password": "Secret123", "shared_users": 1,
		"contact_info": "tel @x", "notes": "panel note",
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	assertNoLegacyVPNMetaFields(t, created)

	id := int(created["id"].(float64))
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	assertNoLegacyVPNMetaFields(t, detail)
	if detail["contact_info"] != "tel @x" {
		t.Fatalf("contact_info: %+v", detail)
	}
	if detail["notes"] != "panel note" {
		t.Fatalf("notes: %+v", detail)
	}
}

func TestManager_patchContactInfo_clearsLegacyFields(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "patch1", "password": "Secret123", "shared_users": 1,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/vpn-users/"+strconv.Itoa(id), map[string]any{
		"contact_info": "email@test.com",
		"notes":        "updated",
	}, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	var patched map[string]any
	testutil.DecodeJSON(t, w, &patched)
	assertNoLegacyVPNMetaFields(t, patched)
	if patched["contact_info"] != "email@test.com" || patched["notes"] != "updated" {
		t.Fatalf("patched: %+v", patched)
	}
}

func TestAdmin_createOrphan_withoutManagerId(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "standalone", "password": "Secret123", "shared_users": 1,
		"contact_info": "wa.me/1",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("orphan create: %d %s", w.Code, w.Body.String())
	}
	var body map[string]any
	testutil.DecodeJSON(t, w, &body)
	assertNoLegacyVPNMetaFields(t, body)
	if body["mikrotik_name"] != "standalone" {
		t.Fatalf("name: %+v", body)
	}
	u, err := fake.GetUser("standalone")
	if err != nil || u.Comment != "" {
		t.Fatalf("router user: %+v err=%v", u, err)
	}

	id := int(body["id"].(float64))
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users/"+strconv.Itoa(id), nil, adminToken)
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	if detail["contact_info"] != "wa.me/1" {
		t.Fatalf("detail contact_info: %+v", detail)
	}
	if detail["manager_id"] != nil {
		t.Fatalf("manager_id should be null: %+v", detail["manager_id"])
	}
}

func TestAdmin_createOrphan_rejectsInvalidManagerId(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"manager_id": 99999, "local_name": "x", "password": "Secret123", "shared_users": 1,
	}, adminToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown manager, got %d %s", w.Code, w.Body.String())
	}
}

func TestManager_patchDisabled_onMikrotik(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "dis1", "password": "Secret123", "shared_users": 1,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/vpn-users/"+strconv.Itoa(id), map[string]any{
		"disabled": true,
	}, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch disabled: %d %s", w.Code, w.Body.String())
	}
	u, _ := fake.GetUser("ali-dis1")
	if !u.Disabled {
		t.Fatal("router user should be disabled")
	}

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/vpn-users/"+strconv.Itoa(id), map[string]any{
		"disabled": false,
	}, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	u, _ = fake.GetUser("ali-dis1")
	if u.Disabled {
		t.Fatal("router user should be enabled again")
	}
}

func TestManager_create_withDisabled_setsRouterFlag(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "newoff", "password": "Secret123", "shared_users": 1, "disabled": true,
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	if created["disabled"] != true {
		t.Fatalf("created: %+v", created)
	}
	u, _ := fake.GetUser("ali-newoff")
	if !u.Disabled {
		t.Fatal("router user should be created disabled")
	}
}

func TestManager_create_storesContactInfo(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	// contact_phone/contact_note are ignored by JSON decoder; create still works with local_name only
	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "leg1", "password": "Secret123", "shared_users": 1,
		"contact_info": "ok",
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	if detail["contact_info"] != "ok" {
		t.Fatalf("stored contact_info: %+v", detail)
	}
}
