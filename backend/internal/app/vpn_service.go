package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/httpx"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/owner"
	"fast-bypass/internal/password"
	"fast-bypass/internal/quota"
	"fast-bypass/internal/store"
)

func (a *App) managerUsedQuota(ctx context.Context, managerID int64) (int, error) {
	reg, err := a.Registry(ctx)
	if err != nil {
		return 0, err
	}
	users, err := a.MT.ListUsers()
	if err != nil {
		return 0, err
	}
	return quota.UsedForManager(reg, managerID, users, a.MT.ListUserProfiles, a.Now())
}

func (a *App) listUsers(ctx context.Context, refresh bool) ([]mikrotik.User, error) {
	if refresh {
		if c := a.cachedMT(); c != nil {
			return c.RefreshListUsers()
		}
	}
	return a.MT.ListUsers()
}

func (a *App) managerByID(ctx context.Context, id int64) (*store.Manager, error) {
	return a.Store.FindManagerByID(ctx, id)
}

func (a *App) enrichOwner(ctx context.Context, reg owner.Registry, name, comment string) (*int64, *string, *string, *string, bool) {
	id := reg.Resolve(name, comment)
	if id == 0 {
		return nil, nil, nil, nil, reg.OwnerMismatch(name, comment, 0)
	}
	m, err := a.Store.FindManagerByID(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, false
	}
	dn := m.DisplayName
	un := m.Username
	sl := m.Slug
	mid := m.ID
	return &mid, &dn, &un, &sl, reg.OwnerMismatch(name, comment, id)
}

func (a *App) connectionBundle(u *mikrotik.User) map[string]any {
	pw := any(nil)
	if u != nil && u.Password != "" {
		pw = u.Password
	}
	return map[string]any{
		"username":               u.Name,
		"password":               pw,
		"openvpn_key_password":   a.Cfg.OpenVPNKeyPassword,
		"l2tp_ipsec_secret":      a.Cfg.L2TPIPsecSecret,
		"l2tp_server":            a.Cfg.L2TPServer,
		"openvpn_download_url":   a.Cfg.OpenVPNDownloadURL,
	}
}

func (a *App) buildVPNDetail(ctx context.Context, reg owner.Registry, meta *store.VPNUserMeta, includeComment bool) (map[string]any, error) {
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil {
		return nil, err
	}
	profs, err := a.MT.ListUserProfiles(meta.MikrotikName)
	if err != nil {
		return nil, err
	}
	acts, _ := a.Store.ListActivationsByMetaID(ctx, meta.ID)
	mid, dn, un, sl, mismatch := a.enrichOwner(ctx, reg, u.Name, u.Comment)
	out := map[string]any{
		"id": meta.ID, "mikrotik_name": u.Name,
		"shared_users": u.SharedUsers,
		"disabled": u.Disabled,
		"contact_info": nullStrVal(meta.ContactInfo),
		"notes": nullStrVal(meta.Notes),
		"profiles": profileDTOs(profs), "activations": activationDTOs(acts),
		"connection_bundle": a.connectionBundle(u),
		"manager_id": mid, "manager_display_name": dn, "manager_username": un, "manager_slug": sl,
		"owner_mismatch": mismatch,
	}
	if includeComment {
		out["mikrotik_comment"] = u.Comment
	}
	return out, nil
}

func activationDTOs(acts []store.ProfileActivation) []map[string]any {
	var out []map[string]any
	for _, a := range acts {
		row := map[string]any{
			"id": a.ID, "profile_name": a.ProfileName, "shared_users": a.SharedUsers,
			"currency": a.Currency, "is_settled": a.IsSettled, "created_at": a.CreatedAt,
		}
		if a.AmountPaid.Valid {
			row["amount_paid"] = a.AmountPaid.Float64
		}
		if a.MikrotikEndTime.Valid {
			row["mikrotik_end_time"] = a.MikrotikEndTime.String
		}
		if a.Note.Valid {
			row["note"] = a.Note.String
		}
		out = append(out, row)
	}
	return out
}

func profileDTOs(profs []mikrotik.UserProfile) []map[string]any {
	var out []map[string]any
	for _, p := range profs {
		out = append(out, map[string]any{
			"id": p.ID, "profile": p.Profile, "state": p.State, "end_time": p.EndTime,
		})
	}
	return out
}

