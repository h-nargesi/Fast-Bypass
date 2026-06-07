package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/server"
	"fast-bypass/internal/testutil"
)

func apiErrorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v raw=%s", err, w.Body.String())
	}
	return body.Error.Code
}

func TestAdminCreateVPN_withCert(t *testing.T) {
	application, st, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name":   "certuser",
		"password":     "Secret123",
		"shared_users": 1,
		"cert_title":   "shared-cert",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &created)

	meta, err := st.FindVPNMetaByID(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.CertTitle.Valid || meta.CertTitle.String != "shared-cert" {
		t.Fatalf("cert_title: %+v", meta.CertTitle)
	}
	if !meta.CertKeyPass.Valid || len(meta.CertKeyPass.String) < 8 {
		t.Fatalf("cert_key_pass: %+v", meta.CertKeyPass)
	}

	body, err := fake.ReadFileContents("open-vpns/config-certuser.ovpn")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `setenv FRIENDLY_NAME "Sabalan certuser"`) {
		t.Fatalf("ovpn missing friendly name: %s", body)
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users/"+strconv.FormatInt(created.ID, 10), nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), meta.CertKeyPass.String) {
		t.Fatal("connection_bundle should include cert key password")
	}
}

func TestManagerCreateVPN_inheritsManagerCert(t *testing.T) {
	application, st, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/managers", map[string]any{
		"username": "m1", "password": "ManagerPass1", "display_name": "M1",
		"slug": "m1", "quota": 5, "cert_title": "mgr-cert",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("manager: %d %s", w.Code, w.Body.String())
	}
	mgrToken := testutil.LoginToken(t, h, "m1", "ManagerPass1")

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/vpn-users", map[string]any{
		"local_name": "u1", "password": "Secret123", "shared_users": 1,
	}, mgrToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("user: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &created)

	meta, err := st.FindVPNMetaByID(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if meta.CertTitle.Valid {
		t.Fatal("manager-created user should not store cert_title on meta")
	}

	mgr, _ := st.FindManagerByUsername(context.Background(), "m1")
	if !mgr.CertKeyPass.Valid {
		t.Fatal("manager missing cert_key_pass")
	}

	body, err := fake.ReadFileContents("open-vpns/config-m1-u1.ovpn")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `setenv FRIENDLY_NAME "Sabalan m1-u1"`) {
		t.Fatalf("ovpn: %s", body)
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/vpn-users/"+strconv.FormatInt(created.ID, 10), nil, mgrToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), mgr.CertKeyPass.String) {
		t.Fatal("bundle should use manager cert password")
	}
}

func TestAdmin_patchManager_certTitle(t *testing.T) {
	application, st, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mgrID, _ := testutil.SeedManager(t, h, adminToken, "certmgr", "certmgr", 5)

	w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
		"cert_title": "mgr-new-cert",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}

	mgr, err := st.FindManagerByID(context.Background(), mgrID)
	if err != nil {
		t.Fatal(err)
	}
	if !mgr.CertTitle.Valid || mgr.CertTitle.String != "mgr-new-cert" {
		t.Fatalf("cert_title: %+v", mgr.CertTitle)
	}
	if _, err := fake.ReadFileContents("open-vpns/config-mgr-new-cert.ovpn"); err != nil {
		t.Fatal("template ovpn should exist after cert provision")
	}
}

func TestAdmin_patchVPNUser_certTitle(t *testing.T) {
	application, st, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "certu", "password": "Secret123", "shared_users": 1,
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d", w.Code)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &created)

	path := "/api/v1/admin/vpn-users/" + strconv.FormatInt(created.ID, 10)
	w = testutil.DoJSON(t, h, http.MethodPatch, path, map[string]any{
		"cert_title": "user-cert-1",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("set cert: %d %s", w.Code, w.Body.String())
	}
	meta, _ := st.FindVPNMetaByID(context.Background(), created.ID)
	if !meta.CertKeyPass.Valid {
		t.Fatal("expected cert_key_pass")
	}

	w = testutil.DoJSON(t, h, http.MethodPatch, path, map[string]any{
		"cert_title": "user-cert-2",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("change cert title: %d %s", w.Code, w.Body.String())
	}
	if _, err := fake.ReadFileContents("open-vpns/config-certu.ovpn"); err != nil {
		t.Fatal(err)
	}

	w = testutil.DoJSON(t, h, http.MethodPatch, path, map[string]any{
		"cert_title": "",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("clear cert: %d", w.Code)
	}
	meta, _ = st.FindVPNMetaByID(context.Background(), created.ID)
	if meta.CertTitle.Valid {
		t.Fatal("cert_title should be cleared")
	}
}

func TestAdmin_createVPN_sharedCertTitle_twoUsers(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	for _, name := range []string{"shared-a", "shared-b"} {
		w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
			"local_name": name, "password": "Secret123", "shared_users": 1,
			"cert_title": "pool-cert",
		}, adminToken)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: %d %s", name, w.Code, w.Body.String())
		}
	}

	for _, name := range []string{"shared-a", "shared-b"} {
		body, err := fake.ReadFileContents("open-vpns/config-" + name + ".ovpn")
		if err != nil {
			t.Fatalf("ovpn file %s: %v", name, err)
		}
		if !strings.Contains(string(body), "Sabalan "+name) {
			t.Fatalf("friendly name for %s: %s", name, body)
		}
	}
}

