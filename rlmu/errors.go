package rlmu

import "errors"

var (
	ErrTooManyWaiting   = errors.New("too many waiting")
	ErrNoSlotsAvailable = errors.New("no slots available")
	ErrClosingDown      = errors.New("system closing down")
)
