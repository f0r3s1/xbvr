package imgcache

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strconv"

	"golang.org/x/image/draw"
	"willnorris.com/go/imageproxy"
)

const defaultQuality = 80

// Passthrough handles width-only resizes (what cards ask for) itself: an image that is already narrow enough
// is served as is, others are scaled with CatmullRom, which costs far less than imageproxy's Lanczos.
// Decoding uses at most half the CPUs, so the API stays responsive while a page of new covers is processed.
type Passthrough struct {
	Next  http.RoundTripper
	Cache *Cache
}

var decodeSlots = make(chan struct{}, max(1, runtime.NumCPU()/2))

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

	out, err := scaleToWidth(body, int(opt.Width), opt.Quality)
	if err != nil {
		return t.Next.RoundTrip(req) // the original is cached now, imageproxy takes it from there
	}

	resp.Header.Del("X-From-Cache")
	resp.Header.Set("Content-Type", "image/jpeg")
	resp.Header.Set("Content-Length", strconv.Itoa(len(out)))
	resp.ContentLength = int64(len(out))
	resp.Body = io.NopCloser(bytes.NewReader(out))
	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		t.Cache.Set(key, dump)
	}
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.Request = req
	return resp, nil
}

// scaleToWidth returns a JPEG no wider than width: a narrow enough JPEG unchanged, anything else scaled.
func scaleToWidth(body []byte, width int, quality int) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if format == "jpeg" && cfg.Width <= width {
		return body, nil
	}
	if format != "jpeg" && format != "webp" {
		return nil, image.ErrFormat
	}

	decodeSlots <- struct{}{}
	defer func() { <-decodeSlots }()

	src, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	dst := src
	if b := src.Bounds(); b.Dx() > width {
		h := b.Dy() * width / b.Dx()
		scaled := image.NewRGBA(image.Rect(0, 0, width, h))
		draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, draw.Src, nil)
		dst = scaled
	}
	q := quality
	if q <= 0 {
		q = defaultQuality
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: q}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
