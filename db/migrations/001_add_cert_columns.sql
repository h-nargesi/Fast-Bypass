-- Migration 001: OpenVPN certificate metadata (panel SQLite)
-- Safe to run on existing deployments. Idempotent via IF NOT EXISTS pattern in app migrate();
-- Manual run (backup first):
--   sqlite3 /path/to/panel.db < db/migrations/001_add_cert_columns.sql

ALTER TABLE managers ADD COLUMN cert_title TEXT;
ALTER TABLE managers ADD COLUMN cert_key_pass TEXT;

ALTER TABLE vpn_user_meta ADD COLUMN cert_title TEXT;
ALTER TABLE vpn_user_meta ADD COLUMN cert_key_pass TEXT;
