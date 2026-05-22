package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

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
		MikrotikName: "ali-u1", LocalName: "u1",
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
		MikrotikName: "ali-u1", LocalName: "u1",
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
