package owner

import (
	"testing"

	"fast-bypass/internal/store"
)

func testRegistry() Registry {
	return Registry{
		Separator: "-",
		Managers: []ManagerInfo{
			{ID: 1, Slug: "ali"},
			{ID: 2, Slug: "bob"},
		},
	}
}

func TestResolve_byNamePrefix(t *testing.T) {
	reg := testRegistry()
	if got := reg.Resolve("ali-reza01", ""); got != 1 {
		t.Fatalf("Resolve() = %d, want 1", got)
	}
	if got := reg.Resolve("bob-user1", "panel:ali"); got != 2 {
		t.Fatalf("name prefix wins: got %d, want 2", got)
	}
}

func TestResolve_byCommentLegacy(t *testing.T) {
	reg := testRegistry()
	if got := reg.Resolve("reza", "panel:ali"); got != 1 {
		t.Fatalf("legacy comment: got %d, want 1", got)
	}
}

func TestResolve_orphan(t *testing.T) {
	reg := testRegistry()
	if got := reg.Resolve("guest01", ""); got != 0 {
		t.Fatalf("orphan: got %d, want 0", got)
	}
}

func TestOwnerMismatch(t *testing.T) {
	reg := testRegistry()
	if !reg.OwnerMismatch("ali-reza01", "panel:bob", 1) {
		t.Fatal("expected mismatch when name=ali and comment=panel:bob")
	}
	if reg.OwnerMismatch("ali-reza01", "panel:ali", 1) {
		t.Fatal("expected no mismatch when aligned")
	}
	if reg.OwnerMismatch("guest01", "", 0) {
		t.Fatal("orphan should not mismatch")
	}
}

func TestSlugOverlaps(t *testing.T) {
	existing := []store.ManagerSlug{
		{ID: 1, Slug: "ali"},
		{ID: 2, Slug: "bob"},
	}
	cases := []struct {
		slug string
		want bool
	}{
		{"alireza", true},
		{"alice", true},
		{"ali", true},
		{"reza", false},
		{"rezvan", false},
	}
	for _, tc := range cases {
		if got := SlugOverlaps(tc.slug, existing, 0); got != tc.want {
			t.Errorf("SlugOverlaps(%q) = %v, want %v", tc.slug, got, tc.want)
		}
	}
	if SlugOverlaps("ali", existing, 1) {
		t.Error("excludeID should skip self")
	}
}

func TestPanelComment(t *testing.T) {
	reg := testRegistry()
	if got := reg.PanelComment(ManagerInfo{Slug: "ali"}); got != "panel:ali" {
		t.Fatalf("PanelComment = %q", got)
	}
}
