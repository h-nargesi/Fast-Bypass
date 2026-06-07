package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"fast-bypass/internal/httpx"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/password"
	"fast-bypass/internal/store"
)

var (
	errInvalidCertTitle = errors.New("invalid cert_title")
)

func userOvpnFileName(mikrotikName string) string {
	return "open-vpns/config-" + mikrotikName + ".ovpn"
}

type openVPNResolved struct {
	useCert     bool
	keyPassword string
	certTitle   string
}

func (a *App) resolveOpenVPN(ctx context.Context, meta *store.VPNUserMeta) (openVPNResolved, error) {
	if meta.CertTitle.Valid && strings.TrimSpace(meta.CertTitle.String) != "" {
		if !meta.CertKeyPass.Valid || meta.CertKeyPass.String == "" {
			return openVPNResolved{}, fmt.Errorf("user cert_key_pass missing")
		}
		return openVPNResolved{
			useCert:     true,
			keyPassword: meta.CertKeyPass.String,
			certTitle:   meta.CertTitle.String,
		}, nil
	}
	if meta.ManagerID.Valid {
		mgr, err := a.Store.FindManagerByID(ctx, meta.ManagerID.Int64)
		if err != nil {
			return openVPNResolved{}, err
		}
		if mgr.CertTitle.Valid && strings.TrimSpace(mgr.CertTitle.String) != "" {
			if !mgr.CertKeyPass.Valid || mgr.CertKeyPass.String == "" {
				return openVPNResolved{}, fmt.Errorf("manager cert_key_pass missing")
			}
			return openVPNResolved{
				useCert:     true,
				keyPassword: mgr.CertKeyPass.String,
				certTitle:   mgr.CertTitle.String,
			}, nil
		}
	}
	return openVPNResolved{useCert: false, keyPassword: a.Cfg.OpenVPNKeyPassword}, nil
}

func patchOvpnFriendlyName(body []byte, mikrotikName string) []byte {
	line := fmt.Sprintf(`setenv FRIENDLY_NAME "Sabalan %s"`, mikrotikName)
	s := string(body)
	if strings.Contains(s, "setenv FRIENDLY_NAME") {
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "setenv FRIENDLY_NAME") {
				lines[i] = line
				return []byte(strings.Join(lines, "\n"))
			}
		}
	}
	return []byte(line + "\n" + s)
}

func patchOvpnServerAddress(body []byte, serverAdr string) []byte {
	line := fmt.Sprintf(`remote %s 1194 tcp`, serverAdr)
	s := string(body)
	if strings.Contains(s, "remote ") {
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "remote ") {
				lines[i] = line
				return []byte(strings.Join(lines, "\n"))
			}
		}
	}
	return []byte(line + "\n" + s)
}

func (a *App) ensureUserOvpnFile(mikrotikName, certTitle string) error {
	src := userOvpnFileName(certTitle)
	body, err := a.MT.ReadFileContents(src)
	if err != nil {
		return err
	}
	body = patchOvpnFriendlyName(body, mikrotikName)
	body = patchOvpnServerAddress(body, a.Cfg.L2TPServer) //TODO: write test
	return a.MT.WriteFileContents(userOvpnFileName(mikrotikName), body)
}

func (a *App) provisionCertificate(title, passphrase string) error {
	return a.MT.GenerateCertificate(a.Cfg.MikrotikCertScript, title, passphrase)
}

// setupVPNCertificates runs after MikroTik user exists, before SQLite insert.
// mgr is nil for orphan users.
func (a *App) setupVPNCertificates(ctx context.Context, mikrotikName string, mgr *store.Manager, adminCertTitle *string) (certTitle, certKeyPass sql.NullString, err error) {
	title := ""
	if adminCertTitle != nil {
		title = strings.TrimSpace(*adminCertTitle)
	}
	if title != "" {
		if !password.ValidCertTitle(title) {
			return sql.NullString{}, sql.NullString{}, errInvalidCertTitle
		}
		pass, err := password.GenerateCertKeyPass()
		if err != nil {
			return sql.NullString{}, sql.NullString{}, err
		}
		if err := a.provisionCertificate(title, pass); err != nil {
			return sql.NullString{}, sql.NullString{}, err
		}
		if err := a.ensureUserOvpnFile(mikrotikName, title); err != nil {
			return sql.NullString{}, sql.NullString{}, err
		}
		return sql.NullString{String: title, Valid: true},
			sql.NullString{String: pass, Valid: true}, nil
	}
	if mgr != nil && mgr.CertTitle.Valid && strings.TrimSpace(mgr.CertTitle.String) != "" {
		t := mgr.CertTitle.String
		if err := a.ensureUserOvpnFile(mikrotikName, t); err != nil {
			return sql.NullString{}, sql.NullString{}, err
		}
	}
	return sql.NullString{}, sql.NullString{}, nil
}

func (a *App) provisionUserCertificate(ctx context.Context, meta *store.VPNUserMeta, title string) error {
	title = strings.TrimSpace(title)
	if !password.ValidCertTitle(title) {
		return errInvalidCertTitle
	}
	pass, err := password.GenerateCertKeyPass()
	if err != nil {
		return err
	}
	if err := a.provisionCertificate(title, pass); err != nil {
		return err
	}
	if err := a.ensureUserOvpnFile(meta.MikrotikName, title); err != nil {
		return err
	}
	return a.Store.UpdateVPNMetaCert(ctx, meta.ID, title, pass)
}

