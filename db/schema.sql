-- User Manager Panel — SQLite schema v1
-- Run migrations from backend on startup; this file is the reference DDL.

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS managers (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash   TEXT NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    slug            TEXT NOT NULL UNIQUE COLLATE NOCASE,
    -- Max concurrent slots: sum(shared_users) for VPN users with active profile
    quota           INTEGER NOT NULL CHECK (quota > 0),
    is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    cert_title      TEXT,  -- OpenVPN cert TITLE (shared across managers OK; see docs/certificates.md)
    cert_key_pass   TEXT,  -- panel-generated export passphrase; not read from router .pass files
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS panel_admins (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash   TEXT NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- One row per MikroTik user name (full name with prefix)
CREATE TABLE IF NOT EXISTS vpn_user_meta (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    mikrotik_name   TEXT NOT NULL UNIQUE,
    manager_id      INTEGER REFERENCES managers(id) ON DELETE SET NULL,
    contact_info    TEXT,
    notes           TEXT,
    cert_title      TEXT,  -- optional; NOT UNIQUE — multiple users may share one certificate
    cert_key_pass   TEXT,  -- set when admin creates user with cert_title
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Payment / activation record per profile assignment (panel-only)
CREATE TABLE IF NOT EXISTS profile_activations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    vpn_user_meta_id INTEGER NOT NULL REFERENCES vpn_user_meta(id) ON DELETE CASCADE,
    profile_name    TEXT NOT NULL,
    -- Snapshot at assign time (for admin renewal ledger / settlement sums)
    shared_users    INTEGER NOT NULL CHECK (shared_users > 0),
    amount_paid     REAL,  -- NULL = not entered; 0 = explicit zero
    currency        TEXT NOT NULL DEFAULT 'IRR',
    paid_at         TEXT,
    note            TEXT,
    mikrotik_end_time TEXT,
    is_settled      INTEGER NOT NULL DEFAULT 0 CHECK (is_settled IN (0, 1)),
    settled_at      TEXT,
    settled_by_admin_id INTEGER REFERENCES panel_admins(id) ON DELETE SET NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_vpn_user_meta_manager ON vpn_user_meta(manager_id);
CREATE INDEX IF NOT EXISTS idx_profile_activations_user ON profile_activations(vpn_user_meta_id);
CREATE INDEX IF NOT EXISTS idx_profile_activations_created ON profile_activations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_profile_activations_settled ON profile_activations(is_settled, created_at);
