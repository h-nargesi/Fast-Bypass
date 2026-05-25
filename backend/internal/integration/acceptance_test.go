package integration_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"fast-bypass/internal/server"
	"fast-bypass/internal/testutil"
)

// Acceptance tests mapped to docs/business-rules.md (فاز ۱).

func TestBootstrap_adminCanLogin(t *testing.T) {
	application, st, _ := testutil.NewTestApp(t)
	h := server.New(application)
	n, _ := st.AdminCount(context.Background())
	if n != 1 {
		t.Fatalf("expected 1 admin, got %d", n)
	}
	testutil.LoginToken(t, h, "admin", "AdminPass1")
}

func TestManager_listIsolation_and_legacyComment(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, aliToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
	_, bobToken := testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)

	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "u1", "password": "Secret123", "shared_users": 1,
	}, aliToken)

	_ = fake.AddUser("reza", "Secret123", "panel=ali", 1, false)
	_ = fake.AddUser("bob-only", "Secret123", "panel=bob", 1, false)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, aliToken)
	var list map[string]any
	testutil.DecodeJSON(t, w, &list)
	names := itemNames(list["items"].([]any))
	if len(names) != 2 || !contains(names, "ali-u1") || !contains(names, "reza") {
		t.Fatalf("ali list: %v", names)
	}
	if contains(names, "bob-only") {
		t.Fatal("ali must not see bob-only")
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, bobToken)
	testutil.DecodeJSON(t, w, &list)
	names = itemNames(list["items"].([]any))
	if len(names) != 1 || names[0] != "bob-only" {
		t.Fatalf("bob list: %v", names)
	}
}

func TestManager_detail_noMikrotikComment_hasConnectionBundle(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "c1", "password": "Secret123", "shared_users": 2, "assign_profile": true,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	if _, ok := detail["mikrotik_comment"]; ok {
		t.Fatal("manager must not see mikrotik_comment")
	}
	bundle, ok := detail["connection_bundle"].(map[string]any)
	if !ok || bundle["username"] != "ali-c1" || bundle["password"] != "Secret123" {
		t.Fatalf("connection_bundle: %+v", bundle)
	}
	for _, key := range []string{"local_name", "contact_phone", "contact_note", "mikrotik_comment"} {
		if _, ok := detail[key]; ok {
			t.Fatalf("manager detail must not include %q", key)
		}
	}
	if _, ok := detail["contact_info"]; !ok {
		t.Fatal("detail should include contact_info key (nullable)")
	}
}

func TestCreateVPN_setsPanelComment_and_activationAmount(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "paid", "password": "Secret123", "shared_users": 2,
		"assign_profile": true, "amount_paid": 150000, "currency": "IRR",
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Body.String())
	}
	var paidCreated map[string]any
	testutil.DecodeJSON(t, w, &paidCreated)
	u, _ := fake.GetUser("ali-paid")
	if u.Comment != "panel=ali" {
		t.Fatalf("comment on router: %q", u.Comment)
	}
	paidID := int(paidCreated["id"].(float64))
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(paidID), nil, mgrToken)
	var paidDetail map[string]any
	testutil.DecodeJSON(t, w, &paidDetail)
	paidActs := paidDetail["activations"].([]any)
	if len(paidActs) == 0 || paidActs[0].(map[string]any)["amount_paid"].(float64) != 150000 {
		t.Fatalf("amount_paid snapshot: %+v", paidActs)
	}

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "free", "password": "Secret123", "shared_users": 1, "assign_profile": true,
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Body.String())
	}
}

func TestManager_disabledCannotLogin(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mid, _ := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mid, 10), map[string]any{
		"is_active": false,
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": "ali", "password": "ManagerPass1",
	}, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("disabled login: %d %s", w.Code, w.Body.String())
	}
	var errBody map[string]any
	testutil.DecodeJSON(t, w, &errBody)
	errObj := errBody["error"].(map[string]any)
	if errObj["code"] != "MANAGER_DISABLED" {
		t.Fatalf("code: %+v", errObj)
	}
}

