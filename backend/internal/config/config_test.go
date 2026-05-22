package config

import "testing"

func TestMikrotikAPIFromEnv(t *testing.T) {
	t.Setenv("MIKROTIK_PORT", "")
	tests := []struct {
		api      string
		wantAPI  string
		wantTLS  bool
		wantPort int
	}{
		{"api", "api", false, 8728},
		{"plain", "api", false, 8728},
		{"api-ssl", "api-ssl", true, 8729},
		{"ssl", "api-ssl", true, 8729},
		{"", "api-ssl", true, 8729},
	}
	for _, tc := range tests {
		t.Run(tc.api, func(t *testing.T) {
			t.Setenv("MIKROTIK_API", tc.api)
			gotAPI, gotTLS, gotPort := mikrotikAPIFromEnv()
			if gotAPI != tc.wantAPI || gotTLS != tc.wantTLS || gotPort != tc.wantPort {
				t.Fatalf("api=%q tls=%v port=%d; want api=%q tls=%v port=%d",
					gotAPI, gotTLS, gotPort, tc.wantAPI, tc.wantTLS, tc.wantPort)
			}
		})
	}
}

func TestMikrotikAPIFromEnv_portOnly8728(t *testing.T) {
	t.Setenv("MIKROTIK_API", "")
	t.Setenv("MIKROTIK_PORT", "8728")
	gotAPI, gotTLS, gotPort := mikrotikAPIFromEnv()
	if gotAPI != "api" || gotTLS || gotPort != 8728 {
		t.Fatalf("got api=%q tls=%v port=%d", gotAPI, gotTLS, gotPort)
	}
}

func TestMikrotikAPIFromEnv_portOnly8729(t *testing.T) {
	t.Setenv("MIKROTIK_API", "")
	t.Setenv("MIKROTIK_PORT", "8729")
	gotAPI, gotTLS, gotPort := mikrotikAPIFromEnv()
	if gotAPI != "api-ssl" || !gotTLS || gotPort != 8729 {
		t.Fatalf("got api=%q tls=%v port=%d", gotAPI, gotTLS, gotPort)
	}
}

func TestLoad_mikrotikAPI(t *testing.T) {
	t.Setenv("MIKROTIK_API", "api")
	t.Setenv("MIKROTIK_PORT", "")
	cfg := Load()
	if cfg.MikrotikAPI != "api" || cfg.MikrotikUseTLS || cfg.MikrotikPort != 8728 {
		t.Fatalf("Load: api=%q tls=%v port=%d", cfg.MikrotikAPI, cfg.MikrotikUseTLS, cfg.MikrotikPort)
	}
}
