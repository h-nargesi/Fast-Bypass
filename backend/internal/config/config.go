package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr     string
	CORSOrigins  []string
	SQLitePath   string
	TZ           string

	MikrotikCacheTTL time.Duration
	MikrotikFake     bool
	MikrotikHost     string
	MikrotikPort     int
	MikrotikUser     string
	MikrotikPass     string
	MikrotikTLSInsec bool
	MikrotikTimeout  time.Duration

	JWTSecret      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration

	AdminUsername string
	AdminPassword string

	DefaultProfile         string
	UsernamePrefixSep      string
	UsernameLocalMaxLen    int
	SharedUsersMax         int

	OpenVPNDownloadURL   string
	OpenVPNKeyPassword   string
	L2TPIPsecSecret      string
	L2TPServer           string
	OpenVPNTemplatePath  string
}

func Load() Config {
	godotenvLoad()
	return Config{
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
		CORSOrigins: strings.Split(env("CORS_ORIGINS", "http://localhost:4200"), ","),
		SQLitePath:  env("SQLITE_PATH", "./data/panel.db"),
		TZ:          env("TZ", "Asia/Tehran"),

		MikrotikCacheTTL: envDuration("MIKROTIK_CACHE_TTL", 30*time.Second),
		MikrotikFake:     envBool("MIKROTIK_FAKE", true),
		MikrotikHost:     env("MIKROTIK_HOST", "192.168.88.1"),
		MikrotikPort:     envInt("MIKROTIK_PORT", 8729),
		MikrotikUser:     env("MIKROTIK_USERNAME", "api-panel"),
		MikrotikPass:     env("MIKROTIK_PASSWORD", ""),
		MikrotikTLSInsec: envBool("MIKROTIK_TLS_INSECURE", false),
		MikrotikTimeout:  envDuration("MIKROTIK_TIMEOUT", 10*time.Second),

		JWTSecret:     env("JWT_SECRET", "change-me"),
		JWTAccessTTL:  envDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: envDuration("JWT_REFRESH_TTL", 168*time.Hour),

		AdminUsername: env("ADMIN_USERNAME", "admin"),
		AdminPassword: env("ADMIN_PASSWORD", "change-me"),

		DefaultProfile:      env("DEFAULT_PROFILE", "profile-open-2M-30d"),
		UsernamePrefixSep:   env("USERNAME_PREFIX_SEPARATOR", "_"),
		UsernameLocalMaxLen: envInt("USERNAME_LOCAL_MAX_LEN", 24),
		SharedUsersMax:      envInt("SHARED_USERS_MAX", 20),

		OpenVPNDownloadURL:  env("OPENVPN_DOWNLOAD_URL", "http://dl.nimbaha.info/dl/"),
		OpenVPNKeyPassword:  env("OPENVPN_KEY_PASSWORD", ""),
		L2TPIPsecSecret:     env("L2TP_IPSEC_SECRET", ""),
		L2TPServer:          env("L2TP_SERVER", ""),
		OpenVPNTemplatePath: env("OPENVPN_TEMPLATE_PATH", "./config/client-template.ovpn"),
	}
}

func godotenvLoad() {
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			return
		}
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
