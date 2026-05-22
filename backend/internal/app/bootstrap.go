package app

import (
	"context"

	"fast-bypass/internal/password"
)

func (a *App) BootstrapAdmin(ctx context.Context) error {
	n, err := a.Store.AdminCount(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if a.Cfg.AdminPassword == "" || a.Cfg.AdminPassword == "change-me" {
		a.Log.Warn("bootstrap admin skipped: set ADMIN_PASSWORD in .env")
		return nil
	}
	hash, err := password.Hash(a.Cfg.AdminPassword)
	if err != nil {
		return err
	}
	if err := a.Store.CreateAdmin(ctx, a.Cfg.AdminUsername, hash); err != nil {
		return err
	}
	a.Log.Info("bootstrap admin created", "username", a.Cfg.AdminUsername)
	return nil
}
