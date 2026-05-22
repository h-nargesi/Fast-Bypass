package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/owner"
	"fast-bypass/internal/store"
)

func vpnMikrotikName(r *http.Request) string {
	return chi.URLParam(r, "name")
}

// buildVPNDetailByName returns router data; panel meta fields empty until a row exists.
func (a *App) buildVPNDetailByName(ctx context.Context, reg owner.Registry, name string, includeComment bool) (map[string]any, error) {
	meta, err := a.Store.FindVPNMetaByName(ctx, name)
	if err == nil {
		return a.buildVPNDetail(ctx, reg, meta, includeComment)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	u, err := a.MT.GetUser(name)
	if err != nil {
		return nil, err
	}
	profs, err := a.MT.ListUserProfiles(name)
	if err != nil {
		return nil, err
	}
	mid, dn, un, sl, mismatch := a.enrichOwner(ctx, reg, u.Name, u.Comment)
	out := map[string]any{
		"id":                   nil,
		"mikrotik_name":        u.Name,
		"shared_users":         u.SharedUsers,
		"disabled":             u.Disabled,
		"contact_info":         nil,
		"notes":                nil,
		"profiles":             profileDTOs(profs),
		"activations":          []map[string]any{},
		"connection_bundle":    a.connectionBundle(u),
		"manager_id":           mid,
		"manager_display_name": dn,
		"manager_username":     un,
		"manager_slug":         sl,
		"owner_mismatch":       mismatch,
	}
	if includeComment {
		out["mikrotik_comment"] = u.Comment
	}
	return out, nil
}

func (a *App) ensureVPNMeta(ctx context.Context, reg owner.Registry, name string, mgr *auth.Claims) (*store.VPNUserMeta, error) {
	meta, err := a.Store.FindVPNMetaByName(ctx, name)
	if err == nil {
		return meta, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	u, err := a.MT.GetUser(name)
	if err != nil {
		return nil, err
	}
	if mgr != nil {
		if err := a.assertManagerOwner(reg, mgr, u.Name, u.Comment); err != nil {
			return nil, err
		}
	}
	ownerID := reg.Resolve(u.Name, u.Comment)
	meta = &store.VPNUserMeta{MikrotikName: name}
	if ownerID != 0 {
		meta.ManagerID = sql.NullInt64{Int64: ownerID, Valid: true}
	}
	if err := a.Store.CreateVPNMeta(ctx, meta); err != nil {
		if again, findErr := a.Store.FindVPNMetaByName(ctx, name); findErr == nil {
			return again, nil
		}
		return nil, err
	}
	return meta, nil
}

func (a *App) deleteVPNByName(ctx context.Context, name string) error {
	meta, err := a.Store.FindVPNMetaByName(ctx, name)
	if err == nil {
		return a.deleteVPNUser(ctx, meta)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return a.MT.RemoveUser(name)
	}
	return err
}

func (a *App) HandleGetVPNUserByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.assertManagerOwner(reg, c, u.Name, u.Comment); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	out, err := a.buildVPNDetailByName(ctx, reg, name, false)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandlePatchVPNUserByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	meta, err := a.ensureVPNMeta(ctx, reg, name, c)
	if err != nil {
		if errors.Is(err, errNotOwner) {
			httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
			return
		}
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	var req patchVPNReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	out, err := a.patchVPNUser(ctx, reg, meta, req, nil, false)
	if err != nil {
		a.writeVPNPatchError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleAssignProfileByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	meta, err := a.ensureVPNMeta(ctx, reg, name, c)
	if err != nil {
		if errors.Is(err, errNotOwner) {
			httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
			return
		}
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
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
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleDeleteVPNUserByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.assertManagerOwner(reg, c, u.Name, u.Comment); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	if err := a.deleteVPNByName(ctx, name); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleConnectionBundleByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	reg, _ := a.Registry(r.Context())
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.assertManagerOwner(reg, c, u.Name, u.Comment); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.connectionBundle(u))
}

func (a *App) HandleDownloadOvpnByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	reg, _ := a.Registry(r.Context())
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.assertManagerOwner(reg, c, u.Name, u.Comment); err != nil {
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

func (a *App) HandleRemoveProfileByName(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	name := vpnMikrotikName(r)
	profileRowID := chi.URLParam(r, "profileRowId")
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.assertManagerOwner(reg, c, u.Name, u.Comment); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	meta, err := a.ensureVPNMeta(ctx, reg, name, c)
	if err != nil {
		if errors.Is(err, errNotOwner) {
			httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	if err := a.removeVPNProfile(meta, profileRowID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "حذف پروفایل ممکن نیست")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleAdminGetVPNUserByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	if _, err := a.MT.GetUser(name); err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	out, err := a.buildVPNDetailByName(ctx, reg, name, true)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleAdminPatchVPNUserByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	meta, err := a.ensureVPNMeta(ctx, reg, name, nil)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	var req adminPatchVPNReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	out, err := a.patchVPNUser(ctx, reg, meta, req.patchVPNReq, req.ManagerID, true)
	if err != nil {
		a.writeVPNPatchError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleAdminAssignProfileByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	meta, err := a.ensureVPNMeta(ctx, reg, name, nil)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	var req assignProfileReq
	if err := httpx.DecodeJSON(r, &req); err != nil || req.ProfileName == "" {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "profile_name لازم است")
		return
	}
	out, err := a.assignVPNProfile(ctx, reg, meta, req, true)
	if err != nil {
		if err == mikrotik.ErrProfileMissing {
			httpx.WriteError(w, http.StatusBadRequest, "PROFILE_NOT_FOUND", "پروفایل در روتر تعریف نشده")
			return
		}
		if errors.Is(err, errOrphanNoOwner) {
			httpx.WriteError(w, http.StatusForbidden, "ORPHAN_NO_OWNER", "کاربر بدون مدیر — ابتدا مالک را در روتر مشخص کنید")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleAdminDeleteVPNUserByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	if _, err := a.MT.GetUser(name); err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.deleteVPNByName(r.Context(), name); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleAdminConnectionBundleByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.connectionBundle(u))
}

func (a *App) HandleAdminDownloadOvpnByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	u, err := a.MT.GetUser(name)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
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

func (a *App) HandleAdminRemoveProfileByName(w http.ResponseWriter, r *http.Request) {
	name := vpnMikrotikName(r)
	profileRowID := chi.URLParam(r, "profileRowId")
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	meta, err := a.ensureVPNMeta(ctx, reg, name, nil)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.removeVPNProfile(meta, profileRowID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "حذف پروفایل ممکن نیست")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
