package mikrotik

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
)

// RouterOS talks to User Manager over RouterOS API (api on 8728 or api-ssl on 8729).
type RouterOS struct {
	mu      sync.Mutex
	addr    string
	user    string
	pass    string
	useTLS  bool
	tls     *tls.Config
	timeout time.Duration
	cli     *routeros.Client
}

func NewRouterOS(addr, user, pass string, useTLS bool, tlsCfg *tls.Config, timeout time.Duration) *RouterOS {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if useTLS && tlsCfg == nil {
		tlsCfg = &tls.Config{}
	}
	return &RouterOS{addr: addr, user: user, pass: pass, useTLS: useTLS, tls: tlsCfg, timeout: timeout}
}

func (r *RouterOS) InvalidateCache() {}

func (r *RouterOS) Ping() error {
	_, err := r.run("/system/identity/print")
	return err
}

func (r *RouterOS) ListUsers() ([]User, error) {
	reply, err := r.run("/user-manager/user/print")
	if err != nil {
		return nil, err
	}
	var out []User
	for _, s := range reply.Re {
		if u := userFromSentence(s); u.Name != "" {
			out = append(out, u)
		}
	}
	return out, nil
}

func (r *RouterOS) GetUser(name string) (*User, error) {
	users, err := r.ListUsers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].Name == name {
			return &users[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *RouterOS) AddUser(name, password, comment string, sharedUsers int, disabled bool) error {
	args := []string{
		"/user-manager/user/add",
		"=name=" + name,
		"=password=" + password,
		fmt.Sprintf("=shared-users=%d", sharedUsers),
		"=disabled=" + disabledYesNo(disabled),
	}
	if comment != "" {
		args = append(args, "=comment="+comment)
	}
	_, err := r.run(args...)
	return mapDeviceErr(err)
}

func (r *RouterOS) SetUser(name string, password *string, sharedUsers *int, comment string, disabled *bool) error {
	u, err := r.GetUser(name)
	if err != nil {
		return err
	}
	args := []string{"/user-manager/user/set", "=numbers=" + u.ID}
	if password != nil {
		args = append(args, "=password="+*password)
	}
	if sharedUsers != nil {
		args = append(args, fmt.Sprintf("=shared-users=%d", *sharedUsers))
	}
	if comment != "" {
		args = append(args, "=comment="+comment)
	}
	if disabled != nil {
		args = append(args, "=disabled="+disabledYesNo(*disabled))
	}
	_, err = r.run(args...)
	return mapDeviceErr(err)
}

func disabledYesNo(disabled bool) string {
	if disabled {
		return "yes"
	}
	return "no"
}

func (r *RouterOS) RemoveUser(name string) error {
	u, err := r.GetUser(name)
	if err != nil {
		return err
	}
	_, err = r.run("/user-manager/user/remove", "=numbers="+u.ID)
	return mapDeviceErr(err)
}

func (r *RouterOS) ListUserProfiles(user string) ([]UserProfile, error) {
	reply, err := r.run("/user-manager/user-profile/print", "?user="+user)
	if err != nil {
		return nil, err
	}
	var out []UserProfile
	for _, s := range reply.Re {
		if p := profileFromSentence(s); p.User != "" || p.ID != "" {
			if p.User == "" {
				p.User = user
			}
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *RouterOS) AddUserProfile(user, profile string) error {
	_, err := r.run(
		"/user-manager/user-profile/add",
		"=user="+user,
		"=profile="+profile,
	)
	return mapDeviceErr(err)
}

func (r *RouterOS) RemoveUserProfile(id string) error {
	_, err := r.run("/user-manager/user-profile/remove", "=numbers="+id)
	return mapDeviceErr(err)
}

func (r *RouterOS) run(sentences ...string) (*routeros.Reply, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	if r.cli == nil {
		var (
			cli *routeros.Client
			err error
		)
		if r.useTLS {
			cli, err = routeros.DialTLSContext(ctx, r.addr, r.user, r.pass, r.tls)
		} else {
			cli, err = routeros.DialContext(ctx, r.addr, r.user, r.pass)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		r.cli = cli
	}

	reply, err := r.cli.RunContext(ctx, sentences...)
	if err != nil {
		if isConnErr(err) {
			_ = r.cli.Close()
			r.cli = nil
		}
		return nil, mapRunErr(err)
	}
	return reply, nil
}

func mapRunErr(err error) error {
	if err == nil {
		return nil
	}
	var dev *routeros.DeviceError
	if errors.As(err, &dev) {
		return mapDeviceErr(dev)
	}
	if isConnErr(err) {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return err
}

func mapDeviceErr(err error) error {
	if err == nil {
		return nil
	}
	var dev *routeros.DeviceError
	if !errors.As(err, &dev) {
		return err
	}
	msg := strings.ToLower(dev.Error())
	switch {
	case strings.Contains(msg, "already") || strings.Contains(msg, "exists"):
		return ErrNameTaken
	case strings.Contains(msg, "not found") || strings.Contains(msg, "no such"):
		return ErrNotFound
	case strings.Contains(msg, "profile") && (strings.Contains(msg, "not") || strings.Contains(msg, "missing")):
		return ErrProfileMissing
	case strings.Contains(msg, "active"):
		return err
	default:
		return err
	}
}

func isConnErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection") ||
		strings.Contains(s, "handshake") ||
		strings.Contains(s, "EOF") ||
		strings.Contains(s, "reset") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "closed")
}

func userFromSentence(s *proto.Sentence) User {
	return User{
		ID:          field(s, ".id"),
		Name:        field(s, "name"),
		Password:    field(s, "password"),
		SharedUsers: intField(s, "shared-users"),
		Comment:     field(s, "comment"),
		Disabled:    fieldYes(field(s, "disabled")),
	}
}

func fieldYes(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes", "true", "1":
		return true
	default:
		return false
	}
}

func profileFromSentence(s *proto.Sentence) UserProfile {
	return UserProfile{
		ID:      field(s, ".id"),
		User:    field(s, "user"),
		Profile: field(s, "profile"),
		State:   field(s, "state"),
		EndTime: field(s, "end-time"),
	}
}

func field(s *proto.Sentence, key string) string {
	if s == nil || s.Map == nil {
		return ""
	}
	if v := s.Map[key]; v != "" {
		return v
	}
	return s.Map["="+key]
}

func intField(s *proto.Sentence, key string) int {
	n, _ := strconv.Atoi(field(s, key))
	return n
}
