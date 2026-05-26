# Database migrations

Migrations run automatically on panel startup (`backend/internal/store`).

For manual upgrade on a deployed server:

1. Stop the panel process.
2. Backup `panel.db` (path from `SQLITE_PATH`).
3. Apply SQL:

```bash
sqlite3 /path/to/panel.db < db/migrations/001_add_cert_columns.sql
```

4. Start the panel (new binary). Startup migration skips columns that already exist.

| File | Description |
|------|-------------|
| `001_add_cert_columns.sql` | `cert_title`, `cert_key_pass` on `managers` and `vpn_user_meta` |
