package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_time_format=sqlite")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) migrate() error {
	b, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	if _, err = s.db.Exec(string(b)); err != nil {
		return err
	}
	return s.migrateVPNMetaContactInfo()
}

// migrateVPNMetaContactInfo drops legacy local_name/contact_phone and renames contact_note → contact_info.
func (s *Store) migrateVPNMetaContactInfo() error {
	has, err := s.tableHasColumn("vpn_user_meta", "local_name")
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []string{
		`CREATE TABLE vpn_user_meta_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mikrotik_name TEXT NOT NULL UNIQUE,
			manager_id INTEGER REFERENCES managers(id) ON DELETE SET NULL,
			contact_info TEXT,
			notes TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`INSERT INTO vpn_user_meta_new (id, mikrotik_name, manager_id, contact_info, notes, created_at, updated_at)
		 SELECT id, mikrotik_name, manager_id, contact_note, notes, created_at, updated_at FROM vpn_user_meta`,
		`DROP TABLE vpn_user_meta`,
		`ALTER TABLE vpn_user_meta_new RENAME TO vpn_user_meta`,
		`CREATE INDEX IF NOT EXISTS idx_vpn_user_meta_manager ON vpn_user_meta(manager_id)`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) tableHasColumn(table, column string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Store) AdminCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM panel_admins`).Scan(&n)
	return n, err
}

func (s *Store) CreateAdmin(ctx context.Context, username, hash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO panel_admins (username, password_hash) VALUES (?, ?)`,
		username, hash)
	return err
}

func (s *Store) FindAdminByUsername(ctx context.Context, username string) (*PanelAdmin, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, is_active FROM panel_admins WHERE username = ? COLLATE NOCASE`,
		username)
	var a PanelAdmin
	var active int
	if err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &active); err != nil {
		return nil, err
	}
	a.IsActive = active == 1
	return &a, nil
}

func (s *Store) FindAdminByID(ctx context.Context, id int64) (*PanelAdmin, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, is_active FROM panel_admins WHERE id = ?`, id)
	var a PanelAdmin
	var active int
	if err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &active); err != nil {
		return nil, err
	}
	a.IsActive = active == 1
	return &a, nil
}

func (s *Store) UpdateAdminPassword(ctx context.Context, id int64, hash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE panel_admins SET password_hash = ? WHERE id = ?`, hash, id)
	return err
}

func (s *Store) FindManagerByUsername(ctx context.Context, username string) (*Manager, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, display_name, slug, quota, is_active, created_at, updated_at
		 FROM managers WHERE username = ? COLLATE NOCASE`, username)
	return scanManager(row)
}

func (s *Store) FindManagerByID(ctx context.Context, id int64) (*Manager, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, display_name, slug, quota, is_active, created_at, updated_at
		 FROM managers WHERE id = ?`, id)
	return scanManager(row)
}

