package quota

import (
	"testing"
	"time"

	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/owner"
)

func TestProfileActive_byState(t *testing.T) {
	now := time.Now()
	if !ProfileActive(mikrotik.UserProfile{State: "active"}, now) {
		t.Fatal("state active should be active")
	}
	if ProfileActive(mikrotik.UserProfile{State: "expired"}, now) {
		t.Fatal("expired state without future end-time should be inactive")
	}
}

func TestProfileActive_byEndTime(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour).Format(time.RFC3339)
	past := now.Add(-24 * time.Hour).Format(time.RFC3339)
	if !ProfileActive(mikrotik.UserProfile{State: "", EndTime: future}, now) {
		t.Fatal("future end-time should be active")
	}
	if ProfileActive(mikrotik.UserProfile{State: "", EndTime: past}, now) {
		t.Fatal("past end-time should be inactive")
	}
}

func TestUsedForManager_skipsDisabled(t *testing.T) {
	reg := owner.Registry{Separator: "-", Managers: []owner.ManagerInfo{{ID: 1, Slug: "ali"}}}
	now := time.Now()
	users := []mikrotik.User{
		{Name: "ali-a", Comment: "panel=ali", SharedUsers: 5, Disabled: true},
		{Name: "ali-b", Comment: "panel=ali", SharedUsers: 2, Disabled: false},
	}
	profs := map[string][]mikrotik.UserProfile{
		"ali-a": {{State: "active", EndTime: now.Add(time.Hour).Format(time.RFC3339)}},
		"ali-b": {{State: "active", EndTime: now.Add(time.Hour).Format(time.RFC3339)}},
	}
	used, err := UsedForManager(reg, 1, users, func(name string) ([]mikrotik.UserProfile, error) {
		return profs[name], nil
	}, now)
	if err != nil || used != 2 {
		t.Fatalf("used = %d, want 2 (disabled user excluded)", used)
	}
}

func TestUsedForManager(t *testing.T) {
	reg := owner.Registry{
		Separator: "-",
		Managers:  []owner.ManagerInfo{{ID: 1, Slug: "ali"}},
	}
	now := time.Now()
	users := []mikrotik.User{
		{Name: "ali-a", Comment: "panel=ali", SharedUsers: 2},
		{Name: "ali-b", Comment: "panel=ali", SharedUsers: 3},
		{Name: "ali-c", Comment: "panel=ali", SharedUsers: 5},
	}
	profiles := map[string][]mikrotik.UserProfile{
		"ali-a": {{State: "active", EndTime: now.Add(time.Hour).Format(time.RFC3339)}},
		"ali-b": {{State: "active", EndTime: now.Add(time.Hour).Format(time.RFC3339)}},
		"ali-c": {{State: "expired", EndTime: now.Add(-time.Hour).Format(time.RFC3339)}},
	}
	used, err := UsedForManager(reg, 1, users, func(name string) ([]mikrotik.UserProfile, error) {
		return profiles[name], nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if used != 5 {
		t.Fatalf("used = %d, want 5 (2+3 active only)", used)
	}
}

func TestCheckAdd_and_CheckIncrease(t *testing.T) {
	if !CheckAdd(5, 10, 3) {
		t.Fatal("5+3 <= 10 should pass")
	}
	if CheckAdd(5, 10, 6) {
		t.Fatal("5+6 > 10 should fail")
	}
	if !CheckIncrease(5, 10, 2, 4) {
		t.Fatal("5-2+4 <= 10 should pass")
	}
	if CheckIncrease(5, 10, 2, 8) {
		t.Fatal("5-2+8 > 10 should fail")
	}
}

func TestAggregateByOwner(t *testing.T) {
	reg := owner.Registry{
		Separator: "-",
		Managers: []owner.ManagerInfo{
			{ID: 1, Slug: "ali"},
			{ID: 2, Slug: "bob"},
		},
	}
	now := time.Now()
	future := now.Add(time.Hour).Format(time.RFC3339)
	past := now.Add(-time.Hour).Format(time.RFC3339)
	users := []mikrotik.User{
		{Name: "ali-a", Comment: "panel:ali", SharedUsers: 2},
		{Name: "ali-b", Comment: "panel:ali", SharedUsers: 3, Disabled: true},
		{Name: "ali-c", Comment: "panel:ali", SharedUsers: 5},
		{Name: "bob-x", Comment: "panel:bob", SharedUsers: 4},
		{Name: "orphan1", Comment: "", SharedUsers: 1},
	}
	profiles := map[string][]mikrotik.UserProfile{
		"ali-a": {{State: "active", EndTime: future}},
		"ali-b": {{State: "active", EndTime: future}},
		"ali-c": {{State: "expired", EndTime: past}},
		"bob-x": {{State: "active", EndTime: future}},
		"orphan1": {{State: "active", EndTime: future}},
	}
	total, orphan, byMgr, err := AggregateByOwner(reg, users, func(name string) ([]mikrotik.UserProfile, error) {
		return profiles[name], nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if total.VPNUsers != 3 || total.Connections != 7 {
		t.Fatalf("total = %+v, want 3 active users and 7 connections (2+4+1)", total)
	}
	if orphan.VPNUsers != 1 || orphan.Connections != 1 {
		t.Fatalf("orphan = %+v", orphan)
	}
	if byMgr[1].VPNUsers != 1 || byMgr[1].Connections != 2 {
		t.Fatalf("ali = %+v, want 1 active user and 2 connections", byMgr[1])
	}
	if byMgr[2].VPNUsers != 1 || byMgr[2].Connections != 4 {
		t.Fatalf("bob = %+v", byMgr[2])
	}
}
