package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"fast-bypass/internal/password"
)

func openTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	return st, func() { _ = st.Close() }
}

func TestMigrate_andAdminBootstrap(t *testing.T) {
	st, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()
	n, err := st.AdminCount(ctx)
	if err != nil || n != 0 {
		t.Fatalf("AdminCount = %d err=%v", n, err)
	}
	hash, _ := password.Hash("AdminPass1")
	if err := st.CreateAdmin(ctx, "admin", hash); err != nil {
		t.Fatal(err)
	}
	adm, err := st.FindAdminByUsername(ctx, "admin")
	if err != nil || !password.Check(adm.PasswordHash, "AdminPass1") {
		t.Fatalf("admin: %+v err=%v", adm, err)
	}
}

func TestManagerCRUD_andSlug(t *testing.T) {
	st, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()
	hash, _ := password.Hash("Manager1")
	m := &Manager{Username: "ali", PasswordHash: hash, DisplayName: "علی", Slug: "ali", Quota: 10, IsActive: true}
	if err := st.CreateManager(ctx, m); err != nil {
		t.Fatal(err)
	}
	slugs, err := st.ListManagerSlugs(ctx)
	if err != nil || len(slugs) != 1 || slugs[0].Slug != "ali" {
		t.Fatalf("slugs: %+v err=%v", slugs, err)
	}
	found, err := st.FindManagerByUsername(ctx, "ali")
	if err != nil || found.Quota != 10 {
		t.Fatal(err)
	}
	dn := "علی احمدی"
	if err := st.UpdateManager(ctx, m.ID, &dn, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	found, _ = st.FindManagerByID(ctx, m.ID)
	if found.DisplayName != dn {
		t.Fatalf("display_name = %q", found.DisplayName)
	}
}

func TestVPNMeta_andActivations(t *testing.T) {
	st, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()
	hash, _ := password.Hash("Manager1")
	m := &Manager{Username: "ali", PasswordHash: hash, Slug: "ali", Quota: 10, IsActive: true}
	_ = st.CreateManager(ctx, m)
	meta := &VPNUserMeta{
		MikrotikName: "ali-u1",
		ManagerID: sql.NullInt64{Int64: m.ID, Valid: true},
	}
	if err := st.CreateVPNMeta(ctx, meta); err != nil {
		t.Fatal(err)
	}
	act := &ProfileActivation{
		VPNUserMetaID: meta.ID, ProfileName: "profile-open-2M-30d", SharedUsers: 2, Currency: "IRR",
	}
	if err := st.CreateActivation(ctx, act); err != nil {
		t.Fatal(err)
	}
	items, total, summary, err := st.ListRenewals(ctx, RenewalFilter{ManagerID: &m.ID})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("renewals: total=%d items=%d err=%v", total, len(items), err)
	}
	if summary.UnsettledSharedUsersSum != 2 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestSettleThrough_rejectsWrongScope(t *testing.T) {
	st, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()
	hash, _ := password.Hash("AdminPass1")
	_ = st.CreateAdmin(ctx, "admin", hash)
	m := &Manager{Username: "ali", PasswordHash: hash, Slug: "ali", Quota: 10, IsActive: true}
	_ = st.CreateManager(ctx, m)
	meta := &VPNUserMeta{
		MikrotikName: "ali-u1",
		ManagerID: sql.NullInt64{Int64: m.ID, Valid: true},
	}
	_ = st.CreateVPNMeta(ctx, meta)
	act := &ProfileActivation{VPNUserMetaID: meta.ID, ProfileName: "p1", SharedUsers: 1, Currency: "IRR"}
	_ = st.CreateActivation(ctx, act)
	_, err := st.SettleThrough(ctx, 1, RenewalThrough{
		ActivationID: act.ID, ManagerID: ptrInt64(999),
	})
	if err == nil {
		t.Fatal("expected scope error")
	}
}

func ptrInt64(v int64) *int64 { return &v }

func TestMigrate_legacyVPNMetaColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE vpn_user_meta (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mikrotik_name TEXT NOT NULL UNIQUE,
			manager_id INTEGER,
			local_name TEXT NOT NULL,
			contact_phone TEXT,
			contact_note TEXT,
			notes TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		INSERT INTO vpn_user_meta (mikrotik_name, manager_id, local_name, contact_phone, contact_note, notes)
		VALUES ('ali-u1', NULL, 'u1', '09121234567', 'tg @x', 'old note');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, cleanup := openTestStoreAt(t, path)
	defer cleanup()
	ctx := context.Background()
	meta, err := st.FindVPNMetaByName(ctx, "ali-u1")
	if err != nil {
		t.Fatal(err)
	}
	if !meta.ContactInfo.Valid || meta.ContactInfo.String != "tg @x" {
		t.Fatalf("contact_info: %+v", meta.ContactInfo)
	}
	if !meta.Notes.Valid || meta.Notes.String != "old note" {
		t.Fatalf("notes: %+v", meta.Notes)
	}
	if meta.ManagerID.Valid {
		t.Fatal("manager_id should stay null")
	}
	has, err := st.tableHasColumn("vpn_user_meta", "local_name")
	if err != nil || has {
		t.Fatalf("local_name column should be gone: has=%v err=%v", has, err)
	}
}

func openTestStoreAt(t *testing.T, path string) (*Store, func()) {
	t.Helper()
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return st, func() { _ = st.Close() }
}

func TestVPNMeta_contactInfo_andRenewalSearch(t *testing.T) {
	st, cleanup := openTestStore(t)
	defer cleanup()
	ctx := context.Background()
	meta := &VPNUserMeta{MikrotikName: "guest01"}
	if err := st.CreateVPNMeta(ctx, meta); err != nil {
		t.Fatal(err)
	}
	info := "telegram @guest"
	if err := st.UpdateVPNMeta(ctx, meta.ID, &info, nil, nil); err != nil {
		t.Fatal(err)
	}
	act := &ProfileActivation{
		VPNUserMetaID: meta.ID, ProfileName: "p1", SharedUsers: 1, Currency: "IRR",
	}
	if err := st.CreateActivation(ctx, act); err != nil {
		t.Fatal(err)
	}
	items, total, _, err := st.ListRenewals(ctx, RenewalFilter{
		OrphanOnly: true, Query: "telegram", Page: 1, PageSize: 20,
	})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("search contact_info: total=%d items=%d err=%v", total, len(items), err)
	}
	if items[0].MikrotikName != "guest01" {
		t.Fatalf("item: %+v", items[0])
	}
}
