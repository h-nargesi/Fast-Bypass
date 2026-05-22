//go:build vm

package vmtest

import (
	"os"
	"time"

	"github.com/joho/godotenv"

	"fast-bypass/internal/config"
)

// VMConfig holds MikroTik + VirtualBox settings for VM-backed tests.
type VMConfig struct {
	config.Config
	VMName     string
	VMSnapshot string
	ManageVM   bool
	WaitHost   string
	WaitPort   int
	WaitTimeout time.Duration
}

// LoadConfig reads .env and environment for VM tests.
func LoadConfig() VMConfig {
	for _, p := range []string{".env", "../.env", "../../.env"} {
		_ = godotenv.Load(p)
	}
	cfg := config.Load()
	cfg.MikrotikFake = false
	if cfg.MikrotikHost == "192.168.88.1" || cfg.MikrotikHost == "192.168.56.2" {
		cfg.MikrotikHost = "192.168.56.11"
	}
	if cfg.MikrotikUser == "" || cfg.MikrotikUser == "api-panel" {
		cfg.MikrotikUser = "admin"
	}
	if cfg.MikrotikPass == "" {
		cfg.MikrotikPass = "admin"
	}
	if !cfg.MikrotikTLSInsec {
		cfg.MikrotikTLSInsec = envBool("MIKROTIK_TLS_INSECURE", true)
	}
	return VMConfig{
		Config:      cfg,
		VMName:      env("MIKROTIK_VM_NAME", "Mikrotik-Base"),
		VMSnapshot:  env("MIKROTIK_VM_SNAPSHOT", "clean-test-state"),
		ManageVM:    envBool("MIKROTIK_VM_MANAGE", true),
		WaitHost:    cfg.MikrotikHost,
		WaitPort:    cfg.MikrotikPort,
		WaitTimeout: envDuration("MIKROTIK_VM_WAIT", 90*time.Second),
	}
}

func (c VMConfig) SkipReason() string {
	return ""
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}

func envDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
