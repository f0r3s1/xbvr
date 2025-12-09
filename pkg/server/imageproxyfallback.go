package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"willnorris.com/go/imageproxy"
)

const (
	// Timeouts for external proxy requests
	externalProxyTimeout = 30 * time.Second
	maxBodyReadSize      = 50 * 1024 * 1024 // 50MB max image size
)

// ImageProxyFallbackHandler wraps the imageproxy and provides fallback to external proxy
// when the original request fails or returns a non-image response.
type ImageProxyFallbackHandler struct {
	ImageProxy *imageproxy.Proxy
	Cache      ImageCache
}

// NewImageProxyFallbackHandler creates a new handler with fallback capability
func NewImageProxyFallbackHandler(proxy *imageproxy.Proxy, cache ImageCache) *ImageProxyFallbackHandler {
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

// getExtensionForContentType returns the appropriate file extension for a content type
func getExtensionForContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "avif"):
		return ".avif"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		return ".jpg"
	default:
		return ".bin"
	}
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
	// Copy headers at the time WriteHeader is called
	for k, v := range r.Header() {
		r.headers[k] = v
	}
	r.contentType = r.headers.Get("Content-Type")
}

func (r *ResponseCapture) Write(b []byte) (int, error) {
	r.written = true
	return r.body.Write(b)
}

// flushToClient sends the captured response to the client
func (r *ResponseCapture) flushToClient() {
	// Detect actual content type from body
	bodyBytes := r.body.Bytes()
	actualContentType := http.DetectContentType(bodyBytes)

	// Copy headers to the underlying response writer, but fix Content-Length and Content-Type
	for k, v := range r.headers {
		// Skip headers we'll set ourselves
		if strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Content-Type") || strings.EqualFold(k, "Content-Disposition") {
			continue
		}
		for _, vv := range v {
			r.ResponseWriter.Header().Add(k, vv)
		}
	}
	// Set correct Content-Type based on actual body content
	r.ResponseWriter.Header().Set("Content-Type", actualContentType)
	// Set correct Content-Length based on actual captured body
	r.ResponseWriter.Header().Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
	// Set correct file extension for downloads based on actual content type
	ext := getExtensionForContentType(actualContentType)
	r.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"image%s\"", ext))
	r.ResponseWriter.WriteHeader(r.statusCode)
	r.ResponseWriter.Write(bodyBytes)
}

// fetchFromExternalProxy fetches the image from the external proxy with timeout
func fetchFromExternalProxy(ctx context.Context, externalURL string) (*http.Response, error) {
	// Create request with context for cancellation
	req, err := http.NewRequestWithContext(ctx, "GET", externalURL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to look like a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	client := &http.Client{
		Timeout: externalProxyTimeout,
	}
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
	// Use the full path (including size options) so different sizes are cached separately
	cacheKey := "fallback:" + r.URL.Path

	// Check if we have a cached response from the external proxy
	if cachedData, ok := h.Cache.Get(cacheKey); ok {
		// Serve from cache
		contentType := http.DetectContentType(cachedData)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cachedData)))
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("X-Cache", "HIT-FALLBACK")
		// Set correct file extension for downloads based on actual content type
		ext := getExtensionForContentType(contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"image%s\"", ext))
		w.WriteHeader(http.StatusOK)
		w.Write(cachedData)
		return
	}

	// Capture the response from the imageproxy
	capture := newResponseCapture(w)
	h.ImageProxy.ServeHTTP(capture, r)

	// Check if the response is valid - check both header content-type and detect from body
	detectedType := http.DetectContentType(capture.body.Bytes())
	isValidImage := capture.statusCode >= 200 && capture.statusCode < 300 &&
		(isImageContentType(capture.contentType) || isImageContentType(detectedType))

	if isValidImage {
		// Response is a valid image, cache it and send to client
		h.Cache.Set(cacheKey, capture.body.Bytes())
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

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(r.Context(), externalProxyTimeout)
	defer cancel()

	resp, err := fetchFromExternalProxy(ctx, externalURL)
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

	// Read the response body with size limit to prevent memory issues
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyReadSize))
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
