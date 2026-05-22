package mikrotik

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type FakeClient struct {
	mu       sync.Mutex
	users    map[string]*User
	profiles map[string][]UserProfile
	nextID   int
	profilesDefined map[string]bool
}

func NewFake() *FakeClient {
	f := &FakeClient{
		users:    make(map[string]*User),
		profiles: make(map[string][]UserProfile),
		profilesDefined: map[string]bool{
			"profile-open-2M-30d": true,
		},
		nextID: 1,
	}
	return f
}

func (f *FakeClient) Ping() error { return nil }

func (f *FakeClient) InvalidateCache() {}

func (f *FakeClient) ListUsers() ([]User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []User
	for _, u := range f.users {
		out = append(out, *u)
	}
	return out, nil
}

func (f *FakeClient) GetUser(name string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[name]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *FakeClient) AddUser(name, password, comment string, sharedUsers int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[name]; ok {
		return ErrNameTaken
	}
	id := fmt.Sprintf("*%d", f.nextID)
	f.nextID++
	f.users[name] = &User{ID: id, Name: name, Password: password, SharedUsers: sharedUsers, Comment: comment}
	return nil
}

func (f *FakeClient) SetUser(name string, password *string, sharedUsers *int, comment string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[name]
	if !ok {
		return ErrNotFound
	}
	if password != nil {
		u.Password = *password
	}
	if sharedUsers != nil {
		u.SharedUsers = *sharedUsers
	}
	u.Comment = comment
	return nil
}

func (f *FakeClient) RemoveUser(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[name]; !ok {
		return ErrNotFound
	}
	delete(f.users, name)
	delete(f.profiles, name)
	return nil
}

func (f *FakeClient) ListUserProfiles(user string) ([]UserProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]UserProfile(nil), f.profiles[user]...), nil
}

func (f *FakeClient) AddUserProfile(user, profile string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.profilesDefined[profile] {
		return ErrProfileMissing
	}
	if _, ok := f.users[user]; !ok {
		return ErrNotFound
	}
	id := fmt.Sprintf("*p%d", f.nextID)
	f.nextID++
	end := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	p := UserProfile{ID: id, User: user, Profile: profile, State: "active", EndTime: end}
	// deactivate previous active
	for i := range f.profiles[user] {
		if strings.Contains(strings.ToLower(f.profiles[user][i].State), "active") {
			f.profiles[user][i].State = "expired"
		}
	}
	f.profiles[user] = append(f.profiles[user], p)
	return nil
}

func (f *FakeClient) RemoveUserProfile(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for user, list := range f.profiles {
		for i, p := range list {
			if p.ID == id {
				if strings.Contains(strings.ToLower(p.State), "active") {
					return fmt.Errorf("cannot remove active profile")
				}
				f.profiles[user] = append(list[:i], list[i+1:]...)
				return nil
			}
		}
	}
	return ErrNotFound
}

