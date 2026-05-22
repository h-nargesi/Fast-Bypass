package auth

import (
	"context"
	"net/http"
	"strings"

	"fast-bypass/internal/httpx"
)

type ctxKey int

const claimsKey ctxKey = 1

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

func Middleware(issuer *Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "توکن احراز هویت نیاز است")
				return
			}
			c, err := issuer.Parse(strings.TrimPrefix(h, "Bearer "))
			if err != nil || c.TokenType != "access" {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "توکن نامعتبر است")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), c)))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := ClaimsFrom(r.Context())
		if !ok || c.Role != RoleAdmin {
			httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "دسترسی ادمین لازم است")
			return
		}
		next.ServeHTTP(w, r)
	})
}
