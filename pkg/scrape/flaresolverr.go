package scrape

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/config"
)

type flareSolverrTransport struct {
	mu        sync.Mutex
	sessionID string
	client    *http.Client
	baseURL   string
	limited   bool
}

func newFlareSolverrTransport() *flareSolverrTransport {
	return &flareSolverrTransport{
		client: &http.Client{
			Timeout: 300 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: false,
				MaxConnsPerHost:   10,
			},
		},
		baseURL: config.Config.Advanced.FlareSolverrAddress,
		limited: false,
	}
}

func (fst *flareSolverrTransport) SetLimited(limited bool) {
	fst.limited = limited
}

func (fst *flareSolverrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fst.mu.Lock()
	defer fst.mu.Unlock()

	if fst.limited {
		ScraperRateLimiterWait(req.URL.Host)
	}

	ctx, cancel := context.WithTimeout(req.Context(), 300*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	log.WithFields(logrus.Fields{
		"type":   "flare",
		"url":    req.URL.String(),
		"method": req.Method,
	}).Info("🚀 Starting FlareSolverr request")

	if fst.sessionID == "" {
		if err := fst.createSession(); err != nil {
			return nil, fmt.Errorf("session creation failed: %w", err)
		}
	}

	payload := map[string]interface{}{
		"cmd":        "request.get",
		"url":        req.URL.String(),
		"session":    fst.sessionID,
		"maxTimeout": 120000,
	}

	jsonPayload, _ := json.Marshal(payload)
	flareReq, _ := http.NewRequestWithContext(ctx, "POST", fst.baseURL+"/v1", bytes.NewReader(jsonPayload))
	flareReq.Header.Set("Content-Type", "application/json")

	resp, err := fst.client.Do(flareReq)
	if err != nil {
		log.WithFields(logrus.Fields{
			"type":  "flare",
			"error": err.Error(),
		}).Error("❌ FlareSolverr request failed")
		ScraperRateLimiterCheckErrors(req.URL.Host, err)
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Solution struct {
			Response string            `json:"response"`
			Status   int               `json:"status"`
			Headers  map[string]string `json:"headers"`
		} `json:"solution"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		log.WithFields(logrus.Fields{
			"type":  "flare",
			"error": err.Error(),
		}).Error("❌ Failed to parse FlareSolverr response")
		return nil, err
	}

	if result.Status != "ok" {
		log.WithFields(logrus.Fields{
			"type":    "flare",
			"status":  result.Status,
			"message": result.Message,
		}).Error("❌ FlareSolverr API error")
		return nil, fmt.Errorf("flare solverr error: %s (%s)", result.Status, result.Message)
	}

	log.WithFields(logrus.Fields{
		"type":   "flare",
		"url":    req.URL.String(),
		"status": result.Solution.Status,
		"length": len(result.Solution.Response),
	}).Info("✅ Successful FlareSolverr response")

	headers := convertHeaders(result.Solution.Headers)
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "text/html")
	}

	return &http.Response{
		StatusCode: result.Solution.Status,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader([]byte(result.Solution.Response))),
	}, nil
}

func (fst *flareSolverrTransport) createSession() error {
	log.Info("🆕 Creating new FlareSolverr session")

	payload := map[string]interface{}{"cmd": "sessions.create"}
	jsonPayload, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", fst.baseURL+"/v1", bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := fst.client.Do(req)
	if err != nil {
		return fmt.Errorf("session creation request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Session string `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("session creation parse error: %w", err)
	}

	if result.Status != "ok" {
		return fmt.Errorf("session creation failed: %s (%s)", result.Status, result.Message)
	}

	fst.sessionID = result.Session
	log.WithFields(logrus.Fields{
		"session": result.Session,
		"status":  result.Status,
	}).Info("🎉 New FlareSolverr session created")
	return nil
}

func convertHeaders(headers map[string]string) http.Header {
	h := make(http.Header)
	for k, v := range headers {
		h.Set(k, v)
	}
	return h
}

func createFlareSolverrCollector(domains ...string) *colly.Collector {
	transport := newFlareSolverrTransport()

	// Check if any domain has rate limiting
	if Limiters == nil {
		LoadScraperRateLimits()
	}
	for _, domain := range domains {
		if limiter := GetRateLimiter(domain); limiter != nil {
			transport.SetLimited(true)
			break
		}
	}

	c := colly.NewCollector(
		colly.AllowedDomains(domains...),
		colly.Async(true),
	)

	c.SetClient(&http.Client{
		Transport: transport,
		Timeout:   300 * time.Second,
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       2 * time.Second,
	})

	return c
}
