package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const certKeyChars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const certKeyLen = 10

var certTitleRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{2,31}$`)

// GenerateCertKeyPass returns a random passphrase for OpenVPN key export (min 8 chars).
func GenerateCertKeyPass() (string, error) {
	for range 8 {
		pw, err := randomFromCharset(certKeyChars, certKeyLen)
		if err != nil {
			return "", err
		}
		if len(pw) >= 8 {
			return pw, nil
		}
	}
	return "", fmt.Errorf("could not generate certificate key passphrase")
}

// ValidCertTitle checks TITLE argument for generate-certificate script.
func ValidCertTitle(title string) bool {
	title = strings.TrimSpace(title)
	return certTitleRe.MatchString(title)
}

func randomFromCharset(charset string, n int) (string, error) {
	b := make([]byte, n)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[v.Int64()]
	}
	return string(b), nil
}
