package mikrotik

import (
	"crypto/tls"
	"fmt"

	"fast-bypass/internal/config"
)

// NewFromConfig returns a MikroTik client (fake or RouterOS API).
func NewFromConfig(cfg config.Config) (Client, error) {
	if cfg.MikrotikFake {
		return NewFake(), nil
	}
	addr := fmt.Sprintf("%s:%d", cfg.MikrotikHost, cfg.MikrotikPort)
	var tlsCfg *tls.Config
	if cfg.MikrotikUseTLS {
		tlsCfg = &tls.Config{InsecureSkipVerify: cfg.MikrotikTLSInsec} //nolint:gosec
	}
	return NewRouterOS(addr, cfg.MikrotikUser, cfg.MikrotikPass, cfg.MikrotikUseTLS, tlsCfg, cfg.MikrotikTimeout), nil
}
