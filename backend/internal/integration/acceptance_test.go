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

	_ = fake.AddUser("reza", "Secret123", "panel:ali", 1)
	_ = fake.AddUser("bob-only", "Secret123", "panel:bob", 1)

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
	if u.Comment != "panel:ali" {
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

	_ = fake.AddUser("guest99", "Secret123", "", 2)
	_ = fake.AddUser("ali-z", "Secret123", "panel:ali", 1)

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
			if row["manager_slug"] != "ali" || row["mikrotik_comment"] != "panel:ali" {
				t.Fatalf("owner fields: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("ali-z not in admin list")
	}
}

func TestAdmin_ownerMismatch_flag(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	testutil.SeedManager(t, h, adminToken, "ali", "ali", 10)
	testutil.SeedManager(t, h, adminToken, "bob", "bob", 10)

	_ = fake.AddUser("ali-conflict", "Secret123", "panel:bob", 1)

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
	_ = fake.AddUser("ali-extra", "Secret123", "panel:ali", 1)

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
