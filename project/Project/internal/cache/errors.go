package cache

import "errors"

var (
	ErrBlockNotFound  = errors.New("block not found")
	ErrVictimDisabled = errors.New("victim cache is disabled")
)
