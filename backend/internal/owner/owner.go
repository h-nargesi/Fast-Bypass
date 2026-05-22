package owner

import (
	"strings"

	"fast-bypass/internal/store"
)

type ManagerInfo struct {
	ID   int64
	Slug string
}

type Registry struct {
	Separator string
	Managers  []ManagerInfo
}

func (r Registry) Pattern(m ManagerInfo) string {
	return m.Slug + r.Separator
}

func (r Registry) PanelComment(m ManagerInfo) string {
	return "panel:" + m.Slug
}

// Resolve returns manager ID or 0 if orphan.
func (r Registry) Resolve(name, comment string) int64 {
	for _, m := range r.Managers {
		if strings.HasPrefix(name, r.Pattern(m)) {
			return m.ID
		}
	}
	for _, m := range r.Managers {
		if comment == r.PanelComment(m) {
			return m.ID
		}
	}
	return 0
}

func (r Registry) OwnerMismatch(name, comment string, resolvedID int64) bool {
	if resolvedID == 0 {
		return false
	}
	var nameID, commentID int64
	for _, m := range r.Managers {
		if strings.HasPrefix(name, r.Pattern(m)) {
			nameID = m.ID
			break
		}
	}
	for _, m := range r.Managers {
		if comment == r.PanelComment(m) {
			commentID = m.ID
			break
		}
	}
	if nameID == 0 || commentID == 0 {
		return false
	}
	return nameID != commentID
}

func SlugOverlaps(slug string, existing []store.ManagerSlug, excludeID int64) bool {
	for _, e := range existing {
		if excludeID > 0 && e.ID == excludeID {
			continue
		}
		if slug == e.Slug || strings.HasPrefix(slug, e.Slug) || strings.HasPrefix(e.Slug, slug) {
			return true
		}
	}
	return false
}

func BuildRegistry(managers []store.Manager, separator string) Registry {
	reg := Registry{Separator: separator}
	for _, m := range managers {
		reg.Managers = append(reg.Managers, ManagerInfo{ID: m.ID, Slug: m.Slug})
	}
	return reg
}
