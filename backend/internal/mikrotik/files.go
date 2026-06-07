package mikrotik

import (
	"fmt"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
)

func templateOvpnName(certTitle string) string {
	return "open-vpns/config-" + certTitle + ".ovpn"
}

// ReadFileContents returns RouterOS /file contents for a file name.
func (r *RouterOS) ReadFileContents(name string) ([]byte, error) {
	reply, err := r.run("/file/print", "?name="+name)
	if err != nil {
		return nil, err
	}
	for _, s := range reply.Re {
		if field(s, "name") == name {
			c := field(s, "contents")
			if c == "" {
				return nil, ErrNotFound
			}
			return []byte(c), nil
		}
	}
	return nil, ErrNotFound
}

// WriteFileContents creates or replaces a file on the router.
func (r *RouterOS) WriteFileContents(name string, body []byte) error {
	contents := string(body)
	if id, err := r.fileID(name); err == nil {
		_, err = r.run("/file/set", "=numbers="+id, "=contents="+contents)
		return err
	}
	_, err := r.run("/file/add", "=name="+name, "=type=file", "=contents="+contents)
	return err
}

func (r *RouterOS) fileID(name string) (string, error) {
	reply, err := r.run("/file/print", "?name="+name)
	if err != nil {
		return "", err
	}
	for _, s := range reply.Re {
		if field(s, "name") == name {
			id := field(s, ".id")
			if id != "" {
				return id, nil
			}
		}
	}
	return "", ErrNotFound
}

// GenerateCertificate runs the RouterOS generate-certificate script with globals set via import.
func (r *RouterOS) GenerateCertificate(scriptName, title, passphrase string) error {
	if scriptName == "" {
		scriptName = "generate-certificate"
	}
	escapedTitle := rosQuote(title)
	escapedPass := rosQuote(passphrase)
	source := fmt.Sprintf(
		`:global TITLE %s; :global PASSPHRASE %s; /system script run [find name=%s]`,
		escapedTitle, escapedPass, scriptName,
	)
	temp := fmt.Sprintf("panel-cert-%s.rsc", strings.ReplaceAll(title, "/", "-"))
	if err := r.WriteFileContents(temp, []byte(source)); err != nil {
		return err
	}
	defer func() { _ = r.removeFile(temp) }()
	_, err := r.run("/import", "=file-name="+temp)
	return err
}

func (r *RouterOS) removeFile(name string) error {
	id, err := r.fileID(name)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return err
	}
	_, err = r.run("/file/remove", "=numbers="+id)
	return err
}

func rosQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// FileExists reports whether a named file exists on the router.
func (r *RouterOS) FileExists(name string) (bool, error) {
	_, err := r.fileID(name)
	if err == ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

// sentence helper for tests
func fileFromSentence(s *proto.Sentence) (name, contents string) {
	return field(s, "name"), field(s, "contents")
}
