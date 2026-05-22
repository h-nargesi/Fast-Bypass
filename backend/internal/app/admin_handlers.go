package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"fast-bypass/internal/httpx"
	"fast-bypass/internal/owner"
	"fast-bypass/internal/password"
	"fast-bypass/internal/store"
)

type createManagerReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
	Quota       int    `json:"quota"`
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
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
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
		mid, dn, un, sl, mismatch := a.enrichOwner(ctx, reg, u.Name, u.Comment)
		profs, _ := a.MT.ListUserProfiles(u.Name)
		item := map[string]any{
			"mikrotik_name": u.Name, "shared_users": u.SharedUsers,
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
	DisplayName *string `json:"display_name"`
	Quota       *int    `json:"quota"`
	IsActive    *bool   `json:"is_active"`
	Password    *string `json:"password"`
}

func (a *App) HandlePatchManager(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req patchManagerReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	ctx := r.Context()
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
	if req.Password != nil && password.ValidPanel(*req.Password) {
		h, _ := password.Hash(*req.Password)
		hash = &h
	}
	if err := a.Store.UpdateManager(ctx, id, req.DisplayName, req.Quota, req.IsActive, hash); err != nil {
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
	})
}
