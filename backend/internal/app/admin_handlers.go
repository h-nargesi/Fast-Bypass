package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"fast-bypass/internal/httpx"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/owner"
	"fast-bypass/internal/password"
	"fast-bypass/internal/quota"
	"fast-bypass/internal/store"
)

type createManagerReq struct {
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	DisplayName string  `json:"display_name"`
	Slug        string  `json:"slug"`
	Quota       int     `json:"quota"`
	CertTitle   *string `json:"cert_title"`
}

func (a *App) HandleListManagers(w http.ResponseWriter, r *http.Request) {
	list, err := a.Store.ListManagers(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	var items []map[string]any
	for _, m := range list {
		used, _ := a.managerUsedQuota(r.Context(), m.ID)
		items = append(items, map[string]any{
			"id": m.ID, "username": m.Username, "display_name": m.DisplayName,
			"slug": m.Slug, "quota": m.Quota, "used_quota": used, "is_active": m.IsActive,
			"cert_title": nullStrVal(m.CertTitle),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *App) HandleAdminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	users, err := a.listUsers(ctx, r.URL.Query().Get("refresh") == "true")
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	total, orphan, byOwner, err := quota.AggregateByOwner(reg, users, a.MT.ListUserProfiles, a.Now())
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	managers, err := a.Store.ListManagers(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	var byManager []map[string]any
	for _, m := range managers {
		s := byOwner[m.ID]
		byManager = append(byManager, map[string]any{
			"manager_id": m.ID, "display_name": m.DisplayName, "username": m.Username,
			"quota": m.Quota, "vpn_users": s.VPNUsers, "connections": s.Connections,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"manager_count": len(managers),
		"totals": map[string]any{
			"vpn_users": total.VPNUsers, "connections": total.Connections,
		},
		"orphan": map[string]any{
			"vpn_users": orphan.VPNUsers, "connections": orphan.Connections,
		},
		"by_manager": byManager,
	})
}

func (a *App) HandleCreateManager(w http.ResponseWriter, r *http.Request) {
	var req createManagerReq
	if err := httpx.DecodeJSON(r, &req); err != nil || !password.ValidPanel(req.Password) || req.Quota < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	ctx := r.Context()
	slugs, _ := a.Store.ListManagerSlugs(ctx)
	if owner.SlugOverlaps(req.Slug, slugs, 0) {
		httpx.WriteError(w, http.StatusConflict, "SLUG_OVERLAPS", "slug با مدیر دیگر همپوشانی دارد")
		return
	}
	hash, _ := password.Hash(req.Password)
	m := &store.Manager{
		Username: req.Username, PasswordHash: hash, DisplayName: req.DisplayName,
		Slug: req.Slug, Quota: req.Quota, IsActive: true,
	}
	if err := a.Store.CreateManager(ctx, m); err != nil {
		httpx.WriteError(w, http.StatusConflict, "SLUG_IN_USE", "نام کاربری یا slug تکراری است")
		return
	}
	if req.CertTitle != nil && strings.TrimSpace(*req.CertTitle) != "" {
		if err := a.setupManagerCertificate(ctx, m.ID, *req.CertTitle); err != nil {
			if errors.Is(err, errInvalidCertTitle) {
				httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "عنوان گواهی نامعتبر است")
				return
			}
			httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ساخت گواهی روی روتر ناموفق بود")
			return
		}
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": m.ID})
}

func (a *App) HandleAdminListVPNUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	users, err := a.listUsers(ctx, r.URL.Query().Get("refresh") == "true")
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	var filterMgr *int64
	if v := r.URL.Query().Get("manager_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		filterMgr = &id
	}
	orphanOnly := r.URL.Query().Get("orphan") == "true"
	var items []map[string]any
	for _, u := range users {
		ownerID := reg.Resolve(u.Name, u.Comment)
		if orphanOnly && ownerID != 0 {
			continue
		}
		if filterMgr != nil && ownerID != *filterMgr {
			continue
		}
		activeOnly := r.URL.Query().Get("active_only") == "true"
		if activeOnly && u.Disabled {
			continue
		}
		mid, dn, un, sl, mismatch := a.enrichOwner(ctx, reg, u.Name, u.Comment)
		profs, _ := a.MT.ListUserProfiles(u.Name)
		item := map[string]any{
			"mikrotik_name": u.Name, "shared_users": u.SharedUsers, "disabled": u.Disabled,
			"mikrotik_comment": u.Comment, "manager_id": mid,
			"manager_display_name": dn, "manager_username": un, "manager_slug": sl,
			"owner_mismatch": mismatch, "profiles": profileDTOs(profs),
		}
		if meta, err := a.Store.FindVPNMetaByName(ctx, u.Name); err == nil {
			item["id"] = meta.ID
		}
		items = append(items, item)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *App) HandleAdminGetVPNUser(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	reg, _ := a.Registry(r.Context())
	out, err := a.buildVPNDetail(r.Context(), reg, meta, true)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type patchManagerReq struct {
	Username    *string `json:"username"`
	DisplayName *string `json:"display_name"`
	Quota       *int    `json:"quota"`
	IsActive    *bool   `json:"is_active"`
	Password    *string `json:"password"`
	CertTitle   *string `json:"cert_title"`
}

func (a *App) HandlePatchManager(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req patchManagerReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	ctx := r.Context()
	cur, err := a.Store.FindManagerByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "مدیر یافت نشد")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	if req.CertTitle != nil {
		if err := a.applyManagerCertTitleChange(ctx, cur, *req.CertTitle); err != nil {
			if errors.Is(err, errInvalidCertTitle) {
				httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "عنوان گواهی نامعتبر است")
				return
			}
			httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ساخت گواهی روی روتر ناموفق بود")
			return
		}
	}
	var username *string
	if req.Username != nil {
		u := strings.TrimSpace(*req.Username)
		if u == "" {
			httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "نام کاربری نامعتبر است")
			return
		}
		existing, err := a.Store.FindManagerByUsername(ctx, u)
		if err == nil && existing.ID != id {
			httpx.WriteError(w, http.StatusConflict, "USERNAME_IN_USE", "نام کاربری تکراری است")
			return
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
		username = &u
	}
	if req.Quota != nil {
		used, err := a.managerUsedQuota(ctx, id)
		if err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر")
			return
		}
		if *req.Quota < used {
			httpx.WriteError(w, http.StatusConflict, "QUOTA_BELOW_USAGE", "quota کمتر از مصرف فعلی است")
			return
		}
	}
	var hash *string
	if req.Password != nil {
		pw := strings.TrimSpace(*req.Password)
		if pw != "" {
			if !password.ValidPanel(pw) {
				httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "رمز نامعتبر است (حداقل ۸ کاراکتر، حروف و عدد)")
				return
			}
			h, err := password.Hash(pw)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
				return
			}
			hash = &h
		}
	}
	if err := a.Store.UpdateManager(ctx, id, username, req.DisplayName, req.Quota, req.IsActive, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "مدیر یافت نشد")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	m, _ := a.Store.FindManagerByID(ctx, id)
	used, _ := a.managerUsedQuota(ctx, id)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"id": m.ID, "username": m.Username, "display_name": m.DisplayName,
		"slug": m.Slug, "quota": m.Quota, "used_quota": used, "is_active": m.IsActive,
		"cert_title": nullStrVal(m.CertTitle),
	})
}

