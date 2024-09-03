package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	atlasURL       = "https://atlas.abiosgaming.com/v3"
	secretHederKey = "Abios-Secret"
)

// rateLimitingInfo - information collected from the response headers
type rateLimitingInfo struct {
	limit      int // from  "X-RateLimit-Limit" header
	burst      int // from  "X-RateLimit-Limit" header
	remaining  int // from  "X-RateLimit-Limit" header
	reset      int // from  "X-RateLimit-Limit" header
	retryAfter int // from  "X-RateLimit-Limit" header
}

// rateLimiter - simple rate limiter that uses the upstream rate limiting
// information to obey it and rate limit the downstream requests. The
type rateLimiter struct {
	// For modification of the rate limiting information
	sync.Mutex
	info  rateLimitingInfo
	allow atomic.Bool
}

// extractRateLimitingInfo Extracts the rate limiting information from the headers
// of an Atlas API response.
func extractRateLimitingInfo(header http.Header) (rateLimitingInfo, error) {
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
			return rateLimitingInfo{}, err
		}
		limit = n
	}

	// get the numeric value of "X-RateLimit-Burst"
	var burst int
	if strBurst != "" {
		n, err := strconv.Atoi(strBurst)
		if err != nil {
			log.Println(err)
			return rateLimitingInfo{}, err
		}
		burst = n
	}

	// get the numeric value of "X-RateLimit-Remaining"
	var remaining int
	if strRemaining != "" {
		n, err := strconv.Atoi(strRemaining)
		if err != nil {
			log.Println(err)
			return rateLimitingInfo{}, err
		}
		remaining = n
	}

	// get the numeric value of "X-RateLimit-Reset"
	var reset int
	if strReset != "" {
		n, err := strconv.Atoi(strReset)
		if err != nil {
			log.Println(err)
			return rateLimitingInfo{}, err
		}
		reset = n
	}

	// get the numeric value of  "Retry-After"
	var retryAfter int
	if strRetryAfter != "" {
		n, err := strconv.Atoi(strRetryAfter)
		if err != nil {
			log.Println(err)
			return rateLimitingInfo{}, err
		}
		retryAfter = n
	}
	rli := rateLimitingInfo{
		limit:      limit,
		burst:      burst,
		remaining:  remaining,
		reset:      reset,
		retryAfter: retryAfter,
	}
	return rli, nil

}

// allowRequest Check if requests are allowed at this moment.
func (s *rateLimiter) allowRequest() bool {
	return s.allow.Load()
}

// update Update the rate limiter with upstream information
func (s *rateLimiter) update(info rateLimitingInfo) {
	s.Lock()
	defer s.Unlock()
	s.info = info
	if s.info.retryAfter > 0 {
		// Do not allow requests until the retry after time has passed
		s.allow.Store(false)
		time.AfterFunc(time.Duration(s.info.retryAfter*int(time.Second)), func() {
			s.allow.Store(true)
		})
	} else if s.info.remaining == 0 {
		// Do not allow requests until the replenishment time has passed
		s.allow.Store(false)
		time.AfterFunc(time.Duration(info.reset*int(time.Millisecond)), func() {
			s.allow.Store(true)
		})
	}
}

// atlasClient - Atlas client
type atlasClient struct {
	// Atlas secret
	secret string
	// Rate limiter
	rlm *rateLimiter
}

func newAtlasClient(secret string) (*atlasClient, error) {
	// Try to learn the upstream rate limitations. We send a request
	// for live series and  take=1.
	bURL, _ := url.Parse(atlasURL + "/series")
	values := url.Values{}
	values.Set("lifecycle", "live")
	values.Set("take", fmt.Sprintf("%d", 1))
	bURL.RawQuery = values.Encode()
	req, _ := http.NewRequest(http.MethodGet, bURL.String(), nil)
	req.Header.Add(secretHederKey, secret)
	cli := &http.Client{}
	rsp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	// Close the response body on return
	defer rsp.Body.Close()
	// Extract the rate limiting information fom the
	// Headers
	info, err := extractRateLimitingInfo(rsp.Header)
	if err != nil {
		return nil, err
	}
	// Create the rate limiter
	rlm := &rateLimiter{}
	// Assume we are allowed to send requests upstream
	rlm.allow.Store(true)
	// Update the rate limiter with the upstream information
	rlm.update(info)
	// Create the Atlas client
	ac := &atlasClient{
		secret: secret,
		rlm:    rlm,
	}
	// Discard the response body
	io.Copy(io.Discard, rsp.Body)
	// Return the Atlas client
	return ac, nil

}