func TestAdmin_quotaBelowUsage_rejected(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mid, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "big", "password": "Secret123", "shared_users": 6, "assign_profile": true,
	}, mgrToken)

	w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mid, 10), map[string]any{
		"quota": 3,
	}, adminToken)
	if w.Code != http.StatusConflict {
		t.Fatalf("QUOTA_BELOW_USAGE: %d %s", w.Code, w.Body.String())
	}
}

func TestAdmin_vpnUsers_orphanFilter_and_ownerFields(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	_ = fake.AddUser("guest99", "Secret123", "", 2, false)
	_ = fake.AddUser("ali-z", "Secret123", "panel=ali", 1, false)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users?orphan=true", nil, adminToken)
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	items := resp["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("orphan items: %+v", items)
	}
	row := items[0].(map[string]any)
	if row["manager_id"] != nil {
		t.Fatalf("orphan row: %+v", row)
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users", nil, adminToken)
	testutil.DecodeJSON(t, w, &resp)
	found := false
	for _, it := range resp["items"].([]any) {
		row = it.(map[string]any)
		if row["mikrotik_name"] == "ali-z" {
			found = true
			if row["manager_slug"] != "ali" || row["mikrotik_comment"] != "panel=ali" {
				t.Fatalf("owner fields: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("ali-z not in admin list")
	}
}

func TestAdmin_stats_totalsAndByManager(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
	testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)

	prof := "profile-open-2M-30d"
	_ = fake.AddUser("ali-a", "Secret123", "panel:ali", 2, false)
	_ = fake.AddUserProfile("ali-a", prof)
	_ = fake.AddUser("ali-off", "Secret123", "panel:ali", 3, true)
	_ = fake.AddUserProfile("ali-off", prof)
	_ = fake.AddUser("bob-x", "Secret123", "panel:bob", 4, false)
	_ = fake.AddUserProfile("bob-x", prof)
	_ = fake.AddUser("orphan1", "Secret123", "", 1, false)
	_ = fake.AddUserProfile("orphan1", prof)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/stats", nil, adminToken)
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	if int(resp["manager_count"].(float64)) != 2 {
		t.Fatalf("manager_count: %+v", resp["manager_count"])
	}
	totals := resp["totals"].(map[string]any)
	if int(totals["vpn_users"].(float64)) != 3 || int(totals["connections"].(float64)) != 7 {
		t.Fatalf("totals: %+v", totals)
	}
	orphan := resp["orphan"].(map[string]any)
	if int(orphan["vpn_users"].(float64)) != 1 || int(orphan["connections"].(float64)) != 1 {
		t.Fatalf("orphan: %+v", orphan)
	}
	var aliConn, bobConn int
	for _, it := range resp["by_manager"].([]any) {
		row := it.(map[string]any)
		switch int(row["connections"].(float64)) {
		case 2:
			if row["display_name"] != "ali" {
				t.Fatalf("ali row: %+v", row)
			}
			aliConn = 2
		case 4:
			if row["display_name"] != "bob" {
				t.Fatalf("bob row: %+v", row)
			}
			bobConn = 4
		}
	}
	if aliConn != 2 || bobConn != 4 {
		t.Fatalf("by_manager connections: ali=%d bob=%d", aliConn, bobConn)
	}
}

func TestAdmin_vpnUser_createPatchDelete(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mid, _ := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"manager_id": mid, "local_name": "adm1", "password": "Secret123", "shared_users": 1,
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin create: %d %s", w.Code, w.Body.String())
	}
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))
	u, _ := fake.GetUser("ali-adm1")
	if u.Comment != "panel=ali" {
		t.Fatalf("comment: %q", u.Comment)
	}

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/vpn-users/"+strconv.Itoa(id), map[string]any{
		"contact_info": "admin contact",
		"notes":        "from admin",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("admin patch: %d %s", w.Code, w.Body.String())
	}
	var patched map[string]any
	testutil.DecodeJSON(t, w, &patched)
	if patched["notes"] != "from admin" {
		t.Fatalf("notes: %+v", patched)
	}
	if patched["contact_info"] != "admin contact" {
		t.Fatalf("contact_info: %+v", patched)
	}
	for _, key := range []string{"local_name", "contact_phone", "contact_note"} {
		if _, ok := patched[key]; ok {
			t.Fatalf("patched response must not include %q", key)
		}
	}
	if _, ok := patched["mikrotik_comment"]; !ok {
		t.Fatal("admin detail should include mikrotik_comment")
	}

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "orphan01", "password": "Secret123", "shared_users": 1,
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin orphan create: %d %s", w.Code, w.Body.String())
	}
	var orphan map[string]any
	testutil.DecodeJSON(t, w, &orphan)
	if orphan["mikrotik_name"] != "orphan01" {
		t.Fatalf("orphan name: %+v", orphan)
	}
	if _, ok := orphan["local_name"]; ok {
		t.Fatal("response must not include local_name")
	}
	uOrphan, _ := fake.GetUser("orphan01")
	if uOrphan.Comment != "" {
		t.Fatalf("orphan comment: %q", uOrphan.Comment)
	}

	w = testutil.DoJSON(t, h, http.MethodDelete, "/api/v1/admin/vpn-users/"+strconv.Itoa(id), nil, adminToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("admin delete: %d %s", w.Code, w.Body.String())
	}
	if _, err := fake.GetUser("ali-adm1"); err == nil {
		t.Fatal("user should be removed from router")
	}
}

