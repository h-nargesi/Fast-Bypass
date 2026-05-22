package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
)

type Claims struct {
	jwt.RegisteredClaims
	Role       Role   `json:"role"`
	ManagerID  int64  `json:"manager_id,omitempty"`
	Slug       string `json:"slug,omitempty"`
	Username   string `json:"username"`
	TokenType  string `json:"typ"` // access | refresh
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Issuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewIssuer(secret string, accessTTL, refreshTTL time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (i *Issuer) NewPair(c Claims) (TokenPair, error) {
	access, err := i.sign(c, i.accessTTL, "access")
	if err != nil {
		return TokenPair{}, err
	}
	c.TokenType = "refresh"
	refresh, err := i.sign(c, i.refreshTTL, "refresh")
	return TokenPair{AccessToken: access, RefreshToken: refresh}, err
}

func (i *Issuer) sign(c Claims, ttl time.Duration, typ string) (string, error) {
	now := time.Now()
	c.TokenType = typ
	c.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   c.Username,
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(i.secret)
}

func (i *Issuer) Parse(token string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return i.secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}