func (ac *atlasClient) forwardRequest(resp http.ResponseWriter, req *http.Request, path string) {
	// Check if the rate limiter allows this request
	if !ac.rlm.allowRequest() {
		log.Println("too many requests from downstream")
		resp.WriteHeader(http.StatusTooManyRequests)
		return
	}

	// Create Atlas base url for series
	baseURL, err := url.Parse(atlasURL + path)
	if err != nil {
		log.Print(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set the query parameters
	ruValues := url.Values{}

	// Set the "lifecycle" query parameter
	ruValues.Set("lifecycle", "live")
	// Add "take" parameters if necessary
	if req.URL.Query().Has("take") {
		take, err := strconv.ParseUint(req.URL.Query().Get("take"), 10, 0)
		if err != nil {
			log.Print(err)
			resp.WriteHeader(http.StatusBadRequest)
		}
		if take > 50 {
			take = 50
		}
		ruValues.Set("take", fmt.Sprintf("%d", take))
	}
	if req.URL.Query().Has("skip") {
		skip, err := strconv.ParseUint(req.URL.Query().Get("skip"), 10, 0)
		if err != nil {
			log.Print(err)
			resp.WriteHeader(http.StatusBadRequest)
		}
		ruValues.Set("skip", fmt.Sprintf("%d", skip))
	}
	baseURL.RawQuery = ruValues.Encode()

	// Create the upstream request
	newReq, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		log.Print(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set the authentication header
	newReq.Header.Add(secretHederKey, ac.secret)
	// Send the request to Atlas
	client := &http.Client{}
	newResp, err := client.Do(newReq)
	if err != nil {
		log.Print(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer newResp.Body.Close()
	// Check the status code
	if newResp.StatusCode != http.StatusOK {
		resp.WriteHeader(newResp.StatusCode)
		return
	}
	rlmInfo, err := extractRateLimitingInfo(newResp.Header)
	if err != nil {
		log.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	ac.rlm.update(rlmInfo)
	resp.Header().Add("Content-Type", newResp.Header.Get("Content-Type"))
	resp.Header().Add("Content-Length", fmt.Sprintf("%d", newResp.ContentLength))
	io.Copy(resp, newResp.Body)
}

// seriesLiveRequestHandler request handler for endpoint "/series/live"
func (ac *atlasClient) seriesLiveRequestHandler(resp http.ResponseWriter, req *http.Request) {
	ac.forwardRequest(resp, req, "/series")
}

// playersLiveRequestHandler request handler for endpoint "/players/live"
func (ac *atlasClient) playersLiveRequestHandler(resp http.ResponseWriter, req *http.Request) {
	ac.forwardRequest(resp, req, "/players")
}

// teamsLiveRequestHandler request handler for endpoint "/teams/live"
func (ac *atlasClient) teamsLiveRequestHandler(resp http.ResponseWriter, req *http.Request) {
	ac.forwardRequest(resp, req, "/teams")
}

func main() {
	// Get the Atlas secret from the environment
	secret := os.Getenv("ATLAS_SECRET")
	if secret == "" {
		log.Fatal("env variable ATLAS_SECRET is not set")
	}
	// Create an Atlas client
	ac, err := newAtlasClient(secret)
	if err != nil {
		log.Fatal(err)
	}
	//
	mux := http.NewServeMux()
	mux.HandleFunc("GET /series/live", ac.seriesLiveRequestHandler)
	mux.HandleFunc("GET /players/live", ac.playersLiveRequestHandler)
	mux.HandleFunc("GET /teams/live", ac.teamsLiveRequestHandler)
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Fatal(err)
	}
}
