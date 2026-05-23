package app

import (
	"context"

	"fast-bypass/internal/store"
)

func (a *App) mikrotikSharedUsersByName() (map[string]int, error) {
	users, err := a.MT.ListUsers()
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(users))
	for _, u := range users {
		m[u.Name] = u.SharedUsers
	}
	return m, nil
}

func liveSharedUsers(mtMap map[string]int, mikrotikName string, dbFallback int) int {
	if v, ok := mtMap[mikrotikName]; ok {
		return v
	}
	return dbFallback
}

// latestUnsettledPerUser maps mikrotik_name → newest unsettled activation id.
// rows must be ordered by created_at DESC (as in ListRenewalSharedUsersInScope).
func latestUnsettledPerUser(rows []store.RenewalSharedUsersRow) map[string]int64 {
	out := make(map[string]int64)
	for _, r := range rows {
		if r.IsSettled {
			continue
		}
		if _, ok := out[r.MikrotikName]; !ok {
			out[r.MikrotikName] = r.ID
		}
	}
	return out
}

func (a *App) syncSharedUsersToUnsettledActivation(ctx context.Context, metaID int64, sharedUsers int) {
	_ = a.Store.UpdateLatestUnsettledActivationSharedUsers(ctx, metaID, sharedUsers)
}

func (a *App) activationDTOsWithLiveShared(ctx context.Context, acts []store.ProfileActivation, mikrotikName string) []map[string]any {
	mtMap, err := a.mikrotikSharedUsersByName()
	if err != nil {
		return activationDTOs(acts)
	}
	var out []map[string]any
	var latestUnsettledSynced bool
	for _, act := range acts {
		su := act.SharedUsers
		if !act.IsSettled && !latestUnsettledSynced {
			su = liveSharedUsers(mtMap, mikrotikName, act.SharedUsers)
			if su != act.SharedUsers {
				_ = a.Store.UpdateActivationSharedUsers(ctx, act.ID, su)
			}
			latestUnsettledSynced = true
		}
		row := map[string]any{
			"id": act.ID, "profile_name": act.ProfileName, "shared_users": su,
			"currency": act.Currency, "is_settled": act.IsSettled, "created_at": act.CreatedAt,
		}
		if act.AmountPaid.Valid {
			row["amount_paid"] = act.AmountPaid.Float64
		}
		if act.MikrotikEndTime.Valid {
			row["mikrotik_end_time"] = act.MikrotikEndTime.String
		}
		if act.Note.Valid {
			row["note"] = act.Note.String
		}
		out = append(out, row)
	}
	return out
}

func (a *App) applyRenewalsLiveSharedUsers(ctx context.Context, filter store.RenewalFilter, items []store.RenewalRow) (store.RenewalSummary, error) {
	scopeRows, err := a.Store.ListRenewalSharedUsersInScope(ctx, filter)
	if err != nil {
		return store.RenewalSummary{}, err
	}
	needsLive := filter.Settled != "settled"
	if !needsLive {
		var unsettledSum, allSum int
		for _, r := range scopeRows {
			allSum += r.SharedUsers
			if !r.IsSettled {
				unsettledSum += r.SharedUsers
			}
		}
		return store.RenewalSummary{UnsettledSharedUsersSum: unsettledSum, AllSharedUsersSum: allSum}, nil
	}

	mtMap, err := a.mikrotikSharedUsersByName()
	if err != nil {
		return store.RenewalSummary{}, err
	}

	latestPerUser := latestUnsettledPerUser(scopeRows)
	var unsettledSum, allSum int
	for _, r := range scopeRows {
		su := r.SharedUsers
		if !r.IsSettled {
			if latestPerUser[r.MikrotikName] == r.ID {
				su = liveSharedUsers(mtMap, r.MikrotikName, su)
				if su != r.SharedUsers {
					_ = a.Store.UpdateActivationSharedUsers(ctx, r.ID, su)
				}
			}
			unsettledSum += su
		}
		allSum += su
	}

	for i := range items {
		if items[i].IsSettled {
			continue
		}
		if latestPerUser[items[i].MikrotikName] == items[i].ID {
			items[i].SharedUsers = liveSharedUsers(mtMap, items[i].MikrotikName, items[i].SharedUsers)
		}
	}

	return store.RenewalSummary{UnsettledSharedUsersSum: unsettledSum, AllSharedUsersSum: allSum}, nil
}
