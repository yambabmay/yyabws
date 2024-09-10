package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/yambabmay/yyabws/server/rlmd"
	"github.com/yambabmay/yyabws/server/rlmu"
	"go.uber.org/zap"
)

const (
	atlasURL       = "https://atlas.abiosgaming.com/v3"
	secretHederKey = "Abios-Secret"
)

// atlasClient - Atlas client
type atlasClient struct {
	settings *Settings
	ds       *rlmd.RateLimiter
	us       *rlmu.RateLimiter
	logger   *zap.Logger
}

func (s *atlasClient) init() error {
	bURL, _ := url.Parse(s.settings.URL + "/series")
	values := url.Values{}
	values.Set("lifecycle", "live")
	values.Set("take", fmt.Sprintf("%d", 1))
	bURL.RawQuery = values.Encode()
	req, _ := http.NewRequest(http.MethodGet, bURL.String(), nil)
	req.Header.Add(secretHederKey, s.settings.Secret)
	cli := &http.Client{}
	rsp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	us, err := rlmu.New(rsp.Header, s.logger)
	if err != nil {
		return err
	}
	s.us = us

	io.Copy(io.Discard, rsp.Body)
	return nil
}

func newAtlasClient(settings *Settings, logger *zap.Logger) (ac *atlasClient, err error) {
	ds, err := rlmd.New(settings.dsRlmSettings, logger)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			ds.Close()
		}
	}()

	ac = &atlasClient{
		settings: settings,
		ds:       ds,
		logger:   logger,
	}
	err = ac.init()
	return ac, nil
}

func (s *atlasClient) forwardRequest(resp http.ResponseWriter, req *http.Request, path string) {
	status, secret, err := s.ds.Allow(context.Background(), req)
	if err != nil {
		if errors.Is(err, rlmd.ErrTooManyRequests) {
			dsInfo, err := s.ds.Info(context.Background(), secret)
			if err != nil {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			for k, v := range dsInfo {
				resp.Header().Add(k, v)
			}
			resp.WriteHeader(status)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	if status != http.StatusOK {
		resp.WriteHeader(status)
		return
	}
	// Check if the upstream rate limiter allows this request
	if !s.us.Slot() {
		s.logger.Debug("upstream rate limiting: no slots available")
		resp.WriteHeader(http.StatusTooManyRequests)
		return
	}
	// Create Atlas base url for series
	baseURL, err := url.Parse(atlasURL + path)
	if err != nil {
		s.logger.Error("url parse", zap.Error(err))
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
			s.logger.Error("parse take parameter", zap.Error(err))
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
			s.logger.Error("parse skip parameter", zap.Error(err))
			resp.WriteHeader(http.StatusBadRequest)
		}
		ruValues.Set("skip", fmt.Sprintf("%d", skip))
	}
	baseURL.RawQuery = ruValues.Encode()
	// Create the upstream request
	newReq, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		s.logger.Error("parse skip parameter", zap.Error(err))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set the authentication header
	newReq.Header.Add(secretHederKey, s.settings.Secret)
	// Send the request to Atlas
	client := &http.Client{}
	newResp, err := client.Do(newReq)
	if err != nil {
		s.logger.Error("error response from atlas", zap.Error(err))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer newResp.Body.Close()
	// Check the status code
	if newResp.StatusCode != http.StatusOK {
		if newResp.StatusCode == http.StatusTooManyRequests {
			s.logger.Debug("too many requests from Atlas")
		}
		resp.WriteHeader(newResp.StatusCode)
		return
	}
	s.us.Update(newResp.Header)
	dsInfo, err := s.ds.Info(context.Background(), secret)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Add("Content-Type", newResp.Header.Get("Content-Type"))
	resp.Header().Add("Content-Length", fmt.Sprintf("%d", newResp.ContentLength))
	for k, v := range dsInfo {
		resp.Header().Add(k, v)
	}
	io.Copy(resp, newResp.Body)
}

// seriesLiveRequestHandler request handler for endpoint "/series/live"
func (s *atlasClient) seriesLiveRequestHandler(resp http.ResponseWriter, req *http.Request) {
	s.forwardRequest(resp, req, "/series")
}

// playersLiveRequestHandler request handler for endpoint "/players/live"
func (s *atlasClient) playersLiveRequestHandler(resp http.ResponseWriter, req *http.Request) {
	s.forwardRequest(resp, req, "/players")
}

// teamsLiveRequestHandler request handler for endpoint "/teams/live"
func (s *atlasClient) teamsLiveRequestHandler(resp http.ResponseWriter, req *http.Request) {
	s.forwardRequest(resp, req, "/teams")
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	s, err := newAtlasClient(loadSettings(), logger)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /series/live", s.seriesLiveRequestHandler)
	mux.HandleFunc("GET /players/live", s.playersLiveRequestHandler)
	mux.HandleFunc("GET /teams/live", s.teamsLiveRequestHandler)
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Fatal(err)
	}
}