func (s *Store) ListManagers(ctx context.Context) ([]Manager, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, display_name, slug, quota, is_active, created_at, updated_at
		 FROM managers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Manager
	for rows.Next() {
		m, err := scanManager(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (s *Store) ListManagerSlugs(ctx context.Context) ([]ManagerSlug, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, slug FROM managers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ManagerSlug
	for rows.Next() {
		var m ManagerSlug
		if err := rows.Scan(&m.ID, &m.Slug); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateManager(ctx context.Context, m *Manager) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO managers (username, password_hash, display_name, slug, quota)
		 VALUES (?, ?, ?, ?, ?)`,
		m.Username, m.PasswordHash, m.DisplayName, m.Slug, m.Quota)
	if err != nil {
		return err
	}
	m.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateManager(ctx context.Context, id int64, username, displayName *string, quota *int, isActive *bool, passwordHash *string) error {
	m, err := s.FindManagerByID(ctx, id)
	if err != nil {
		return err
	}
	if username != nil {
		m.Username = *username
	}
	if displayName != nil {
		m.DisplayName = *displayName
	}
	if quota != nil {
		m.Quota = *quota
	}
	if isActive != nil {
		m.IsActive = *isActive
	}
	if passwordHash != nil {
		m.PasswordHash = *passwordHash
	}
	active := 0
	if m.IsActive {
		active = 1
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE managers SET username=?, display_name=?, quota=?, is_active=?, password_hash=?, updated_at=datetime('now') WHERE id=?`,
		m.Username, m.DisplayName, m.Quota, active, m.PasswordHash, id)
	return err
}

func (s *Store) ManagerVPNCount(ctx context.Context, managerID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM vpn_user_meta WHERE manager_id = ?`, managerID).Scan(&n)
	return n, err
}

func (s *Store) SlugExists(ctx context.Context, slug string, excludeID int64) (bool, error) {
	var n int
	q := `SELECT COUNT(*) FROM managers WHERE slug = ? COLLATE NOCASE`
	args := []any{slug}
	if excludeID > 0 {
		q += ` AND id != ?`
		args = append(args, excludeID)
	}
	err := s.db.QueryRowContext(ctx, q, args...).Scan(&n)
	return n > 0, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanManager(row scanner) (*Manager, error) {
	var m Manager
	var active int
	if err := row.Scan(&m.ID, &m.Username, &m.PasswordHash, &m.DisplayName, &m.Slug,
		&m.Quota, &active, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	m.IsActive = active == 1
	return &m, nil
}

func (s *Store) FindVPNMetaByID(ctx context.Context, id int64) (*VPNUserMeta, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, mikrotik_name, manager_id, contact_info, notes, created_at, updated_at
		 FROM vpn_user_meta WHERE id = ?`, id)
	return scanVPNMeta(row)
}

func (s *Store) FindVPNMetaByName(ctx context.Context, name string) (*VPNUserMeta, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, mikrotik_name, manager_id, contact_info, notes, created_at, updated_at
		 FROM vpn_user_meta WHERE mikrotik_name = ?`, name)
	return scanVPNMeta(row)
}

func (s *Store) CreateVPNMeta(ctx context.Context, m *VPNUserMeta) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO vpn_user_meta (mikrotik_name, manager_id, contact_info, notes)
		 VALUES (?, ?, ?, ?)`,
		m.MikrotikName, m.ManagerID, m.ContactInfo, m.Notes)
	if err != nil {
		return err
	}
	m.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateVPNMeta(ctx context.Context, id int64, contactInfo, notes *string, managerID *int64) error {
	cur, err := s.FindVPNMetaByID(ctx, id)
	if err != nil {
		return err
	}
	if contactInfo != nil {
		cur.ContactInfo = sql.NullString{String: *contactInfo, Valid: true}
	}
	if notes != nil {
		cur.Notes = sql.NullString{String: *notes, Valid: true}
	}
	if managerID != nil {
		cur.ManagerID = sql.NullInt64{Int64: *managerID, Valid: true}
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE vpn_user_meta SET contact_info=?, notes=?, manager_id=?, updated_at=datetime('now') WHERE id=?`,
		nullStr(cur.ContactInfo), nullStr(cur.Notes), nullInt(cur.ManagerID), id)
	return err
}

func (s *Store) DeleteVPNMeta(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM vpn_user_meta WHERE id = ?`, id)
	return err
}

func scanVPNMeta(row scanner) (*VPNUserMeta, error) {
	var m VPNUserMeta
	var mgr sql.NullInt64
	var contact, notes sql.NullString
	if err := row.Scan(&m.ID, &m.MikrotikName, &mgr, &contact, &notes, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	m.ManagerID = mgr
	m.ContactInfo = contact
	m.Notes = notes
	return &m, nil
}

func (s *Store) CreateActivation(ctx context.Context, a *ProfileActivation) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO profile_activations (vpn_user_meta_id, profile_name, shared_users, amount_paid, currency, paid_at, note, mikrotik_end_time)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.VPNUserMetaID, a.ProfileName, a.SharedUsers, a.AmountPaid, a.Currency, a.PaidAt, a.Note, a.MikrotikEndTime)
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) ListActivationsByMetaID(ctx context.Context, metaID int64) ([]ProfileActivation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, vpn_user_meta_id, profile_name, shared_users, amount_paid, currency, paid_at, note,
		        mikrotik_end_time, is_settled, settled_at, settled_by_admin_id, created_at
		 FROM profile_activations WHERE vpn_user_meta_id = ? ORDER BY created_at DESC`, metaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivations(rows)
}

func (s *Store) UpdateLatestUnsettledActivationSharedUsers(ctx context.Context, metaID int64, sharedUsers int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE profile_activations SET shared_users = ?
		 WHERE id = (
		   SELECT id FROM profile_activations
		   WHERE vpn_user_meta_id = ? AND is_settled = 0
		   ORDER BY created_at DESC, id DESC LIMIT 1
		 )`, sharedUsers, metaID)
	return err
}

