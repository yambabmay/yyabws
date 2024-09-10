// This package manages the and acts on the upstream [Atlas side]
// rate limiting information.
package rlmu

import (
	"log"
	"net/http"
	"strconv"
	"sync"
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
func RlmInfo(header http.Header) (*Info, error) {
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

// RateLimiter - Processes the  rate limiting information
// received from Atlas API
type RateLimiter struct {
	sync.Mutex
	// manages the current rate limiting information.
	burst *Burst
	// slotQueueLen the number of currently queued requests. To
	// forward a client request to the request handler has to
	// acquire a slot in a burst.
	slotQueueLen int
	// slotQueueMax is the maximum length of the slot queue. This
	// defaults to 3 times the value of X-RateLimit-Burst in the
	// first response from Atlas API
	slotQueueMax int
	logger       *zap.Logger
}

// slot - Returns true if a slot is available
func (s *RateLimiter) slot() bool {
	s.Lock()
	defer s.Unlock()
	return s.burst.Slot()
}

// update - Updates the state of the current burst
func (s *RateLimiter) update(info *Info) {
	s.Lock()
	defer s.Unlock()
	s.burst.Update(info)
}

// reset - performs the reset action
func (s *RateLimiter) reset() {
	s.Lock()
	defer s.Unlock()
	if s.burst.Reset() {
		s.burst = &Burst{
			limit:  s.burst.limit,
			logger: s.logger,
		}
	}
}

// enqueue - allocate a place in the slot queue.
// returns true on success
func (s *RateLimiter) enqueue() (success bool) {
	s.Lock()
	defer s.Unlock()
	if s.slotQueueLen < s.slotQueueMax {
		s.slotQueueLen++
		success = true
	}
	return success
}

// dequeue - frees a previously allocated slot
// queue place.
func (s *RateLimiter) dequeue() {
	s.Lock()
	defer s.Unlock()
	if s.slotQueueLen > 0 {
		s.slotQueueLen--
	}
}

// Slot - tries to acquire a burst slot
func (s *RateLimiter) Slot() bool {
	// reset the slot, if necessary.
	s.reset()
	// try to get a slot
	if s.slot() {
		return true
	}
	// try get a place on the queue
	if s.enqueue() {
		// go and wait for a slot
		return s.WaitForSlot()
	}
	// There is no place in the queue
	s.logger.Debug("slot queue is full")
	return false
}

func (s *RateLimiter) WaitForSlot() bool {
	// Prepare to free the queue place
	defer s.dequeue()
	// Wait a maximum of 4 seconds in the queue.
	tmr := time.NewTimer(4 * time.Second)
	defer tmr.Stop()
	tkr := time.NewTicker(10 * time.Millisecond)
	defer tkr.Stop()
	for {
		select {
		case <-tmr.C:
			// Leave the queue without a slot
			s.logger.Debug("leaving slot queue without a slot")
			return false
		default:
			s.reset()
			if s.slot() {
				s.logger.Debug("leaving slot queue with a slot")
				// Leave the queue with a slot
				return true
			}
			<-tkr.C
		}
	}
}

// Update called from request handlers to update the
// the rate limiting information
func (s *RateLimiter) Update(header http.Header) {
	info, err := RlmInfo(header)
	if err != nil {
		return
	}
	s.logger.Debug("rate limiter from atlas",
		zap.Int("X-RateLimit-Limit", info.limit),
		zap.Int("X-RateLimit-Burst", info.burst),
		zap.Int("X-RateLimit-Remaining", info.remaining),
		zap.Int("X-RateLimit-Reset", info.reset),
		zap.Int("Retry-After", info.retryAfter),
	)
	s.update(info)
	s.reset()
}

// New - Creates a new RateLimiter
func New(header http.Header, logger *zap.Logger) (rl *RateLimiter, err error) {
	info, err := RlmInfo(header)
	if err != nil {
		return nil, err
	}
	rl = &RateLimiter{
		logger: logger,
		burst: &Burst{
			logger: logger,
		},
		// Allow a queue of 10 times the initial value of X-RateLimit-Limit
		// requests. This can be prohibitive if info.limit is too large. In
		// a production scenario this should be a configurable setting or
		// calculated from the information on the available resources.
		slotQueueMax: info.limit * 10,
	}
	rl.update(info)
	return rl, nil
}
