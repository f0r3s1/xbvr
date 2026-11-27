package imgcache

import (
	"bufio"
	"bytes"
	"context"
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

const (
	defaultQuality = 80
	// cards asked for 1200 px wide covers before 0.4.40.7; those cached copies still serve smaller sizes
	largerCachedWidth = 1200
)

// Passthrough handles width-only resizes (what cards ask for) itself: an image that is already narrow enough
// is served as is, others are scaled with CatmullRom, which costs far less than imageproxy's Lanczos.
// Decoding uses at most half the CPUs, so the API stays responsive while a page of new covers is processed.
type Passthrough struct {
	Next  http.RoundTripper
	Cache *Cache
}

var (
	decodeSlots = make(chan struct{}, max(1, runtime.NumCPU()/2))
	workSlots   = make(chan struct{}, 3)
)

func (t *Passthrough) RoundTrip(req *http.Request) (*http.Response, error) {
	opt := imageproxy.ParseOptions(req.URL.Fragment)
	key := req.URL.String()
	sizeOnly := (opt.Width >= 1) != (opt.Height >= 1) && opt == imageproxy.Options{Width: opt.Width, Height: opt.Height, Quality: opt.Quality}
	if !sizeOnly || t.Cache.Has(key) {
		return t.Next.RoundTrip(req)
	}

	// a request abandoned while waiting is dropped; once started it finishes, so the result is cached for the next visit
	select {
	case workSlots <- struct{}{}:
		defer func() { <-workSlots }()
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	req = req.Clone(context.WithoutCancel(req.Context()))

	source, header, err := t.source(req, opt)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return t.Next.RoundTrip(req)
	}
	out, err := scaleTo(source, int(opt.Width), int(opt.Height), opt.Quality)
	if err != nil {
		return t.Next.RoundTrip(req) // the original is cached now, imageproxy takes it from there
	}

	resp := &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: header, Request: req}
	resp.Header.Del("X-From-Cache")
	resp.Header.Set("Content-Type", "image/jpeg")
	resp.Header.Set("Content-Length", strconv.Itoa(len(out)))
	resp.ContentLength = int64(len(out))
	resp.Body = io.NopCloser(bytes.NewReader(out))
	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		t.Cache.Set(key, dump)
	}
	resp.Body = io.NopCloser(bytes.NewReader(out))
	return resp, nil
}

// source is a larger cached copy of the picture when there is one, else the original (nil when unusable).
func (t *Passthrough) source(req *http.Request, opt imageproxy.Options) ([]byte, http.Header, error) {
	u := *req.URL
	u.Fragment = ""
	if opt.Width > 0 && opt.Width < largerCachedWidth {
		larger := u
		larger.Fragment = imageproxy.Options{Width: largerCachedWidth, Quality: defaultQuality}.String()
		if dump, ok := t.Cache.Get(larger.String()); ok {
			if resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(dump)), nil); err == nil {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err == nil && resp.StatusCode == http.StatusOK {
					return body, resp.Header, nil
				}
			}
		}
	}

	orig := req.Clone(req.Context())
	orig.URL = &u
	resp, err := t.Next.RoundTrip(orig)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return body, resp.Header, nil
}

// scaleTo returns a JPEG fitting the width or the height (one of them is 0): a small enough JPEG unchanged, anything else scaled.
func scaleTo(body []byte, width, height int, quality int) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if width == 0 {
		width = cfg.Width * height / max(cfg.Height, 1)
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
