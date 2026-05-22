package auth

import (
	"testing"
	"time"
)

func TestIssuer_pair_and_parse(t *testing.T) {
	iss := NewIssuer("test-secret", time.Minute, time.Hour)
	pair, err := iss.NewPair(Claims{Role: RoleManager, ManagerID: 1, Slug: "ali", Username: "ali"})
	if err != nil {
		t.Fatal(err)
	}
	access, err := iss.Parse(pair.AccessToken)
	if err != nil || access.TokenType != "access" || access.ManagerID != 1 {
		t.Fatalf("access: %+v err=%v", access, err)
	}
	refresh, err := iss.Parse(pair.RefreshToken)
	if err != nil || refresh.TokenType != "refresh" {
		t.Fatalf("refresh: %+v err=%v", refresh, err)
	}
}
