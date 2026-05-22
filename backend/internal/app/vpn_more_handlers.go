package app

import (
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
	if err := a.MT.AddUserProfile(u.Name, req.ProfileName); err != nil {
		if err == mikrotik.ErrProfileMissing {
			httpx.WriteError(w, http.StatusBadRequest, "PROFILE_NOT_FOUND", "پروفایل در روتر تعریف نشده")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	profs, _ := a.MT.ListUserProfiles(u.Name)
	_ = a.recordActivation(ctx, meta.ID, req.ProfileName, u.SharedUsers, req.AmountPaid, req.Currency, req.Note, profs)
	out, _ := a.buildVPNDetail(ctx, reg, meta, false)
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
	_ = a.MT.RemoveUser(meta.MikrotikName)
	_ = a.Store.DeleteVPNMeta(ctx, meta.ID)
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
	httpx.WriteJSON(w, http.StatusOK, a.connectionBundle(u))
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
	if u.Password == "" {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "رمز در دسترس نیست")
		return
	}
	body, err := a.renderOvpn(u.Name, u.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "TEMPLATE_MISSING", "قالب ovpn پیکربندی نشده")
		return
	}
	w.Header().Set("Content-Type", "application/x-openvpn-profile")
	w.Header().Set("Content-Disposition", `attachment; filename="`+u.Name+`.ovpn"`)
	_, _ = w.Write(body)
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
	if err := a.MT.RemoveUserProfile(profileRowID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "حذف پروفایل ممکن نیست")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