func TestAdmin_ownerMismatch_flag(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
	testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)

	_ = fake.AddUser("ali-conflict", "Secret123", "panel=bob", 1, false)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users", nil, adminToken)
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	for _, it := range resp["items"].([]any) {
		row := it.(map[string]any)
		if row["mikrotik_name"] == "ali-conflict" {
			if row["owner_mismatch"] != true {
				t.Fatalf("expected owner_mismatch: %+v", row)
			}
			return
		}
	}
	t.Fatal("ali-conflict not found")
}

func TestManager_NOT_OWNER(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, aliToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
	_, bobToken := testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "mine", "password": "Secret123", "shared_users": 1,
	}, aliToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, bobToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("NOT_OWNER: %d %s", w.Code, w.Body.String())
	}
}

func TestPATCH_me_displayName_and_rejectQuota(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/me", map[string]any{
		"display_name": "علی جدید",
	}, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	var me map[string]any
	testutil.DecodeJSON(t, w, &me)
	if me["display_name"] != "علی جدید" {
		t.Fatalf("me: %+v", me)
	}

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/me", map[string]any{"quota": 99}, mgrToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("reject extra field: %d", w.Code)
	}
}

func TestManager_renewals_onlyOwn(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, aliToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
	_, bobToken := testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)

	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "r1", "password": "Secret123", "shared_users": 1, "assign_profile": true,
	}, aliToken)
	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "r2", "password": "Secret123", "shared_users": 1, "assign_profile": true,
	}, bobToken)

	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/renewals", nil, aliToken)
	var resp map[string]any
	testutil.DecodeJSON(t, w, &resp)
	if resp["can_settle"] != false {
		t.Fatal("manager can_settle must be false")
	}
	for _, it := range resp["items"].([]any) {
		row := it.(map[string]any)
		name := row["mikrotik_name"].(string)
		if !strings.HasPrefix(name, "ali-") {
			t.Fatalf("foreign renewal: %s", name)
		}
	}
}

func TestDeleteVPNUser(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "del", "password": "Secret123", "shared_users": 1,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodDelete, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	if w.Code != http.StatusNoContent {
		t.Fatal(w.Code)
	}
	if _, err := fake.GetUser("ali-del"); err == nil {
		t.Fatal("user should be removed from router")
	}
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	if w.Code != http.StatusNotFound {
		t.Fatalf("detail after delete: %d", w.Code)
	}
}

func TestAssign_storesSharedUsersSnapshot(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "snap", "password": "Secret123", "shared_users": 3, "assign_profile": true,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	acts := detail["activations"].([]any)
	if len(acts) == 0 {
		t.Fatal("no activations")
	}
	if int(acts[0].(map[string]any)["shared_users"].(float64)) != 3 {
		t.Fatalf("snapshot: %+v", acts[0])
	}
}

