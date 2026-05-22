package store

import "database/sql"

type PanelAdmin struct {
	ID           int64
	Username     string
	PasswordHash string
	IsActive     bool
}

type Manager struct {
	ID           int64
	Username     string
	PasswordHash string
	DisplayName  string
	Slug         string
	Quota        int
	IsActive     bool
	CreatedAt    string
	UpdatedAt    string
}

type ManagerSlug struct {
	ID   int64
	Slug string
}

type VPNUserMeta struct {
	ID           int64
	MikrotikName string
	ManagerID    sql.NullInt64
	LocalName    string
	ContactPhone sql.NullString
	ContactNote  sql.NullString
	Notes        sql.NullString
	CreatedAt    string
	UpdatedAt    string
}

type ProfileActivation struct {
	ID               int64
	VPNUserMetaID    int64
	ProfileName      string
	SharedUsers      int
	AmountPaid       sql.NullFloat64
	Currency         string
	PaidAt           sql.NullString
	Note             sql.NullString
	MikrotikEndTime  sql.NullString
	IsSettled        bool
	SettledAt        sql.NullString
	SettledByAdminID sql.NullInt64
	CreatedAt        string
}

type RenewalFilter struct {
	ManagerID   *int64
	OrphanOnly  bool
	Settled     string
	From        string
	To          string
	Query       string
	Page        int
	PageSize    int
}

type RenewalSummary struct {
	UnsettledSharedUsersSum int `json:"unsettled_shared_users_sum"`
	AllSharedUsersSum       int `json:"all_shared_users_sum"`
}

type RenewalRow struct {
	ID                 int64
	RenewedAt          string
	MikrotikName       string
	ManagerID          *int64
	ManagerDisplayName *string
	SharedUsers        int
	ProfileName        string
	ProfileState       string
	MikrotikEndTime    *string
	IsSettled          bool
	AmountPaid         *float64
	Currency           string
}

type RenewalThrough struct {
	ActivationID int64
	ManagerID    *int64
	OrphanOnly   bool
}
