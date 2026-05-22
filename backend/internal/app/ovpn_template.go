package app

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed templates/client-template.ovpn
var embeddedOvpnTemplate string

func (a *App) readOvpnTemplate() ([]byte, error) {
	if a.Cfg.OpenVPNTemplatePath != "" {
		for _, p := range ovpnTemplateCandidates(a.Cfg.OpenVPNTemplatePath) {
			b, err := os.ReadFile(p)
			if err == nil {
				return b, nil
			}
		}
	}
	if embeddedOvpnTemplate != "" {
		return []byte(embeddedOvpnTemplate), nil
	}
	return nil, os.ErrNotExist
}

func ovpnTemplateCandidates(path string) []string {
	if filepath.IsAbs(path) {
		return []string{path}
	}
	var out []string
	out = append(out, path)
	// go test often runs with cwd = package dir or module root
	for _, base := range []string{".", "..", "../..", "../../.."} {
		out = append(out, filepath.Join(base, path))
	}
	// module root: backend/../config/...
	out = append(out, filepath.Join("..", "config", "client-template.ovpn"))
	return out
}