func (s *Store) UpdateActivationSharedUsers(ctx context.Context, activationID int64, sharedUsers int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE profile_activations SET shared_users = ? WHERE id = ?`, sharedUsers, activationID)
	return err
}

func (s *Store) ListRenewalSharedUsersInScope(ctx context.Context, filter RenewalFilter) ([]RenewalSharedUsersRow, error) {
	where, args := renewalWhere(filter)
	q := `SELECT pa.id, v.mikrotik_name, pa.shared_users, pa.is_settled
		FROM profile_activations pa
		JOIN vpn_user_meta v ON v.id = pa.vpn_user_meta_id ` + where +
		` ORDER BY pa.created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RenewalSharedUsersRow
	for rows.Next() {
		var r RenewalSharedUsersRow
		var settled int
		if err := rows.Scan(&r.ID, &r.MikrotikName, &r.SharedUsers, &settled); err != nil {
			return nil, err
		}
		r.IsSettled = settled == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) FindActivationByID(ctx context.Context, id int64) (*ProfileActivation, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, vpn_user_meta_id, profile_name, shared_users, amount_paid, currency, paid_at, note,
		        mikrotik_end_time, is_settled, settled_at, settled_by_admin_id, created_at
		 FROM profile_activations WHERE id = ?`, id)
	var a ProfileActivation
	var amount sql.NullFloat64
	var paidAt, note, endTime, settledAt sql.NullString
	var settledBy sql.NullInt64
	var settled int
	if err := row.Scan(&a.ID, &a.VPNUserMetaID, &a.ProfileName, &a.SharedUsers,
		&amount, &a.Currency, &paidAt, &note, &endTime, &settled, &settledAt, &settledBy, &a.CreatedAt); err != nil {
		return nil, err
	}
	a.AmountPaid = amount
	a.PaidAt = paidAt
	a.Note = note
	a.MikrotikEndTime = endTime
	a.IsSettled = settled == 1
	a.SettledAt = settledAt
	a.SettledByAdminID = settledBy
	return &a, nil
}

func scanActivations(rows interface {
	Next() bool
	Scan(...any) error
}) ([]ProfileActivation, error) {
	var out []ProfileActivation
	for rows.Next() {
		var a ProfileActivation
		var amount sql.NullFloat64
		var paidAt, note, endTime, settledAt sql.NullString
		var settledBy sql.NullInt64
		var settled int
		if err := rows.Scan(&a.ID, &a.VPNUserMetaID, &a.ProfileName, &a.SharedUsers,
			&amount, &a.Currency, &paidAt, &note, &endTime, &settled, &settledAt, &settledBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.IsSettled = settled == 1
		a.AmountPaid = amount
		a.PaidAt = paidAt
		a.Note = note
		a.MikrotikEndTime = endTime
		a.SettledAt = settledAt
		a.SettledByAdminID = settledBy
		out = append(out, a)
	}
	return out, nil
}

func nullStr(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func nullInt(ni sql.NullInt64) any {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

func (s *Store) ListRenewals(ctx context.Context, filter RenewalFilter) ([]RenewalRow, int, RenewalSummary, error) {
	where, args := renewalWhere(filter)
	countQ := `SELECT COUNT(*), COALESCE(SUM(CASE WHEN pa.is_settled=0 THEN pa.shared_users ELSE 0 END),0),
		COALESCE(SUM(pa.shared_users),0)
		FROM profile_activations pa
		JOIN vpn_user_meta v ON v.id = pa.vpn_user_meta_id ` + where
	var total int
	var unsettledSum, allSum int
	if err := s.db.QueryRowContext(ctx, countQ, args...).Scan(&total, &unsettledSum, &allSum); err != nil {
		return nil, 0, RenewalSummary{}, err
	}
	summary := RenewalSummary{UnsettledSharedUsersSum: unsettledSum, AllSharedUsersSum: allSum}

	q := `SELECT pa.id, pa.created_at, v.mikrotik_name, v.manager_id, m.display_name,
		pa.shared_users, pa.profile_name, pa.mikrotik_end_time, pa.is_settled, pa.amount_paid, pa.currency
		FROM profile_activations pa
		JOIN vpn_user_meta v ON v.id = pa.vpn_user_meta_id
		LEFT JOIN managers m ON m.id = v.manager_id ` + where +
		` ORDER BY pa.created_at DESC LIMIT ? OFFSET ?`
	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.PageSize
	if size < 1 {
		size = 50
	}
	argsPage := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := s.db.QueryContext(ctx, q, argsPage...)
	if err != nil {
		return nil, 0, summary, err
	}
	defer rows.Close()
	var items []RenewalRow
	for rows.Next() {
		var r RenewalRow
		var mgrID sql.NullInt64
		var mgrName sql.NullString
		var amount sql.NullFloat64
		var endTime sql.NullString
		if err := rows.Scan(&r.ID, &r.RenewedAt, &r.MikrotikName, &mgrID, &mgrName,
			&r.SharedUsers, &r.ProfileName, &endTime, &r.IsSettled, &amount, &r.Currency); err != nil {
			return nil, 0, summary, err
		}
		if mgrID.Valid {
			r.ManagerID = &mgrID.Int64
		}
		if mgrName.Valid {
			r.ManagerDisplayName = &mgrName.String
		}
		if endTime.Valid {
			r.MikrotikEndTime = &endTime.String
		}
		if amount.Valid {
			r.AmountPaid = &amount.Float64
		}
		items = append(items, r)
	}
	return items, total, summary, rows.Err()
}

func renewalWhere(f RenewalFilter) (string, []any) {
	var parts []string
	var args []any
	parts = append(parts, "1=1")
	if f.OrphanOnly {
		parts = append(parts, "v.manager_id IS NULL")
	} else if f.ManagerID != nil {
		parts = append(parts, "v.manager_id = ?")
		args = append(args, *f.ManagerID)
	}
	switch f.Settled {
	case "settled":
		parts = append(parts, "pa.is_settled = 1")
	case "all":
		// no filter
	default:
		parts = append(parts, "pa.is_settled = 0")
	}
	if f.From != "" {
		parts = append(parts, "pa.created_at >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		parts = append(parts, "pa.created_at <= ?")
		args = append(args, f.To)
	}
	if f.Query != "" {
		parts = append(parts, "(v.mikrotik_name LIKE ? OR v.contact_info LIKE ?)")
		q := "%" + f.Query + "%"
		args = append(args, q, q)
	}
	return "WHERE " + stringsJoin(parts, " AND "), args
}

func stringsJoin(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

func (s *Store) SettleThrough(ctx context.Context, adminID int64, through RenewalThrough) (int64, error) {
	act, err := s.FindActivationByID(ctx, through.ActivationID)
	if err != nil {
		return 0, err
	}
	meta, err := s.FindVPNMetaByID(ctx, act.VPNUserMetaID)
	if err != nil {
		return 0, err
	}
	if through.OrphanOnly {
		if meta.ManagerID.Valid {
			return 0, fmt.Errorf("not in orphan scope")
		}
	} else if through.ManagerID != nil {
		if !meta.ManagerID.Valid || meta.ManagerID.Int64 != *through.ManagerID {
			return 0, fmt.Errorf("not in manager scope")
		}
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE profile_activations SET is_settled=1, settled_at=datetime('now'), settled_by_admin_id=?
		 WHERE is_settled=0 AND vpn_user_meta_id IN (
		   SELECT v.id FROM vpn_user_meta v WHERE `+scopeSQL(through)+`
		 ) AND (created_at < ? OR (created_at = ? AND id <= ?))`,
		adminID, act.CreatedAt, act.CreatedAt, act.ID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scopeSQL(t RenewalThrough) string {
	if t.OrphanOnly {
		return "v.manager_id IS NULL"
	}
	return "v.manager_id = " + fmt.Sprintf("%d", *t.ManagerID)
}