func TestAdmin_downloadOvpn_certFile(t *testing.T) {
	application, _, fake := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "ovpnu", "password": "Secret123", "shared_users": 1,
		"cert_title": "dl-cert",
	}, adminToken)
	var created struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &created)

	marker := "unique-ovpn-marker-dl-cert"
	if err := fake.WriteFileContents("open-vpns/config-ovpnu.ovpn", []byte("client\ndev tun\n"+marker+"\n")); err != nil {
		t.Fatal(err)
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users/"+strconv.FormatInt(created.ID, 10)+"/ovpn", nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("download: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), marker) {
		t.Fatal("expected cert ovpn body from router file")
	}
	if strings.Contains(w.Body.String(), "{{username}}") {
		t.Fatal("should not use legacy template")
	}
}

func TestConnectionBundle_certPriority(t *testing.T) {
	application, st, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/managers", map[string]any{
		"username": "prio", "password": "ManagerPass1", "display_name": "P",
		"slug": "prio", "quota": 10, "cert_title": "mgr-only",
	}, adminToken)
	var mgr struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &mgr)
	mgrRow, err := st.FindManagerByID(context.Background(), mgr.ID)
	if err != nil || !mgrRow.CertKeyPass.Valid {
		t.Fatal("manager cert_key_pass")
	}
	mgrPass := mgrRow.CertKeyPass.String

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "uown", "password": "Secret123", "shared_users": 1,
		"manager_id": mgr.ID, "cert_title": "user-only",
	}, adminToken)
	var uOwn struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &uOwn)
	meta, err := st.FindVPNMetaByID(context.Background(), uOwn.ID)
	if err != nil || !meta.CertKeyPass.Valid {
		t.Fatal("user cert_key_pass")
	}

	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users/"+strconv.FormatInt(uOwn.ID, 10), nil, adminToken)
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	bundle := detail["connection_bundle"].(map[string]any)
	if bundle["openvpn_key_password"] != meta.CertKeyPass.String {
		t.Fatalf("expected user cert pass, got %v want %s", bundle["openvpn_key_password"], meta.CertKeyPass.String)
	}
	if bundle["openvpn_key_password"] == mgrPass {
		t.Fatal("user cert should differ from manager cert")
	}

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "uinherit", "password": "Secret123", "shared_users": 1,
		"manager_id": mgr.ID,
	}, adminToken)
	var uInh struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &uInh)
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users/"+strconv.FormatInt(uInh.ID, 10), nil, adminToken)
	testutil.DecodeJSON(t, w, &detail)
	bundle = detail["connection_bundle"].(map[string]any)
	if bundle["openvpn_key_password"] != mgrPass {
		t.Fatalf("expected manager cert pass, got %v want %s", bundle["openvpn_key_password"], mgrPass)
	}

	w = testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "orph", "password": "Secret123", "shared_users": 1,
	}, adminToken)
	var uOrph struct {
		ID int64 `json:"id"`
	}
	testutil.DecodeJSON(t, w, &uOrph)
	w = testutil.DoJSON(t, h, http.MethodGet, "/api/v1/admin/vpn-users/"+strconv.FormatInt(uOrph.ID, 10), nil, adminToken)
	testutil.DecodeJSON(t, w, &detail)
	bundle = detail["connection_bundle"].(map[string]any)
	if bundle["openvpn_key_password"] != "EnvKeyPass123" {
		t.Fatalf("expected env key pass, got %v", bundle["openvpn_key_password"])
	}
}

func TestAdmin_createVPN_invalidCertTitle(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "bad", "password": "Secret123", "shared_users": 1,
		"cert_title": "ab",
	}, adminToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", w.Code)
	}
	if code := apiErrorCode(t, w); code != "VALIDATION" {
		t.Fatalf("code: %s", code)
	}
}

func TestAdmin_patchManager_changeCertTitle(t *testing.T) {
	application, st, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")
	mgrID, _ := testutil.SeedManager(t, h, adminToken, "chgmgr", "chgmgr", 5)

	w := testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
		"cert_title": "first-mgr-cert",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("first patch: %d %s", w.Code, w.Body.String())
	}
	mgr, _ := st.FindManagerByID(context.Background(), mgrID)
	pass1 := mgr.CertKeyPass.String

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/managers/"+strconv.FormatInt(mgrID, 10), map[string]any{
		"cert_title": "second-mgr-cert",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("second patch: %d %s", w.Code, w.Body.String())
	}
	mgr, _ = st.FindManagerByID(context.Background(), mgrID)
	if mgr.CertTitle.String != "second-mgr-cert" {
		t.Fatalf("title: %s", mgr.CertTitle.String)
	}
	if mgr.CertKeyPass.String == pass1 {
		t.Fatal("expected new passphrase after title change")
	}
}

func TestAdmin_patchVPNUser_certTitle_byName(t *testing.T) {
	application, _, _ := testutil.NewTestApp(t)
	h := server.New(application)
	adminToken := testutil.LoginToken(t, h, "admin", "AdminPass1")

	w := testutil.DoJSON(t, h, http.MethodPost, "/api/v1/admin/vpn-users", map[string]any{
		"local_name": "bynameu", "password": "Secret123", "shared_users": 1,
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d", w.Code)
	}

	w = testutil.DoJSON(t, h, http.MethodPatch, "/api/v1/admin/vpn-users/by-name/bynameu", map[string]any{
		"cert_title": "byname-cert",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("patch by name: %d %s", w.Code, w.Body.String())
	}
	var detail map[string]any
	testutil.DecodeJSON(t, w, &detail)
	if detail["cert_title"] != "byname-cert" {
		t.Fatalf("cert_title: %v", detail["cert_title"])
	}
}

// compile-time check fake implements cert APIs
var _ interface {
	GenerateCertificate(string, string, string) error
	ReadFileContents(string) ([]byte, error)
} = (*mikrotik.FakeClient)(nil)
