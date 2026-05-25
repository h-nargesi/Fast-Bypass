package quota

import (
	"strings"
	"time"

	"fast-bypass/internal/mikrotik"
	"fast-bypass/internal/owner"
)

func ProfileActive(p mikrotik.UserProfile, now time.Time) bool {
	st := strings.ToLower(strings.TrimSpace(p.State))
	if strings.Contains(st, "active") {
		return true
	}
	if p.EndTime == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, p.EndTime)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", p.EndTime)
	}
	if err != nil {
		return false
	}
	return t.After(now)
}

// ScopeStats counts active VPN users and their concurrent connection slots (shared_users).
type ScopeStats struct {
	VPNUsers    int // users with active profile, not disabled on router
	Connections int // sum of shared_users for those users
}

func userHasActiveProfile(profs []mikrotik.UserProfile, now time.Time) bool {
	for _, p := range profs {
		if ProfileActive(p, now) {
			return true
		}
	}
	return false
}

// AggregateByOwner totals active VPN users and connection slots per owner (manager ID or orphan).
func AggregateByOwner(reg owner.Registry, users []mikrotik.User, profilesFn func(string) ([]mikrotik.UserProfile, error), now time.Time) (total ScopeStats, orphan ScopeStats, byManager map[int64]ScopeStats, err error) {
	byManager = make(map[int64]ScopeStats)
	for _, u := range users {
		ownerID := reg.Resolve(u.Name, u.Comment)
		if u.Disabled {
			continue
		}
		profs, perr := profilesFn(u.Name)
		if perr != nil {
			return ScopeStats{}, ScopeStats{}, nil, perr
		}
		if !userHasActiveProfile(profs, now) {
			continue
		}
		total.VPNUsers++
		total.Connections += u.SharedUsers
		if ownerID == 0 {
			orphan.VPNUsers++
			orphan.Connections += u.SharedUsers
			continue
		}
		s := byManager[ownerID]
		s.VPNUsers++
		s.Connections += u.SharedUsers
		byManager[ownerID] = s
	}
	return total, orphan, byManager, nil
}

func UsedForManager(reg owner.Registry, managerID int64, users []mikrotik.User, profilesFn func(string) ([]mikrotik.UserProfile, error), now time.Time) (int, error) {
	used := 0
	for _, u := range users {
		if reg.Resolve(u.Name, u.Comment) != managerID || u.Disabled {
			continue
		}
		profs, err := profilesFn(u.Name)
		if err != nil {
			return 0, err
		}
		if userHasActiveProfile(profs, now) {
			used += u.SharedUsers
		}
	}
	return used, nil
}

func CheckAdd(used, quota, newShared int) bool {
	return used+newShared <= quota
}

func CheckIncrease(used, quota, oldShared, newShared int) bool {
	return used-oldShared+newShared <= quota
}