func TestSharedUsers_syncPatchAndRenewalsLive(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "sync1", "password": "Secret123", "shared_users": 2, "assign_profile": true,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))
	name := created["mikrotik_name"].(string)

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/vpn-users/"+strconv.Itoa(id), map[string]any{
		"shared_users": 4,
	}, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	acts := detail["activations"].([]any)
	if int(acts[0].(map[string]any)["shared_users"].(float64)) != 4 {
		t.Fatalf("activation after patch: %+v", acts[0])
	}

	su := 5
	if err := fake.SetUser(name, nil, &su, "panel=ali", nil); err != nil {
		t.Fatal(err)
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/renewals", nil, mgrToken)
	var renewals map[string]any
	testutil.DecodeJSON(t, w, &renewals)
	items := renewals["items"].([]any)
	found := false
	for _, it := range items {
		row := it.(map[string]any)
		if row["mikrotik_name"] == name {
			found = true
			if int(row["shared_users"].(float64)) != 5 {
				t.Fatalf("renewals live: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("renewal row not found")
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	testutil.DecodeJSON(t, w, &detail)
	acts = detail["activations"].([]any)
	if int(acts[0].(map[string]any)["shared_users"].(float64)) != 5 {
		t.Fatalf("activation healed on read: %+v", acts[0])
	}
}

func TestMikrotikList_cacheAndRefresh(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "cached", "password": "Secret123", "shared_users": 1,
	}, mgrToken)

	// پر کردن کش با یک کاربر
	w := testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, mgrToken)
	var list map[string]any
	testutil.DecodeJSON(t, w, &list)
	if len(list["items"].([]any)) != 1 {
		t.Fatalf("initial list: %+v", list["items"])
	}

	// تغییر مستقیم روی fake بدون عبور از API — کش باید قدیمی بماند
	_ = fake.AddUser("ali-extra", "Secret123", "panel=ali", 1, false)

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users", nil, mgrToken)
	testutil.DecodeJSON(t, w, &list)
	if len(list["items"].([]any)) != 1 {
		t.Fatalf("cached list: %+v", list["items"])
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users?refresh=true", nil, mgrToken)
	testutil.DecodeJSON(t, w, &list)
	if len(list["items"].([]any)) != 2 {
		t.Fatalf("refresh list: %+v", list["items"])
	}
}

func TestQuota_increaseSharedUsers_rejected(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	// quota=2, user active with shared_users=2 → افزایش به 3 باید رد شود (2-2+3 > 2)
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 2)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "q1", "password": "Secret123", "shared_users": 2, "assign_profile": true,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/vpn-users/"+strconv.Itoa(id), map[string]any{
		"shared_users": 3,
	}, mgrToken)
	if w.Code != http.StatusConflict {
		t.Fatalf("increase shared_users: %d %s", w.Code, w.Body.String())
	}
}

func TestDownloadOvpn(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "ovpn", "password": "Secret123", "shared_users": 1,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id)+"/ovpn", nil, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatalf("ovpn: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ali-ovpn") || !strings.Contains(w.Body.String(), "Secret123") {
		t.Fatal("ovpn body missing credentials")
	}
}

func TestAdmin_patchManager(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	t.Run("username_and_password", func(t *testing.T) {
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"username": "ali2",
			"password": "ResetPass1",
		}, adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("patch: %d %s", w.Code, w.Body.String())
		}
		testutil.LoginToken(t, h, "ali2", "ResetPass1")
		w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
			"username": "ali", "password": "ManagerPass1",
		}, "")
		if w.Code == http.StatusOK {
			t.Fatal("old username should not login")
		}
	})

	t.Run("password_only", func(t *testing.T) {
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"password": "OnlyPass1",
		}, adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("patch: %d %s", w.Code, w.Body.String())
		}
		testutil.LoginToken(t, h, "bob", "OnlyPass1")
	})

	t.Run("username_unchanged_same_id", func(t *testing.T) {
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "carol", "carol", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"username": "carol",
		}, adminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("patch same username: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("username_duplicate", func(t *testing.T) {
		_, _ = testutil.SeedManager(t, h, adminToken, "taken", "taken", 10)
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "other", "other", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"username": "taken",
		}, adminToken)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409: %d %s", w.Code, w.Body.String())
		}
		var body struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		testutil.DecodeJSON(t, w, &body)
		if body.Error.Code != "USERNAME_IN_USE" {
			t.Fatalf("code: %s", body.Error.Code)
		}
	})

	t.Run("username_case_insensitive_duplicate", func(t *testing.T) {
		_, _ = testutil.SeedManager(t, h, adminToken, "CaseUser", "caseuser", 10)
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "caseother", "caseother", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"username": "caseuser",
		}, adminToken)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("username_empty", func(t *testing.T) {
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "empty", "empty", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"username": "   ",
		}, adminToken)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("password_invalid", func(t *testing.T) {
		mgrID, _ := testutil.SeedManager(t, h, adminToken, "weak", "weak", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"password": "short",
		}, adminToken)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400: %d %s", w.Code, w.Body.String())
		}
		testutil.LoginToken(t, h, "weak", "ManagerPass1")
	})

	t.Run("forbidden_for_manager", func(t *testing.T) {
		mgrID, mgrToken := testutil.SeedManager(t, h, adminToken, "guard", "guard", 10)
		w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
			"password": "HackPass1",
		}, mgrToken)
		if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
			t.Fatalf("manager patch: %d %s", w.Code, w.Body.String())
		}
	})
}

