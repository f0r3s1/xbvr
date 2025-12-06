package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/fcjr/aia-transport-go"
)

// Change INCOMING response header's Cache-Control for persistent disk cache
type ForceCacheTransport struct {
	Transport http.RoundTripper
}

// getRefererForURL generates an appropriate Referer header for the given URL.
// This helps bypass anti-hotlinking protection by making requests appear to come from the same site.
func getRefererForURL(u *url.URL) string {
	if u == nil || u.Host == "" {
		return ""
	}
	// Return the origin (scheme + host) as the referer
	return u.Scheme + "://" + u.Host + "/"
}

// RoundTrip transport function that will force a Cache-Control of 5 years
// on all HTTP 2xx responses, so that httpcache used by imageproxy will continue
// to handle the cache as fresh, even when no cache header is set by upstream
// server.
// It also adds appropriate headers to bypass anti-hotlinking protection.
func (s *ForceCacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req := r.Clone(r.Context())

	// Add Referer header if not present - this helps bypass anti-hotlinking protection
	// Some sites like VRBangers block requests without a proper Referer
	if req.Header.Get("Referer") == "" {
		referer := getRefererForURL(req.URL)
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
	}

	// Add common browser headers to make requests look more legitimate
	// This helps bypass User-Agent based blocking
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}
	if req.Header.Get("Sec-Fetch-Dest") == "" {
		req.Header.Set("Sec-Fetch-Dest", "image")
	}
	if req.Header.Get("Sec-Fetch-Mode") == "" {
		req.Header.Set("Sec-Fetch-Mode", "no-cors")
	}
	if req.Header.Get("Sec-Fetch-Site") == "" {
		// Determine if this is same-origin or cross-site based on referer
		if strings.Contains(req.URL.Host, "vrbangers") ||
			strings.Contains(req.URL.Host, "content.vrbangers") {
			req.Header.Set("Sec-Fetch-Site", "same-site")
		} else {
			req.Header.Set("Sec-Fetch-Site", "cross-site")
		}
	}

	// Perform original request
	resp, err := s.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Overwrite cache behavior on 2xx responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Force cache duration in the diskCache to 5 years
		resp.Header.Set("Cache-Control", "public, max-age=157680000")
	}

	return resp, nil
}

func NewForceCacheTransport() *ForceCacheTransport {
	fct := new(ForceCacheTransport)

	// this is what willnorris.com/go/imageproxy does by default,
	// so keep the same here
	fct.Transport, _ = aia.NewTransport()

	return fct
}

// Change OUTGOING response header cache control, so that VR client
// will not cache as long as we do. This helps refresh the data in the
// VR client after a user has wiped the disk cache in xbvr.
type CacheHeaderResponseWriter struct {
	http.ResponseWriter
}

func (w *CacheHeaderResponseWriter) WriteHeader(statusCode int) {
	if statusCode >= 200 && statusCode < 300 {
		// Force cache duration for VR client to 1 day
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func ForceShortCacheHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&CacheHeaderResponseWriter{w}, r)
	})
}
