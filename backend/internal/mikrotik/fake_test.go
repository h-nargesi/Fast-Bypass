package mikrotik

import "testing"

func TestFakeClient_lifecycle(t *testing.T) {
	c := NewFake()
	if err := c.AddUser("ali-test1", "Secret123", "panel:ali", 2); err != nil {
		t.Fatal(err)
	}
	u, err := c.GetUser("ali-test1")
	if err != nil || u.SharedUsers != 2 || u.Comment != "panel:ali" {
		t.Fatalf("GetUser: %+v err=%v", u, err)
	}
	if err := c.AddUser("ali-test1", "x", "", 1); err != ErrNameTaken {
		t.Fatalf("duplicate: got %v", err)
	}
	if err := c.AddUserProfile("ali-test1", "profile-open-2M-30d"); err != nil {
		t.Fatal(err)
	}
	profs, err := c.ListUserProfiles("ali-test1")
	if err != nil || len(profs) != 1 || profs[0].State != "active" {
		t.Fatalf("profiles: %+v err=%v", profs, err)
	}
	if err := c.RemoveUserProfile(profs[0].ID); err == nil {
		t.Fatal("should not remove active profile")
	}
	if err := c.RemoveUser("ali-test1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetUser("ali-test1"); err != ErrNotFound {
		t.Fatalf("after remove: %v", err)
	}
}

func TestFakeClient_unknownProfile(t *testing.T) {
	c := NewFake()
	_ = c.AddUser("u1", "Secret123", "", 1)
	if err := c.AddUserProfile("u1", "no-such-profile"); err != ErrProfileMissing {
		t.Fatalf("got %v", err)
	}
}
