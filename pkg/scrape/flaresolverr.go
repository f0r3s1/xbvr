package scrape

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/xbapps/xbvr/pkg/config"
)

// Global shared transport for FlareSolverr to reuse browser session
var (
	sharedFlareSolverrTransport *flareSolverrTransport
	sharedTransportMu           sync.Mutex
)

type flareSolverrTransport struct {
	mu        sync.Mutex
	sessionID string
	client    *http.Client
	baseURL   string
}

func getSharedFlareSolverrTransport() *flareSolverrTransport {
	sharedTransportMu.Lock()
	defer sharedTransportMu.Unlock()

	currentURL := config.Config.Advanced.FlareSolverrAddress

	// Create new transport if none exists or if URL changed
	if sharedFlareSolverrTransport == nil || sharedFlareSolverrTransport.baseURL != currentURL {
		sharedFlareSolverrTransport = &flareSolverrTransport{
			client: &http.Client{
				Timeout: 300 * time.Second,
				Transport: &http.Transport{
					DisableKeepAlives: false,
					MaxConnsPerHost:   1,
				},
			},
			baseURL: currentURL,
		}
	}
	return sharedFlareSolverrTransport
}

func (fst *flareSolverrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Lock only for session creation, not the whole request
	fst.mu.Lock()
	if fst.sessionID == "" {
		if err := fst.createSession(); err != nil {
			fst.mu.Unlock()
			return nil, fmt.Errorf("session creation failed: %w", err)
		}
	}
	sessionID := fst.sessionID
	fst.mu.Unlock()

	// Random delay 100-300ms (~200ms avg = ~50 requests per 10 seconds)
	delay := time.Duration(100+rand.Intn(200)) * time.Millisecond
	time.Sleep(delay)

	ctx, cancel := context.WithTimeout(req.Context(), 300*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// Log actual request (not just queued)
	log.Infof("FlareSolverr fetching: %s", req.URL.String())

	if fst.baseURL == "" {
		return nil, fmt.Errorf("flaresolverr address not configured")
	}

	payload := map[string]interface{}{
		"cmd":        "request.get",
		"url":        req.URL.String(),
		"session":    sessionID,
		"maxTimeout": 120000,
	}

	jsonPayload, _ := json.Marshal(payload)
	flareReq, _ := http.NewRequestWithContext(ctx, "POST", fst.baseURL+"/v1", bytes.NewReader(jsonPayload))
	flareReq.Header.Set("Content-Type", "application/json")

	log.Debugf("FlareSolverr requesting: %s", req.URL.String())

	resp, err := fst.client.Do(flareReq)
	if err != nil {
		log.Errorf("FlareSolverr request failed: %v", err)
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
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	// Clear body slice to allow GC
	body = nil

	if result.Status != "ok" {
		return nil, fmt.Errorf("flaresolverr error: %s (%s)", result.Status, result.Message)
	}

	responseBytes := []byte(result.Solution.Response)
	// Clear the string from result to allow GC
	result.Solution.Response = ""

	log.Debugf("FlareSolverr response: %s (%d bytes)", req.URL.String(), len(responseBytes))

	headers := convertHeaders(result.Solution.Headers)
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "text/html")
	}

	return &http.Response{
		StatusCode: result.Solution.Status,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(responseBytes)),
	}, nil
}

func (fst *flareSolverrTransport) createSession() error {
	log.Infof("Creating FlareSolverr session at %s", fst.baseURL)

	if fst.baseURL == "" {
		return fmt.Errorf("flaresolverr address not configured")
	}

	payload := map[string]interface{}{"cmd": "sessions.create"}
	jsonPayload, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", fst.baseURL+"/v1", bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := fst.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to FlareSolverr: %w", err)
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
	log.Infof("FlareSolverr session created: %s", result.Session)
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
	transport := getSharedFlareSolverrTransport()

	// Pre-create session so it's ready before scraping
	transport.mu.Lock()
	if transport.sessionID == "" {
		transport.createSession()
	}
	transport.mu.Unlock()

	// Expand domains to include www. variants
	expandedDomains := make([]string, 0, len(domains)*2)
	for _, domain := range domains {
		expandedDomains = append(expandedDomains, domain)
		if !strings.HasPrefix(domain, "www.") {
			expandedDomains = append(expandedDomains, "www."+domain)
		}
	}

	log.Debugf("FlareSolverr collector for: %v", expandedDomains)

	// Don't use async mode - FlareSolverr is sequential anyway and async causes memory buildup
	c := colly.NewCollector(
		colly.AllowedDomains(expandedDomains...),
	)

	c.SetClient(&http.Client{
		Transport: transport,
		Timeout:   300 * time.Second,
	})

	// FlareSolverr is inherently slow (browser-based), sequential processing
	// Parallelism 1 is required since we share a single browser session
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       0,
	})

	// Apply the same callbacks and setup as regular collectors
	c = createCallbacks(c)

	// Setup headers/cookies for domains
	for _, domain := range domains {
		SetupCollector(GetCoreDomain(domain)+"-scraper", c)
	}

	return c
}

// FlareSolverrGet performs an HTTP GET request through FlareSolverr
// This is useful for API-based scrapers that use resty instead of colly
func FlareSolverrGet(url string) (string, error) {
	transport := getSharedFlareSolverrTransport()

	// Ensure session is created
	transport.mu.Lock()
	if transport.sessionID == "" {
		if err := transport.createSession(); err != nil {
			transport.mu.Unlock()
			return "", fmt.Errorf("session creation failed: %w", err)
		}
	}
	sessionID := transport.sessionID
	transport.mu.Unlock()

	// Random delay 100-300ms
	delay := time.Duration(100+rand.Intn(200)) * time.Millisecond
	time.Sleep(delay)

	log.Infof("FlareSolverr GET: %s", url)

	if transport.baseURL == "" {
		return "", fmt.Errorf("flaresolverr address not configured")
	}

	payload := map[string]interface{}{
		"cmd":        "request.get",
		"url":        url,
		"session":    sessionID,
		"maxTimeout": 120000,
	}

	jsonPayload, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", transport.baseURL+"/v1", bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := transport.client.Do(req)
	if err != nil {
		log.Errorf("FlareSolverr request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Solution struct {
			Response string `json:"response"`
			Status   int    `json:"status"`
		} `json:"solution"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "ok" {
		return "", fmt.Errorf("flaresolverr error: %s (%s)", result.Status, result.Message)
	}

	log.Debugf("FlareSolverr GET response: %s (%d bytes)", url, len(result.Solution.Response))
	return result.Solution.Response, nil
}
