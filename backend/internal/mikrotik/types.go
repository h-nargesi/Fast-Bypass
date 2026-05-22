package mikrotik

import "time"

type User struct {
	ID          string
	Name        string
	Password    string
	SharedUsers int
	Comment     string
}

type UserProfile struct {
	ID       string
	User     string
	Profile  string
	State    string
	EndTime  string
}

type Client interface {
	Ping() error
	ListUsers() ([]User, error)
	GetUser(name string) (*User, error)
	AddUser(name, password, comment string, sharedUsers int) error
	SetUser(name string, password *string, sharedUsers *int, comment string) error
	RemoveUser(name string) error
	ListUserProfiles(user string) ([]UserProfile, error)
	AddUserProfile(user, profile string) error
	RemoveUserProfile(id string) error
	InvalidateCache()
}

type CachedClient struct {
	inner Client
	ttl   time.Duration
	usersAt time.Time
	users   []User
	profilesAt time.Time
	profiles map[string][]UserProfile
}

func NewCached(inner Client, ttl time.Duration) *CachedClient {
	return &CachedClient{inner: inner, ttl: ttl, profiles: make(map[string][]UserProfile)}
}

func (c *CachedClient) Ping() error { return c.inner.Ping() }

func (c *CachedClient) ListUsers() ([]User, error) {
	if time.Since(c.usersAt) < c.ttl && c.users != nil {
		return c.users, nil
	}
	u, err := c.inner.ListUsers()
	if err != nil {
		return nil, err
	}
	c.users = u
	c.usersAt = time.Now()
	return u, nil
}

func (c *CachedClient) forceUsers() ([]User, error) {
	c.users = nil
	return c.ListUsers()
}

func (c *CachedClient) GetUser(name string) (*User, error) {
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Name == name {
			return &u, nil
		}
	}
	return nil, ErrNotFound
}

func (c *CachedClient) AddUser(name, password, comment string, sharedUsers int) error {
	if err := c.inner.AddUser(name, password, comment, sharedUsers); err != nil {
		return err
	}
	c.InvalidateCache()
	return nil
}

func (c *CachedClient) SetUser(name string, password *string, sharedUsers *int, comment string) error {
	if err := c.inner.SetUser(name, password, sharedUsers, comment); err != nil {
		return err
	}
	c.InvalidateCache()
	return nil
}

func (c *CachedClient) RemoveUser(name string) error {
	if err := c.inner.RemoveUser(name); err != nil {
		return err
	}
	c.InvalidateCache()
	return nil
}

func (c *CachedClient) ListUserProfiles(user string) ([]UserProfile, error) {
	if p, ok := c.profiles[user]; ok && time.Since(c.profilesAt) < c.ttl {
		return p, nil
	}
	p, err := c.inner.ListUserProfiles(user)
	if err != nil {
		return nil, err
	}
	c.profiles[user] = p
	c.profilesAt = time.Now()
	return p, nil
}

func (c *CachedClient) AddUserProfile(user, profile string) error {
	if err := c.inner.AddUserProfile(user, profile); err != nil {
		return err
	}
	c.InvalidateCache()
	return nil
}

func (c *CachedClient) RemoveUserProfile(id string) error {
	if err := c.inner.RemoveUserProfile(id); err != nil {
		return err
	}
	c.InvalidateCache()
	return nil
}

func (c *CachedClient) InvalidateCache() {
	c.users = nil
	c.profiles = make(map[string][]UserProfile)
	c.inner.InvalidateCache()
}

func (c *CachedClient) RefreshListUsers() ([]User, error) {
	c.users = nil
	return c.forceUsers()
}
