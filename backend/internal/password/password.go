package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const vpnGenChars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const vpnGenLen = 8

var vpnPasswordRe = regexp.MustCompile(`[A-Za-z]`)
var vpnDigitRe = regexp.MustCompile(`[0-9]`)

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func Check(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// GenerateVPN returns a random password that satisfies ValidVPN.
func GenerateVPN() (string, error) {
	for range 8 {
		pw, err := randomVPNCandidate()
		if err != nil {
			return "", err
		}
		if ValidVPN(pw) {
			return pw, nil
		}
	}
	return "", fmt.Errorf("could not generate VPN password")
}

func randomVPNCandidate() (string, error) {
	b := make([]byte, vpnGenLen)
	max := big.NewInt(int64(len(vpnGenChars)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = vpnGenChars[n.Int64()]
	}
	return string(b), nil
}

func ValidVPN(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	if !vpnPasswordRe.MatchString(pw) || !vpnDigitRe.MatchString(pw) {
		return false
	}
	return true
}

func ValidPanel(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	var letter, digit bool
	for _, r := range pw {
		if unicode.IsLetter(r) {
			letter = true
		}
		if unicode.IsDigit(r) {
			digit = true
		}
	}
	return letter && digit
}