type adminCreateVPNReq struct {
	ManagerID *int64 `json:"manager_id"`
	createVPNReq
}

func (a *App) HandleAdminCreateVPNUser(w http.ResponseWriter, r *http.Request) {
	var req adminCreateVPNReq
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.LocalName) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	ctx := r.Context()
	var out map[string]any
	var err error
	if req.ManagerID == nil || *req.ManagerID < 1 {
		out, err = a.createOrphanVPNUser(ctx, req.createVPNReq)
	} else {
		mgr, findErr := a.Store.FindManagerByID(ctx, *req.ManagerID)
		if errors.Is(findErr, sql.ErrNoRows) {
			httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "مدیر یافت نشد")
			return
		}
		if findErr != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
		out, err = a.createVPNUser(ctx, mgr, req.createVPNReq)
	}
	if errors.Is(err, errQuotaExceeded) {
		httpx.WriteError(w, http.StatusConflict, "QUOTA_EXCEEDED", "سقف تعداد کاربران (اتصال همزمان) پر است")
		return
	}
	if errors.Is(err, errNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", "نام کاربر در روتر وجود دارد")
		return
	}
	if errors.Is(err, errInvalidCertTitle) {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "عنوان گواهی نامعتبر است")
		return
	}
	if err != nil {
		if errors.Is(err, mikrotik.ErrUnavailable) || errors.Is(err, mikrotik.ErrNotFound) {
			httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر یا ساخت گواهی ناموفق بود")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (a *App) HandleAdminPatchVPNUser(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	var req adminPatchVPNReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	reg, _ := a.Registry(r.Context())
	out, err := a.adminPatchVPNUser(r.Context(), reg, meta, req)
	if err != nil {
		a.writeVPNPatchError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleAdminDeleteVPNUser(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	if err := a.deleteVPNUser(r.Context(), meta); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleAdminAssignProfile(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	var req assignProfileReq
	if err := httpx.DecodeJSON(r, &req); err != nil || req.ProfileName == "" {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "profile_name لازم است")
		return
	}
	reg, _ := a.Registry(r.Context())
	out, err := a.assignVPNProfile(r.Context(), reg, meta, req, true)
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

func (a *App) HandleAdminConnectionBundle(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	bundle, err := a.connectionBundleFor(r.Context(), meta, u)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bundle)
}

func (a *App) HandleAdminDownloadOvpn(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	a.writeOvpnDownload(w, r, meta)
}

func (a *App) HandleAdminRemoveProfile(w http.ResponseWriter, r *http.Request) {
	meta, ok := a.vpnMetaByID(w, r)
	if !ok {
		return
	}
	profileRowID := chi.URLParam(r, "profileRowId")
	if err := a.removeVPNProfile(meta, profileRowID); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "حذف پروفایل ممکن نیست")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
