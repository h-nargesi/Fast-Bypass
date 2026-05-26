package app

import (
	"net/http"
	"strconv"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/store"
)

func (a *App) HandleManagerRenewals(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleManager {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "فقط مدیر")
		return
	}
	mid := c.ManagerID
	a.writeRenewals(w, r, store.RenewalFilter{
		ManagerID: &mid, Settled: r.URL.Query().Get("settled"),
		Page: parsePage(r), PageSize: parsePageSize(r),
		Query: r.URL.Query().Get("q"),
	}, false)
}

func (a *App) HandleAdminRenewals(w http.ResponseWriter, r *http.Request) {
	filter := store.RenewalFilter{
		Settled: r.URL.Query().Get("settled"),
		Page:    parsePage(r), PageSize: parsePageSize(r),
		From:    r.URL.Query().Get("from"), To: r.URL.Query().Get("to"),
		Query:   r.URL.Query().Get("q"),
	}
	if v := r.URL.Query().Get("manager_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		filter.ManagerID = &id
	} else {
		filter.OrphanOnly = true
	}
	a.writeRenewals(w, r, filter, true)
}

func (a *App) writeRenewals(w http.ResponseWriter, r *http.Request, filter store.RenewalFilter, canSettle bool) {
	items, total, _, err := a.Store.ListRenewals(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	summary, err := a.applyRenewalsLiveSharedUsers(r.Context(), filter, items)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
		return
	}
	scope := map[string]any{"orphan": filter.OrphanOnly}
	if filter.ManagerID != nil {
		scope["manager_id"] = *filter.ManagerID
		if m, err := a.Store.FindManagerByID(r.Context(), *filter.ManagerID); err == nil {
			scope["manager_display_name"] = m.DisplayName
		}
	} else if filter.OrphanOnly {
		scope["manager_id"] = nil
	}
	var dto []map[string]any
	for _, it := range items {
		dto = append(dto, map[string]any{
			"id": it.ID, "renewed_at": it.RenewedAt, "mikrotik_name": it.MikrotikName,
			"shared_users": it.SharedUsers, "profile_name": it.ProfileName,
			"profile_state": it.ProfileState, "mikrotik_end_time": it.MikrotikEndTime,
			"is_settled": it.IsSettled, "amount_paid": it.AmountPaid, "currency": it.Currency,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"scope": scope, "can_settle": canSettle, "summary": summary,
		"items": dto, "page": filter.Page, "page_size": filter.PageSize, "total": total,
	})
}

func (a *App) HandleSettleThrough(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c.Role != auth.RoleAdmin {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "تسویه فقط برای ادمین")
		return
	}
	var body struct {
		ThroughActivationID int64  `json:"through_activation_id"`
		ManagerID           *int64 `json:"manager_id"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "ورودی نامعتبر")
		return
	}
	through := store.RenewalThrough{ActivationID: body.ThroughActivationID, ManagerID: body.ManagerID}
	if body.ManagerID == nil {
		through.OrphanOnly = true
	}
	adm, err := a.Store.FindAdminByUsername(r.Context(), c.Username)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	n, err := a.Store.SettleThrough(r.Context(), adm.ID, through)
	if err != nil {
		httpx.WriteError(w, http.StatusForbidden, "NOT_IN_SCOPE", "خارج از محدوده فیلتر")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated_count": n})
}

func parsePage(r *http.Request) int {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if p < 1 {
		return 1
	}
	return p
}

func parsePageSize(r *http.Request) int {
	p, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if p < 1 {
		return 20
	}
	return p
}

// pageWindow returns [start, end) indices for slicing a slice of length total.
func pageWindow(total, page, pageSize int) (int, int) {
	start := (page - 1) * pageSize
	if start >= total {
		return total, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}
