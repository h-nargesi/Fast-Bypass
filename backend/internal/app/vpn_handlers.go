package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/owner"
	"fast-bypass/internal/password"
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
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	users, err := a.listUsers(ctx, r.URL.Query().Get("refresh") == "true")
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	activeOnly := r.URL.Query().Get("active_only") == "true"
	now := a.Now()
	var items []map[string]any
	for _, u := range users {
		if reg.Resolve(u.Name, u.Comment) != c.ManagerID {
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
		item := map[string]any{"mikrotik_name": u.Name, "shared_users": u.SharedUsers, "profiles": profileDTOs(profs)}
		if meta, err := a.Store.FindVPNMetaByName(ctx, u.Name); err == nil {
			item["id"] = meta.ID
			item["local_name"] = meta.LocalName
		}
		items = append(items, item)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
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
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	out, err := a.createVPNUser(r.Context(), mgr, req)
	if errors.Is(err, errQuotaExceeded) {
		httpx.WriteError(w, http.StatusConflict, "QUOTA_EXCEEDED", "سقف اتصال همزمان پر است")
		return
	}
	if errors.Is(err, errNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", "نام کاربر در روتر وجود دارد")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

type patchVPNReq struct {
	Password     *string `json:"password"`
	SharedUsers  *int    `json:"shared_users"`
	ContactPhone *string `json:"contact_phone"`
	ContactNote  *string `json:"contact_note"`
	Notes        *string `json:"notes"`
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
	mgr, _ := a.Store.FindManagerByID(ctx, c.ManagerID)
	comment := reg.PanelComment(owner.ManagerInfo{ID: mgr.ID, Slug: mgr.Slug})
	if req.Password != nil {
		if !password.ValidVPN(*req.Password) {
			httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "رمز VPN نامعتبر")
			return
		}
	}
	if req.SharedUsers != nil {
		used, _ := a.managerUsedQuota(ctx, c.ManagerID)
		profs, _ := a.MT.ListUserProfiles(u.Name)
		active := false
		for _, p := range profs {
			if quota.ProfileActive(p, a.Now()) {
				active = true
				break
			}
		}
		if active && !quota.CheckIncrease(used, mgr.Quota, u.SharedUsers, *req.SharedUsers) {
			httpx.WriteError(w, http.StatusConflict, "QUOTA_EXCEEDED", "سقف اتصال همزمان پر است")
			return
		}
	}
	if err := a.MT.SetUser(u.Name, req.Password, req.SharedUsers, comment); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	_ = a.Store.UpdateVPNMeta(ctx, meta.ID, req.ContactPhone, req.ContactNote, req.Notes, nil)
	out, _ := a.buildVPNDetail(ctx, reg, meta, false)
	httpx.WriteJSON(w, http.StatusOK, out)
}
