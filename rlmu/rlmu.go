package rlmu

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Info - information collected from the response headers
type Info struct {
	limit      int // from  "X-RateLimit-Limit" header
	burst      int // from  "X-RateLimit-Burst" header
	remaining  int // from  "X-RateLimit-Remaining" header
	reset      int // from  "X-RateLimit-Reset" header
	retryAfter int // from  "Retry-After" header
}

// RlmInfo Extracts the rate limiting information from the headers
// of an Atlas API response.
func rlmInfo(header http.Header) (*Info, error) {
	// Extract the rate limiting headers
	strLimit := header.Get("X-RateLimit-Limit")
	strBurst := header.Get("X-RateLimit-Burst")
	strRemaining := header.Get("X-RateLimit-Remaining")
	strReset := header.Get("X-RateLimit-Reset")
	strRetryAfter := header.Get("Retry-After")

	// get the numeric value of  "X-RateLimit-Limit"
	var limit int
	if strLimit != "" {
		n, err := strconv.Atoi(strLimit)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		limit = n
	}

	// get the numeric value of "X-RateLimit-Burst"
	var burst int
	if strBurst != "" {
		n, err := strconv.Atoi(strBurst)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		burst = n
	}

	// get the numeric value of "X-RateLimit-Remaining"
	var remaining int
	if strRemaining != "" {
		n, err := strconv.Atoi(strRemaining)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		remaining = n
	}

	// get the numeric value of "X-RateLimit-Reset"
	var reset int
	if strReset != "" {
		n, err := strconv.Atoi(strReset)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		reset = n
	}

	// get the numeric value of  "Retry-After"
	var retryAfter int
	if strRetryAfter != "" {
		n, err := strconv.Atoi(strRetryAfter)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		retryAfter = n
	}
	rli := &Info{
		limit:      limit,
		burst:      burst,
		remaining:  remaining,
		reset:      reset,
		retryAfter: retryAfter,
	}
	return rli, nil
}

type RateLimiter struct {
	info         *Info
	logger       *zap.Logger
	retryWaiting atomic.Bool
	resetWaiting atomic.Bool
	remaining    int
}

func (s *RateLimiter) GetSlot() bool {
	return s.remaining > 0 && !s.retryWaiting.Load() && !s.resetWaiting.Load()
}

func (s *RateLimiter) Update(header http.Header) {
	// Get the rate limiting information from the response
	info, err := rlmInfo(header)
	if err != nil {
		return
	}
	s.info = info
	s.remaining = info.remaining
	if s.info.retryAfter > 0 {
		s.retryWaiting.Store(true)
		time.AfterFunc(time.Duration(s.info.retryAfter*int(time.Second)), func() {
			s.retryWaiting.Store(false)
			s.remaining = 1
		})
	} else if s.info.remaining == 0 {
		s.resetWaiting.Store(true)
		time.AfterFunc(time.Duration(s.info.reset*int(time.Millisecond)), func() {
			s.resetWaiting.Store(false)
			s.remaining = 1
		})
	}
}

func New(h http.Header, maxWaitingRequests int, logger *zap.Logger) (*RateLimiter, error) {
	info, err := rlmInfo(h)
	if err != nil {
		return nil, err
	}
	s := &RateLimiter{
		info:      info,
		logger:    logger,
		remaining: info.remaining,
	}
	return s, nil
}
