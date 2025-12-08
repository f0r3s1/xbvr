package server

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gregjones/httpcache/diskcache"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"willnorris.com/go/imageproxy"
)

// ImageProxyFallbackHandler wraps the imageproxy and provides fallback to external proxy
// when the original request fails or returns a non-image response.
type ImageProxyFallbackHandler struct {
	ImageProxy *imageproxy.Proxy
	Cache      *diskcache.Cache
}

// NewImageProxyFallbackHandler creates a new handler with fallback capability
func NewImageProxyFallbackHandler(proxy *imageproxy.Proxy, cache *diskcache.Cache) *ImageProxyFallbackHandler {
	return &ImageProxyFallbackHandler{
		ImageProxy: proxy,
		Cache:      cache,
	}
}

// isImageContentType checks if the content type indicates an image
func isImageContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "image/")
}

// buildExternalProxyURL constructs the URL to fetch the image through the external proxy
func buildExternalProxyURL(originalURL string) string {
	proxyURL := config.Config.Advanced.ImageProxyURL
	keyName := config.Config.Advanced.ImageProxyApiKeyName
	keyValue := config.Config.Advanced.ImageProxyApiKeyValue

	if proxyURL == "" {
		return ""
	}

	// Parse the proxy URL to add query parameters
	u, err := url.Parse(proxyURL)
	if err != nil {
		common.Log.Errorf("Invalid image proxy URL: %v", err)
		return ""
	}

	q := u.Query()
	// Add the target URL as a parameter
	q.Set("url", originalURL)
	// Add API key if configured
	if keyName != "" && keyValue != "" {
		q.Set(keyName, keyValue)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// extractOriginalURL extracts the original image URL from the imageproxy request path
// The path format is: /[options]/[url] where url has :// replaced with :/
func extractOriginalURL(path string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Find the URL part - it's after the options (e.g., "700x/", "120x/", etc.)
	// Options end at the first occurrence of a URL scheme pattern
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return ""
	}

	urlPart := parts[1]
	// The URL has :// replaced with :/ in the path, restore it
	// Handle both http:/ and https:/
	if strings.HasPrefix(urlPart, "http:/") && !strings.HasPrefix(urlPart, "http://") {
		urlPart = strings.Replace(urlPart, "http:/", "http://", 1)
	} else if strings.HasPrefix(urlPart, "https:/") && !strings.HasPrefix(urlPart, "https://") {
		urlPart = strings.Replace(urlPart, "https:/", "https://", 1)
	}

	return urlPart
}

// ResponseCapture captures the response from the underlying handler
type ResponseCapture struct {
	http.ResponseWriter
	statusCode  int
	body        *bytes.Buffer
	contentType string
	headers     http.Header
	written     bool
}

func newResponseCapture(w http.ResponseWriter) *ResponseCapture {
	return &ResponseCapture{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		headers:        make(http.Header),
		statusCode:     http.StatusOK,
	}
}

func (r *ResponseCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.contentType = r.Header().Get("Content-Type")
	// Copy headers
	for k, v := range r.Header() {
		r.headers[k] = v
	}
}

func (r *ResponseCapture) Write(b []byte) (int, error) {
	r.written = true
	return r.body.Write(b)
}

// flushToClient sends the captured response to the client
func (r *ResponseCapture) flushToClient() {
	// Copy headers to the underlying response writer
	for k, v := range r.headers {
		for _, vv := range v {
			r.ResponseWriter.Header().Add(k, vv)
		}
	}
	r.ResponseWriter.WriteHeader(r.statusCode)
	io.Copy(r.ResponseWriter, r.body)
}

// fetchFromExternalProxy fetches the image from the external proxy
func fetchFromExternalProxy(externalURL string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", externalURL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to look like a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	return client.Do(req)
}

func (h *ImageProxyFallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if external proxy is configured
	if config.Config.Advanced.ImageProxyURL == "" {
		// No fallback configured, just use the original proxy
		h.ImageProxy.ServeHTTP(w, r)
		return
	}

	// Extract the original URL from the request path
	originalURL := extractOriginalURL(r.URL.Path)
	if originalURL == "" {
		h.ImageProxy.ServeHTTP(w, r)
		return
	}

	// Create a cache key for the proxied response
	cacheKey := "fallback:" + originalURL

	// Check if we have a cached response from the external proxy
	if cachedData, ok := h.Cache.Get(cacheKey); ok {
		// Serve from cache
		// Determine content type from cached data
		contentType := http.DetectContentType(cachedData)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		w.Write(cachedData)
		return
	}

	// Capture the response from the imageproxy
	capture := newResponseCapture(w)
	h.ImageProxy.ServeHTTP(capture, r)

	// Check if the response is valid
	isValidImage := capture.statusCode >= 200 && capture.statusCode < 300 &&
		isImageContentType(capture.contentType)

	if isValidImage {
		// Response is a valid image, send it to the client
		capture.flushToClient()
		return
	}

	// Response is not a valid image, try the external proxy
	common.Log.Debugf("Image proxy fallback: original request failed or returned non-image (status: %d, content-type: %s), trying external proxy for %s",
		capture.statusCode, capture.contentType, originalURL)

	externalURL := buildExternalProxyURL(originalURL)
	if externalURL == "" {
		// Can't build external URL, return the original response
		capture.flushToClient()
		return
	}

	resp, err := fetchFromExternalProxy(externalURL)
	if err != nil {
		common.Log.Errorf("Image proxy fallback: failed to fetch from external proxy: %v", err)
		capture.flushToClient()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		common.Log.Errorf("Image proxy fallback: external proxy returned error status: %d", resp.StatusCode)
		capture.flushToClient()
		return
	}

	// Check content type from external proxy
	externalContentType := resp.Header.Get("Content-Type")
	if !isImageContentType(externalContentType) {
		common.Log.Errorf("Image proxy fallback: external proxy returned non-image content type: %s", externalContentType)
		capture.flushToClient()
		return
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		common.Log.Errorf("Image proxy fallback: failed to read external proxy response: %v", err)
		capture.flushToClient()
		return
	}

	// Cache the response for future requests
	h.Cache.Set(cacheKey, body)

	// Send the response to the client
	w.Header().Set("Content-Type", externalContentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(body)

	common.Log.Debugf("Image proxy fallback: successfully fetched image from external proxy for %s", originalURL)
}