func TestPOST_me_password(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/me/password", map[string]string{
		"current_password": "ManagerPass1", "new_password": "NewPass123",
	}, mgrToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("change password: %d %s", w.Code, w.Body.String())
	}
	testutil.LoginToken(t, h, "ali", "NewPass123")

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/me/password", map[string]string{
		"current_password": "AdminPass1", "new_password": "NewAdmin123",
	}, adminToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("admin change password: %d %s", w.Code, w.Body.String())
	}
	testutil.LoginToken(t, h, "admin", "NewAdmin123")
}

func TestPOST_me_password_wrongCurrent(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	for _, tc := range []struct {
		name   string
		token  string
		user   string
		pass   string
		expect string
	}{
		{"admin", adminToken, "admin", "AdminPass1", "NewAdmin123"},
		{"manager", mgrToken, "ali", "ManagerPass1", "NewPass123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/me/password", map[string]string{
				"current_password": "wrong-password",
				"new_password":     tc.expect,
			}, tc.token)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			testutil.DecodeJSON(t, w, &body)
			if body.Error.Code != "INVALID_CURRENT_PASSWORD" {
				t.Fatalf("code=%q", body.Error.Code)
			}
			testutil.LoginToken(t, h, tc.user, tc.pass)
		})
	}
}

func TestPOST_me_password_invalidNew(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/me/password", map[string]string{
		"current_password": "AdminPass1", "new_password": "ab",
	}, adminToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	testutil.LoginToken(t, h, "admin", "AdminPass1")
}

func TestRemoveInactiveProfileReserve(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	_, mgrToken := testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "res", "password": "Secret123", "shared_users": 1, "assign_profile": true,
	}, mgrToken)
	var created map[string]any
	testutil.DecodeJSON(t, w, &created)
	id := int(created["id"].(float64))
	// second assign creates expired reserve
	testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users/"+strconv.Itoa(id)+"/assign-profile", map[string]any{
		"profile_name": "profile-open-2M-30d",
	}, mgrToken)
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.Itoa(id), nil, mgrToken)
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	profs := detail["profiles"].([]any)
	var reserveID string
	for _, p := range profs {
		row := p.(map[string]any)
		if row["state"] == "expired" {
			reserveID = row["id"].(string)
			break
		}
	}
	if reserveID == "" {
		t.Fatal("no expired reserve profile")
	}
	w = testutil.DoJSON(t, h, http.MethodDelete, "/api/v1/vpn-users/"+strconv.Itoa(id)+"/profiles/"+reserveID, nil, mgrToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("remove reserve: %d %s", w.Code, w.Body.String())
	}
}

func TestAuth_refreshToken(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": "admin", "password": "AdminPass1",
	}, "")
	var login map[string]any
	testutil.DecodeJSON(t, w, &login)
	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": login["refresh_token"].(string),
	}, "")
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
}

func itemNames(items []any) []string {
	var out []string
	for _, it := range items {
		row := it.(map[string]any)
		out = append(out, row["mikrotik_name"].(string))
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