func nullStrVal(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func (a *App) fullName(prefix, local string) string {
	return prefix + local
}

func resolveVPNPassword(pw string) (string, error) {
	pw = strings.TrimSpace(pw)
	if pw == "" {
		return password.GenerateVPN()
	}
	if !password.ValidVPN(pw) {
		return "", fmt.Errorf("invalid password")
	}
	return pw, nil
}

func (a *App) createVPNUser(ctx context.Context, mgr *store.Manager, req createVPNReq) (map[string]any, error) {
	reg, err := a.Registry(ctx)
	if err != nil {
		return nil, err
	}
	prefix := mgr.Slug + a.Cfg.UsernamePrefixSep
	if len(req.LocalName) > a.Cfg.UsernameLocalMaxLen {
		return nil, fmt.Errorf("local name too long")
	}
	pass, err := resolveVPNPassword(req.Password)
	if err != nil {
		return nil, err
	}
	if req.SharedUsers < 1 || req.SharedUsers > a.Cfg.SharedUsersMax {
		return nil, fmt.Errorf("invalid shared_users")
	}
	name := a.fullName(prefix, req.LocalName)
	comment := reg.PanelComment(owner.ManagerInfo{ID: mgr.ID, Slug: mgr.Slug})

	used, err := a.managerUsedQuota(ctx, mgr.ID)
	if err != nil {
		return nil, err
	}
	assign := req.AssignProfile == nil || *req.AssignProfile
	profile := a.Cfg.DefaultProfile
	if req.ProfileName != nil && *req.ProfileName != "" {
		profile = *req.ProfileName
	}
	if assign && !quota.CheckAdd(used, mgr.Quota, req.SharedUsers) {
		return nil, errQuotaExceeded
	}

	if err := a.MT.AddUser(name, pass, comment, req.SharedUsers, vpnUserDisabled(req.Disabled)); err != nil {
		if errors.Is(err, mikrotik.ErrNameTaken) {
			return nil, errNameTaken
		}
		return nil, err
	}

	mid := mgr.ID
	meta := &store.VPNUserMeta{
		MikrotikName: name, ManagerID: sql.NullInt64{Int64: mid, Valid: true},
	}
	if req.ContactInfo != nil {
		meta.ContactInfo = sql.NullString{String: *req.ContactInfo, Valid: true}
	}
	if req.Notes != nil {
		meta.Notes = sql.NullString{String: *req.Notes, Valid: true}
	}
	if err := a.Store.CreateVPNMeta(ctx, meta); err != nil {
		_ = a.MT.RemoveUser(name)
		return nil, err
	}

	var profs []mikrotik.UserProfile
	if assign {
		if err := a.MT.AddUserProfile(name, profile); err != nil {
			_ = a.MT.RemoveUser(name)
			_ = a.Store.DeleteVPNMeta(ctx, meta.ID)
			return nil, err
		}
		profs, _ = a.MT.ListUserProfiles(name)
		_ = a.recordActivation(ctx, meta.ID, profile, req.SharedUsers, req.AmountPaid, req.Currency, nil, profs)
	}

	return map[string]any{
		"id": meta.ID, "mikrotik_name": name,
		"shared_users": req.SharedUsers, "disabled": vpnUserDisabled(req.Disabled),
		"profiles": profileDTOs(profs),
	}, nil
}

func (a *App) createOrphanVPNUser(ctx context.Context, req createVPNReq) (map[string]any, error) {
	name := strings.TrimSpace(req.LocalName)
	if name == "" || len(name) > 32 {
		return nil, fmt.Errorf("invalid name")
	}
	pass, err := resolveVPNPassword(req.Password)
	if err != nil {
		return nil, err
	}
	if req.SharedUsers < 1 || req.SharedUsers > a.Cfg.SharedUsersMax {
		return nil, fmt.Errorf("invalid shared_users")
	}
	if err := a.MT.AddUser(name, pass, "", req.SharedUsers, vpnUserDisabled(req.Disabled)); err != nil {
		if errors.Is(err, mikrotik.ErrNameTaken) {
			return nil, errNameTaken
		}
		return nil, err
	}
	meta := &store.VPNUserMeta{MikrotikName: name}
	if req.ContactInfo != nil {
		meta.ContactInfo = sql.NullString{String: *req.ContactInfo, Valid: true}
	}
	if req.Notes != nil {
		meta.Notes = sql.NullString{String: *req.Notes, Valid: true}
	}
	if err := a.Store.CreateVPNMeta(ctx, meta); err != nil {
		_ = a.MT.RemoveUser(name)
		return nil, err
	}
	assign := req.AssignProfile == nil || *req.AssignProfile
	profile := a.Cfg.DefaultProfile
	if req.ProfileName != nil && *req.ProfileName != "" {
		profile = *req.ProfileName
	}
	var profs []mikrotik.UserProfile
	if assign {
		if err := a.MT.AddUserProfile(name, profile); err != nil {
			_ = a.MT.RemoveUser(name)
			_ = a.Store.DeleteVPNMeta(ctx, meta.ID)
			return nil, err
		}
		profs, _ = a.MT.ListUserProfiles(name)
		_ = a.recordActivation(ctx, meta.ID, profile, req.SharedUsers, req.AmountPaid, req.Currency, nil, profs)
	}
	return map[string]any{
		"id": meta.ID, "mikrotik_name": name,
		"shared_users": req.SharedUsers, "disabled": vpnUserDisabled(req.Disabled),
		"profiles": profileDTOs(profs),
	}, nil
}

func vpnUserDisabled(p *bool) bool {
	return p != nil && *p
}

var (
	errQuotaExceeded   = errors.New("quota exceeded")
	errNameTaken       = errors.New("name taken")
	errNotOwner        = errors.New("not owner")
	errOrphanNoOwner   = errors.New("orphan no owner")
	errOwnerDBMismatch = errors.New("owner db mismatch")
)

type patchVPNReq struct {
	Password    *string `json:"password"`
	SharedUsers *int    `json:"shared_users"`
	Disabled    *bool   `json:"disabled"`
	ContactInfo *string `json:"contact_info"`
	Notes       *string `json:"notes"`
}

type adminPatchVPNReq struct {
	patchVPNReq
	ManagerID *int64 `json:"manager_id"`
}

func (a *App) patchVPNUser(ctx context.Context, reg owner.Registry, meta *store.VPNUserMeta, req patchVPNReq, syncManagerID *int64, includeComment bool) (map[string]any, error) {
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil {
		return nil, err
	}
	ownerID := reg.Resolve(u.Name, u.Comment)
	var passPtr *string
	if req.Password != nil {
		pass, err := resolveVPNPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		passPtr = &pass
	}
	comment := u.Comment
	if ownerID != 0 {
		mgr, err := a.Store.FindManagerByID(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		comment = reg.PanelComment(owner.ManagerInfo{ID: mgr.ID, Slug: mgr.Slug})
	}
	if req.SharedUsers != nil {
		if ownerID == 0 {
			return nil, errOrphanNoOwner
		}
		mgr, err := a.Store.FindManagerByID(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		used, err := a.managerUsedQuota(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		profs, _ := a.MT.ListUserProfiles(u.Name)
		active := false
		for _, p := range profs {
			if quota.ProfileActive(p, a.Now()) {
				active = true
				break
			}
		}
		if active && !quota.CheckIncrease(used, mgr.Quota, u.SharedUsers, *req.SharedUsers) {
			return nil, errQuotaExceeded
		}
	}
	var mid *int64
	if syncManagerID != nil {
		if ownerID == 0 {
			return nil, errOrphanNoOwner
		}
		if *syncManagerID != ownerID {
			return nil, errOwnerDBMismatch
		}
		mid = syncManagerID
	}
	if err := a.MT.SetUser(u.Name, passPtr, req.SharedUsers, comment, req.Disabled); err != nil {
		return nil, err
	}
	if err := a.Store.UpdateVPNMeta(ctx, meta.ID, req.ContactInfo, req.Notes, mid); err != nil {
		return nil, err
	}
	meta, err = a.Store.FindVPNMetaByID(ctx, meta.ID)
	if err != nil {
		return nil, err
	}
	return a.buildVPNDetail(ctx, reg, meta, includeComment)
}

func (a *App) deleteVPNUser(ctx context.Context, meta *store.VPNUserMeta) error {
	_ = a.MT.RemoveUser(meta.MikrotikName)
	return a.Store.DeleteVPNMeta(ctx, meta.ID)
}

func (a *App) assignVPNProfile(ctx context.Context, reg owner.Registry, meta *store.VPNUserMeta, req assignProfileReq, includeComment bool) (map[string]any, error) {
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil {
		return nil, err
	}
	if reg.Resolve(u.Name, u.Comment) == 0 {
		return nil, errOrphanNoOwner
	}
	if err := a.MT.AddUserProfile(u.Name, req.ProfileName); err != nil {
		return nil, err
	}
	profs, _ := a.MT.ListUserProfiles(u.Name)
	_ = a.recordActivation(ctx, meta.ID, req.ProfileName, u.SharedUsers, req.AmountPaid, req.Currency, req.Note, profs)
	return a.buildVPNDetail(ctx, reg, meta, includeComment)
}

func (a *App) removeVPNProfile(meta *store.VPNUserMeta, profileRowID string) error {
	return a.MT.RemoveUserProfile(profileRowID)
}

func (a *App) writeVPNPatchError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errQuotaExceeded):
		httpx.WriteError(w, http.StatusConflict, "QUOTA_EXCEEDED", "سقف اتصال همزمان پر است")
	case errors.Is(err, errOrphanNoOwner):
		httpx.WriteError(w, http.StatusForbidden, "ORPHAN_NO_OWNER", "کاربر بدون مدیر — ابتدا مالک را در روتر مشخص کنید")
	case errors.Is(err, errOwnerDBMismatch):
		httpx.WriteError(w, http.StatusConflict, "OWNER_MISMATCH", "manager_id با مالک واقعی روتر هم‌خوان نیست")
	default:
		if err != nil && err.Error() == "invalid password" {
			httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "رمز VPN نامعتبر")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "ارتباط با روتر برقرار نشد")
	}
}

