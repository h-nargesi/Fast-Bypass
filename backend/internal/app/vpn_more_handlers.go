package app

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/mikrotik"
)

type assignProfileReq struct {
	ProfileName string   `json:"profile_name"`
	AmountPaid  *float64 `json:"amount_paid"`
	Currency    *string  `json:"currency"`
	PaidAt      *string  `json:"paid_at"`
	Note        *string  `json:"note"`
}

func (a *App) HandleAssignProfile(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	reg, _ := a.Registry(ctx)
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil || a.assertManagerOwner(reg, c, u.Name, u.Comment) != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	var req assignProfileReq
	if err := httpx.DecodeJSON(r, &req); err != nil || req.ProfileName == "" {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "profile_name لازم است")
		return
	}
	out, err := a.assignVPNProfile(ctx, reg, meta, req, false)
	if err != nil {
		if err == mikrotik.ErrProfileMissing {
			httpx.WriteError(w, http.StatusBadRequest, "PROFILE_NOT_FOUND", "پروفایل در روتر تعریف نشده")
			return
		}
		if errors.Is(err, errOrphanNoOwner) {
			httpx.WriteError(w, http.StatusForbidden, "ORPHAN_NO_OWNER", "کاربر بدون مدیر — ابتدا مالک را در روتر مشخص کنید")
			return
		}
		a.Log.Error("faild to assign vpn profile", "error", err, "request", req)
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleDeleteVPNUser(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	reg, _ := a.Registry(ctx)
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil || a.assertManagerOwner(reg, c, u.Name, u.Comment) != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	if err := a.deleteVPNUser(ctx, meta); err != nil {
		a.Log.Error("faild to delete vpn user", "error", err, "meta", meta)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleConnectionBundle(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	reg, _ := a.Registry(r.Context())
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil || a.assertManagerOwner(reg, c, u.Name, u.Comment) != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	bundle, err := a.connectionBundleFor(r.Context(), meta, u)
	if err != nil {
		a.Log.Error("faild to connection bundle", "error", err, "username", meta.MikrotikName)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bundle)
}

func (a *App) HandleDownloadOvpn(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	reg, _ := a.Registry(r.Context())
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil || a.assertManagerOwner(reg, c, u.Name, u.Comment) != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	a.writeOvpnDownload(w, r, meta)
}

func (a *App) HandleRemoveProfile(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	profileRowID := chi.URLParam(r, "profileRowId")
	ctx := r.Context()
	reg, _ := a.Registry(ctx)
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil || a.assertManagerOwner(reg, c, u.Name, u.Comment) != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	if err := a.removeVPNProfile(meta, profileRowID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "حذف پروفایل ممکن نیست")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
