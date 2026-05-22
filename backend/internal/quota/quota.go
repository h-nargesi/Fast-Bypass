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

func UsedForManager(reg owner.Registry, managerID int64, users []mikrotik.User, profilesFn func(string) ([]mikrotik.UserProfile, error), now time.Time) (int, error) {
	used := 0
	for _, u := range users {
		if reg.Resolve(u.Name, u.Comment) != managerID {
			continue
		}
		profs, err := profilesFn(u.Name)
		if err != nil {
			return 0, err
		}
		active := false
		for _, p := range profs {
			if ProfileActive(p, now) {
				active = true
				break
			}
		}
		if active {
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