type createVPNReq struct {
	LocalName     string   `json:"local_name"` // suffix with manager; full mikrotik name when orphan (admin)
	Password      string   `json:"password"`
	SharedUsers   int      `json:"shared_users"`
	Disabled      *bool    `json:"disabled"` // true = غیرفعال در User Manager روتر
	ContactInfo   *string  `json:"contact_info"`
	Notes         *string  `json:"notes"`
	AssignProfile *bool    `json:"assign_profile"`
	ProfileName   *string  `json:"profile_name"`
	AmountPaid    *float64 `json:"amount_paid"`
	Currency      *string  `json:"currency"`
	Note          *string  `json:"note"`
}

func (a *App) recordActivation(ctx context.Context, metaID int64, profile string, shared int, amount *float64, currency, note *string, profs []mikrotik.UserProfile) error {
	var end string
	for _, p := range profs {
		if quota.ProfileActive(p, a.Now()) {
			end = p.EndTime
			break
		}
	}
	cur := "IRR"
	if currency != nil {
		cur = *currency
	}
	act := &store.ProfileActivation{
		VPNUserMetaID: metaID, ProfileName: profile, SharedUsers: shared, Currency: cur,
	}
	if amount != nil {
		act.AmountPaid = sql.NullFloat64{Float64: *amount, Valid: true}
	}
	if note != nil {
		act.Note = sql.NullString{String: *note, Valid: true}
	}
	if end != "" {
		act.MikrotikEndTime = sql.NullString{String: end, Valid: true}
	}
	return a.Store.CreateActivation(ctx, act)
}

func (a *App) assertManagerOwner(reg owner.Registry, c *auth.Claims, name, comment string) error {
	if reg.Resolve(name, comment) != c.ManagerID {
		return errNotOwner
	}
	return nil
}

func (a *App) renderOvpn(username, pass string) ([]byte, error) {
	b, err := a.readOvpnTemplate()
	if err != nil {
		return nil, err
	}
	s := string(b)
	s = strings.ReplaceAll(s, "{{username}}", username)
	s = strings.ReplaceAll(s, "{{password}}", pass)
	s = strings.ReplaceAll(s, "{{openvpn_key_password}}", a.Cfg.OpenVPNKeyPassword)
	return []byte(s), nil
}
