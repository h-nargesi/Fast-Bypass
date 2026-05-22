package password

import (
	"regexp"
	// "unicode"

	"golang.org/x/crypto/bcrypt"
)

var vpnPasswordRe = regexp.MustCompile(`[A-Za-z]`)
var vpnDigitRe = regexp.MustCompile(`[0-9]`)

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func Check(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
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
	if len(pw) < 3 {
		return false
	}
	return true
	// if len(pw) < 8 {
	// 	return false
	// }
	// var letter, digit bool
	// for _, r := range pw {
	// 	if unicode.IsLetter(r) {
	// 		letter = true
	// 	}
	// 	if unicode.IsDigit(r) {
	// 		digit = true
	// 	}
	// }
	// return letter && digit
}
