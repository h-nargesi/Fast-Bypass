package mikrotik

import (
	"testing"
	"time"
)

func TestCachedClient_invalidateOnWrite(t *testing.T) {
	fake := NewFake()
	c := NewCached(fake, time.Hour)
	_ = c.AddUser("u1", "Secret123", "panel=ali", 1, false)
	list1, _ := c.ListUsers()
	if len(list1) != 1 {
		t.Fatalf("list1: %d", len(list1))
	}
	_ = fake.AddUser("u2", "Secret123", "panel=ali", 1, false)
	listStale, _ := c.ListUsers()
	if len(listStale) != 1 {
		t.Fatalf("expected cached stale list len=1, got %d", len(listStale))
	}
	c.InvalidateCache()
	listFresh, _ := c.ListUsers()
	if len(listFresh) != 2 {
		t.Fatalf("after invalidate: %d", len(listFresh))
	}
}

func TestCachedClient_refreshListUsers(t *testing.T) {
	fake := NewFake()
	c := NewCached(fake, time.Hour)
	_ = c.AddUser("u1", "Secret123", "", 1, false)
	_ = fake.AddUser("u2", "Secret123", "", 1, false)
	refreshed, err := c.RefreshListUsers()
	if err != nil || len(refreshed) != 2 {
		t.Fatalf("refresh: %+v err=%v", refreshed, err)
	}
}
