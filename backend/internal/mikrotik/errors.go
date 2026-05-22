package mikrotik

import "errors"

var (
	ErrNotFound      = errors.New("mikrotik: not found")
	ErrNameTaken     = errors.New("mikrotik: name taken")
	ErrUnavailable   = errors.New("mikrotik: unavailable")
	ErrProfileMissing = errors.New("mikrotik: profile not found")
)
