package store

import (
	"path/filepath"
	"testing"
)

func TestMigrate_certificateColumns(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "cert-migrate.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	for _, spec := range []struct{ table, col string }{
		{"managers", "cert_title"},
		{"managers", "cert_key_pass"},
		{"vpn_user_meta", "cert_title"},
		{"vpn_user_meta", "cert_key_pass"},
	} {
		has, err := st.tableHasColumn(spec.table, spec.col)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatalf("missing %s.%s after migrate", spec.table, spec.col)
		}
	}
}