func (a *App) applyVPNUserCertTitleChange(ctx context.Context, meta *store.VPNUserMeta, newTitle string) error {
	newTitle = strings.TrimSpace(newTitle)
	oldTitle := ""
	if meta.CertTitle.Valid {
		oldTitle = strings.TrimSpace(meta.CertTitle.String)
	}
	if newTitle == oldTitle {
		return nil
	}
	if newTitle == "" {
		return a.Store.ClearVPNMetaCert(ctx, meta.ID)
	}
	return a.provisionUserCertificate(ctx, meta, newTitle)
}

func (a *App) applyManagerCertTitleChange(ctx context.Context, mgr *store.Manager, newTitle string) error {
	newTitle = strings.TrimSpace(newTitle)
	oldTitle := ""
	if mgr.CertTitle.Valid {
		oldTitle = strings.TrimSpace(mgr.CertTitle.String)
	}
	if newTitle == oldTitle {
		return nil
	}
	if newTitle == "" {
		return a.Store.ClearManagerCert(ctx, mgr.ID)
	}
	return a.setupManagerCertificate(ctx, mgr.ID, newTitle)
}

func (a *App) setupManagerCertificate(ctx context.Context, managerID int64, certTitle string) error {
	title := strings.TrimSpace(certTitle)
	if title == "" {
		return nil
	}
	if !password.ValidCertTitle(title) {
		return errInvalidCertTitle
	}
	pass, err := password.GenerateCertKeyPass()
	if err != nil {
		return err
	}
	if err := a.provisionCertificate(title, pass); err != nil {
		return err
	}
	return a.Store.UpdateManagerCert(ctx, managerID, title, pass)
}

func (a *App) ovpnBodyForMeta(ctx context.Context, meta *store.VPNUserMeta, u *mikrotik.User) ([]byte, error) {
	resolved, err := a.resolveOpenVPN(ctx, meta)
	if err != nil {
		return nil, err
	}
	if resolved.useCert {
		return a.MT.ReadFileContents(userOvpnFileName(meta.MikrotikName))
	}
	if u == nil {
		var err error
		u, err = a.MT.GetUser(meta.MikrotikName)
		if err != nil {
			return nil, err
		}
	}
	return a.renderOvpn(u.Name, u.Password, resolved.keyPassword)
}

func (a *App) connectionBundleFor(ctx context.Context, meta *store.VPNUserMeta, u *mikrotik.User) (map[string]any, error) {
	resolved, err := a.resolveOpenVPN(ctx, meta)
	if err != nil {
		return nil, err
	}
	pw := any(nil)
	if u != nil && u.Password != "" {
		pw = u.Password
	}
	name := meta.MikrotikName
	if u != nil && u.Name != "" {
		name = u.Name
	}
	return map[string]any{
		"username":             name,
		"password":             pw,
		"openvpn_key_password": resolved.keyPassword,
		"l2tp_ipsec_secret":    a.Cfg.L2TPIPsecSecret,
		"l2tp_server":          a.Cfg.L2TPServer,
		"openvpn_download_url": a.Cfg.OpenVPNDownloadURL,
	}, nil
}

func (a *App) connectionBundleForOwner(ctx context.Context, mikrotikName string, ownerID int64, u *mikrotik.User) (map[string]any, error) {
	meta, err := a.Store.FindVPNMetaByName(ctx, mikrotikName)
	if err == nil {
		return a.connectionBundleFor(ctx, meta, u)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	synthetic := &store.VPNUserMeta{MikrotikName: mikrotikName}
	if ownerID != 0 {
		synthetic.ManagerID = sql.NullInt64{Int64: ownerID, Valid: true}
	}
	return a.connectionBundleFor(ctx, synthetic, u)
}

func (a *App) writeOvpnDownload(w http.ResponseWriter, r *http.Request, meta *store.VPNUserMeta) {
	ctx := r.Context()
	u, err := a.MT.GetUser(meta.MikrotikName)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "کاربر در روتر یافت نشد")
		return
	}
	resolved, err := a.resolveOpenVPN(ctx, meta)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "خطای سرور")
		return
	}
	if !resolved.useCert && u.Password == "" {
		httpx.WriteError(w, http.StatusServiceUnavailable, "MIKROTIK_UNAVAILABLE", "رمز در دسترس نیست")
		return
	}
	body, err := a.ovpnBodyForMeta(ctx, meta, u)
	if err != nil {
		if errors.Is(err, mikrotik.ErrNotFound) {
			httpx.WriteError(w, http.StatusServiceUnavailable, "OVPN_MISSING", "فایل پیکربندی OpenVPN یافت نشد")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "TEMPLATE_MISSING", "قالب ovpn پیکربندی نشده")
		return
	}
	w.Header().Set("Content-Type", "application/x-openvpn-profile")
	w.Header().Set("Content-Disposition", `attachment; filename="`+meta.MikrotikName+`.ovpn"`)
	_, _ = w.Write(body)
}
