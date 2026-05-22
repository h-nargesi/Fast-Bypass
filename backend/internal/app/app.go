package app

import (
	"context"
	"log/slog"
	"time"

	"fast-bypass/internal/auth"
	"fast-bypass/internal/config"
	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/owner"
	"fast-bypass/internal/store"
)

type App struct {
	Cfg    config.Config
	Log    *slog.Logger
	Store  *store.Store
	MT     mikrotik.Client
	Auth   *auth.Issuer
	loc    *time.Location
}

func New(cfg config.Config, log *slog.Logger, st *store.Store) (*App, error) {
	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		loc = time.UTC
	}
	var mt mikrotik.Client = mikrotik.NewFake()
	if !cfg.MikrotikFake {
		mt = mikrotik.NewFake()
	}
	cached := mikrotik.NewCached(mt, cfg.MikrotikCacheTTL)
	return &App{
		Cfg:   cfg,
		Log:   log,
		Store: st,
		MT:    cached,
		Auth:  auth.NewIssuer(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL),
		loc:   loc,
	}, nil
}

func (a *App) Now() time.Time {
	return time.Now().In(a.loc)
}

func (a *App) Registry(ctx context.Context) (owner.Registry, error) {
	managers, err := a.Store.ListManagers(ctx)
	if err != nil {
		return owner.Registry{}, err
	}
	return owner.BuildRegistry(managers, a.Cfg.UsernamePrefixSep), nil
}

func (a *App) cachedMT() *mikrotik.CachedClient {
	if c, ok := a.MT.(*mikrotik.CachedClient); ok {
		return c
	}
	return nil
}

// UseMikrotikClient replaces the router client (for tests).
func (a *App) UseMikrotikClient(c mikrotik.Client) {
	a.MT = mikrotik.NewCached(c, a.Cfg.MikrotikCacheTTL)
}
