package rlmu

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// Represents a sequence of
type Burst struct {
	sync.Mutex
	limit             int
	nextReset         time.Time
	nextRetry         time.Time
	infoCount         int
	slots             int
	logger            *zap.Logger
	resetMilliseconds int // Just for logging
}

func (s *Burst) Slot() bool {
	s.Lock()
	defer s.Unlock()
	if !s.nextRetry.IsZero() {
		// We are in "Retry-After" situation
		return false
	}
	if s.slots < s.limit {
		s.slots++
		return true
	}
	return false
}

func (s *Burst) Update(info *Info) {
	s.Lock()
	defer s.Unlock()
	// Increment info count
	s.infoCount++
	// Update the limit. Can it change in the fly? Assuming it can.
	s.limit = info.limit
	if info.retryAfter > 0 {
		// We got a "Retry-After" header
		retryTime := time.Now().Add(time.Duration(info.retryAfter * int(time.Second)))
		if retryTime.After(s.nextRetry) {
			s.nextRetry = retryTime
		}
	}
	//X-RateLimit-Reset is always present in the headers
	nextReset := time.Now().Add(time.Duration(info.reset * int(time.Millisecond)))
	if nextReset.After(s.nextReset) {
		s.resetMilliseconds = info.reset
		s.nextReset = nextReset
	}
}

// Reset - returns true if it is time to reset.
func (s *Burst) Reset() bool {
	s.Lock()
	defer s.Unlock()
	if !s.nextRetry.IsZero() && time.Now().After(s.nextRetry) {
		return true
	}
	if !s.nextReset.IsZero() && s.nextReset.Before(time.Now()) && s.infoCount >= s.limit {
		s.logger.Debug("Reset here")
		return true
	}
	return false
}
