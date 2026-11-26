package imgcache

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"

	"willnorris.com/go/imageproxy"
)

// Passthrough serves and caches the upstream JPEG as is when a width-only resize would not shrink it, instead of re-encoding it.
type Passthrough struct {
	Next  http.RoundTripper
	Cache *Cache
}

func (t *Passthrough) RoundTrip(req *http.Request) (*http.Response, error) {
	opt := imageproxy.ParseOptions(req.URL.Fragment)
	key := req.URL.String()
	widthOnly := opt.Width >= 1 && opt == imageproxy.Options{Width: opt.Width, Quality: opt.Quality}
	if !widthOnly || t.Cache.Has(key) {
		return t.Next.RoundTrip(req)
	}

	orig := req.Clone(req.Context())
	u := *req.URL
	u.Fragment = ""
	orig.URL = &u
	resp, err := t.Next.RoundTrip(orig)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return t.Next.RoundTrip(req)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	cfg, err := jpeg.DecodeConfig(bytes.NewReader(body))
	if err != nil || float64(cfg.Width) > opt.Width {
		return t.Next.RoundTrip(req) // original is now cached in memory
	}

	resp.Header.Del("X-From-Cache")
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp.ContentLength = int64(len(body))
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		t.Cache.Set(key, dump)
	}
	resp.Request = req
	return resp, nil
}
