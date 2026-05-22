package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/password"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	Role         string  `json:"role"`
	ManagerID    *int64  `json:"manager_id,omitempty"`
	Slug         *string `json:"slug,omitempty"`
	NamePrefix   *string `json:"name_prefix,omitempty"`
}

func (a *App) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	ctx := r.Context()

	if adm, err := a.Store.FindAdminByUsername(ctx, req.Username); err == nil {
		if !adm.IsActive || !password.Check(adm.PasswordHash, req.Password) {
			httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "نام کاربری یا رمز اشتباه است")
			return
		}
		pair, err := a.Auth.NewPair(auth.Claims{Role: auth.RoleAdmin, Username: adm.Username})
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, loginResp{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, Role: "admin"})
		return
	}

	mgr, err := a.Store.FindManagerByUsername(ctx, req.Username)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "نام کاربری یا رمز اشتباه است")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	if !mgr.IsActive {
		httpx.WriteError(w, http.StatusForbidden, "MANAGER_DISABLED", "حساب مدیر غیرفعال است")
		return
	}
	if !password.Check(mgr.PasswordHash, req.Password) {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "نام کاربری یا رمز اشتباه است")
		return
	}
	prefix := mgr.Slug + a.Cfg.UsernamePrefixSep
	pair, err := a.Auth.NewPair(auth.Claims{
		Role: auth.RoleManager, ManagerID: mgr.ID, Slug: mgr.Slug, Username: mgr.Username,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	mid := mgr.ID
	slug := mgr.Slug
	httpx.WriteJSON(w, http.StatusOK, loginResp{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, Role: "manager",
		ManagerID: &mid, Slug: &slug, NamePrefix: &prefix,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (a *App) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	c, err := a.Auth.Parse(req.RefreshToken)
	if err != nil || c.TokenType != "refresh" {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "توکن نامعتبر است")
		return
	}
	pair, err := a.Auth.NewPair(*c)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token": pair.AccessToken, "refresh_token": pair.RefreshToken,
	})
}

func (a *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleMe(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "توکن نامعتبر")
		return
	}
	ctx := r.Context()
	if c.Role == auth.RoleAdmin {
		adm, err := a.Store.FindAdminByUsername(ctx, c.Username)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"username": adm.Username, "role": "admin",
		})
		return
	}
	mgr, err := a.Store.FindManagerByID(ctx, c.ManagerID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	used, err := a.managerUsedQuota(ctx, mgr.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"username": mgr.Username, "display_name": mgr.DisplayName, "slug": mgr.Slug,
		"name_prefix": mgr.Slug + a.Cfg.UsernamePrefixSep, "quota": mgr.Quota, "used_quota": used,
	})
}

type patchMeReq struct {
	DisplayName *string `json:"display_name"`
}

func (a *App) HandlePatchMe(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "توکن نامعتبر")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	for k := range raw {
		if k != "display_name" {
			httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "فقط display_name مجاز است")
			return
		}
	}
	var req patchMeReq
	_ = json.Unmarshal(body, &req)
	ctx := r.Context()
	if c.Role == auth.RoleManager {
		if req.DisplayName == nil {
			httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "display_name لازم است")
			return
		}
		if err := a.Store.UpdateManager(ctx, c.ManagerID, req.DisplayName, nil, nil, nil); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
	}
	a.HandleMe(w, r)
}

type changePwReq struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (a *App) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "توکن نامعتبر")
		return
	}
	var req changePwReq
	if err := httpx.DecodeJSON(r, &req); err != nil || !password.ValidPanel(req.NewPassword) {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "رمز جدید نامعتبر است")
		return
	}
	ctx := r.Context()
	if c.Role == auth.RoleAdmin {
		adm, err := a.Store.FindAdminByUsername(ctx, c.Username)
		if err != nil || !password.Check(adm.PasswordHash, req.CurrentPassword) {
			httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CURRENT_PASSWORD", "رمز فعلی نادرست است")
			return
		}
		hash, err := password.Hash(req.NewPassword)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
		if err := a.Store.UpdateAdminPassword(ctx, adm.ID, hash); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
	} else {
		mgr, err := a.Store.FindManagerByID(ctx, c.ManagerID)
		if err != nil || !password.Check(mgr.PasswordHash, req.CurrentPassword) {
			httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CURRENT_PASSWORD", "رمز فعلی نادرست است")
			return
		}
		hash, err := password.Hash(req.NewPassword)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
		if err := a.Store.UpdateManager(ctx, c.ManagerID, nil, nil, nil, &hash); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) HandleMeQuota(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	mgr, err := a.Store.FindManagerByID(r.Context(), c.ManagerID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	used, err := a.managerUsedQuota(r.Context(), c.ManagerID)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{
		"quota": mgr.Quota, "used": used, "available": mgr.Quota - used,
	})
}
