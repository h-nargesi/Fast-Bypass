package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/quota"
	"fast-bypass/internal/store"
)

func (a *App) HandleListVPNUsers(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	ctx := r.Context()
	reg, err := a.Registry(ctx)
	if err != nil {
		a.Log.Error("faild to registry", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	users, err := a.listUsers(ctx, r.URL.Query().Get("refresh") == "true")
	if err != nil {
		a.Log.Error("faild to list users", "error", err)
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	activeOnly := r.URL.Query().Get("active_only") == "true"
	qSearch := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	now := a.Now()

	type vpnRow struct {
		name     string
		shared   int
		disabled bool
		profs    []mikrotik.UserProfile
	}
	var filtered []vpnRow
	var names []string
	for _, u := range users {
		if reg.Resolve(u.Name, u.Comment) != c.ManagerID {
			continue
		}
		if activeOnly && u.Disabled {
			continue
		}
		if qSearch != "" && !strings.Contains(strings.ToLower(u.Name), qSearch) {
			continue
		}
		profs, _ := a.MT.ListUserProfiles(u.Name)
		if activeOnly {
			okActive := false
			for _, p := range profs {
				if quota.ProfileActive(p, now) {
					okActive = true
					break
				}
			}
			if !okActive {
				continue
			}
		}
		filtered = append(filtered, vpnRow{name: u.Name, shared: u.SharedUsers, disabled: u.Disabled, profs: profs})
		names = append(names, u.Name)
	}

	metaMap := a.Store.FindVPNMetasByNames(ctx, names)
	total := len(filtered)
	page, pageSize := parsePage(r), parsePageSize(r)
	start, end := pageWindow(total, page, pageSize)

	var items []map[string]any
	for _, f := range filtered[start:end] {
		item := map[string]any{
			"mikrotik_name": f.name, "shared_users": f.shared,
			"disabled": f.disabled, "profiles": profileDTOs(f.profs),
		}
		if meta, ok := metaMap[f.name]; ok {
			item["id"] = meta.ID
		}
		items = append(items, item)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items, "page": page, "page_size": pageSize, "total": total,
	})
}

func (a *App) vpnMetaByID(w http.ResponseWriter, r *http.Request) (*store.VPNUserMeta, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "شناسه نامعتبر")
		return nil, false
	}
	meta, err := a.Store.FindVPNMetaByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر یافت نشد")
		return nil, false
	}
	if err != nil {
		a.Log.Error("faild to find vpn meta by id", "error", err, "id", id)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return nil, false
	}
	return meta, true
}

func (a *App) HandleGetVPNUser(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	if err := a.assertManagerOwner(reg, c, u.Name, u.Comment); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_OWNER", "کاربر متعلق به شما نیست")
		return
	}
	out, err := a.buildVPNDetail(ctx, reg, meta, false)
	if err != nil {
		a.Log.Error("faild to build vpn detail", "error", err, "name", meta.MikrotikName)
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (a *App) HandleCreateVPNUser(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	var req createVPNReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	mgr, err := a.Store.FindManagerByID(r.Context(), c.ManagerID)
	if err != nil {
		a.Log.Error("faild to find manager by id", "error", err, "manager_id", c.ManagerID)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	out, err := a.createVPNUser(r.Context(), mgr, req)
	if errors.Is(err, errQuotaExceeded) {
		httpx.WriteError(w, http.StatusConflict, "QUOTA_EXCEEDED", "سقف تعداد کاربران (اتصال همزمان) پر است")
		return
	}
	if errors.Is(err, errNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", "نام کاربر در روتر وجود دارد")
		return
	}
	if err != nil {
		a.Log.Error("faild to create vpn user", "error", err, "request", req)
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (a *App) HandlePatchVPNUser(w http.ResponseWriter, r *http.Request) {
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
	var req patchVPNReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	out, err := a.patchVPNUser(ctx, reg, meta, req, nil, false)
	if err != nil {
		a.writeVPNPatchError(w, err, "HandlePatchVPNUser")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
