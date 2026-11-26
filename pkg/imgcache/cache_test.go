package imgcache

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/peterbourgon/diskv"
)

func newTestCache(t *testing.T, memBytes int64) (*Cache, *diskv.Diskv) {
	d := diskv.New(diskv.Options{
		BasePath:  t.TempDir(),
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return New(d, memBytes), d
}

func countFiles(t *testing.T, dir string) (n int) {
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			n++
		}
		return nil
	})
	return
}

func TestOriginalsStayInMemory(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	c.Set("http://x/a.jpg", []byte("orig"))
	if b, ok := c.Get("http://x/a.jpg"); !ok || string(b) != "orig" {
		t.Fatal("original not served from memory")
	}
	if n := countFiles(t, d.BasePath); n != 0 {
		t.Fatalf("original written to disk (%d files)", n)
	}
}

func TestVariantsGoToDisk(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	c.Set("http://x/a.jpg#200x", []byte("var"))
	if n := countFiles(t, d.BasePath); n != 1 {
		t.Fatalf("want 1 disk file, got %d", n)
	}
	c2 := New(d, 1<<20) // fresh memory, same disk
	if b, ok := c2.Get("http://x/a.jpg#200x"); !ok || string(b) != "var" {
		t.Fatal("variant not read back from disk")
	}
	c.Delete("http://x/a.jpg#200x")
	if _, ok := c.Get("http://x/a.jpg#200x"); ok {
		t.Fatal("variant still present after delete")
	}
}

func TestLegacyEntriesStillRead(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	// legacy layout: plain md5 name, for both originals and variants
	d.Write(hashName("http://x/a.jpg"), []byte("old original"))
	d.Write(hashName("http://x/a.jpg#700x"), []byte("old variant"))

	if b, ok := c.Get("http://x/a.jpg"); !ok || string(b) != "old original" {
		t.Fatal("legacy original not read")
	}
	if b, ok := c.Get("http://x/a.jpg#700x"); !ok || string(b) != "old variant" {
		t.Fatal("legacy variant not read")
	}
	if _, err := os.Stat(c.path(hashName("http://x/a.jpg#700x") + VariantSuffix)); err != nil {
		t.Fatal("legacy variant was not promoted")
	}
	if _, ok := c.Get("http://x/a.jpg#700x"); !ok {
		t.Fatal("promoted variant not readable")
	}
}

func TestMemoryBound(t *testing.T) {
	c, _ := newTestCache(t, 10)
	c.Set("a", []byte("123456"))
	c.Set("b", []byte("123456"))
	if _, ok := c.Get("a"); ok {
		t.Fatal("oldest entry not evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("newest entry missing")
	}
	c.Set("big", bytes.Repeat([]byte("x"), 11))
	if _, ok := c.Get("big"); ok {
		t.Fatal("oversized entry stored")
	}
}

func TestEvictOrder(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	write := func(name string, age time.Duration) {
		p := filepath.Join(dir, name)
		os.WriteFile(p, bytes.Repeat([]byte("x"), 10), 0o644)
		os.Chtimes(p, now.Add(-age), now.Add(-age))
	}
	write("origNew", time.Hour)
	write("origOld", 3*time.Hour)
	write("varOld"+VariantSuffix, 5*time.Hour)
	write("varNew"+VariantSuffix, time.Minute)

	if got := Evict(dir, 100, 0, true); got != 0 {
		t.Fatalf("under cap, removed %d", got)
	}
	// 40 bytes, cap 25: originals go first (oldest first), then stop
	Evict(dir, 25, 0, true)
	if _, err := os.Stat(filepath.Join(dir, "origOld")); err == nil {
		t.Fatal("old original kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "origNew")); err == nil {
		t.Fatal("new original kept before variants")
	}
	if _, err := os.Stat(filepath.Join(dir, "varOld"+VariantSuffix)); err != nil {
		t.Fatal("variant removed while under cap")
	}
	// cap 10: the older variant goes before the newer one
	Evict(dir, 10, 0, true)
	if _, err := os.Stat(filepath.Join(dir, "varOld"+VariantSuffix)); err == nil {
		t.Fatal("older variant kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "varNew"+VariantSuffix)); err != nil {
		t.Fatal("newer variant evicted")
	}
}

func TestQ80AliasesLegacyKey(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	d.Write(hashName("http://x/a.jpg#700x"), []byte("old variant"))

	b, ok := c.Get("http://x/a.jpg#700x,q80")
	if !ok || string(b) != "old variant" {
		t.Fatal("legacy variant not found through q80 key")
	}
	if _, err := os.Stat(c.path(hashName("http://x/a.jpg#700x,q80") + VariantSuffix)); err != nil {
		t.Fatal("alias not copied to the new key")
	}
	if _, ok := c.Get("http://x/b.jpg#700x,q80"); ok {
		t.Fatal("unexpected hit")
	}
}

func TestEvictKeepsLegacyWhenDeferred(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "legacy"), bytes.Repeat([]byte("x"), 10), 0o644)
	os.WriteFile(filepath.Join(dir, "a"+VariantSuffix), bytes.Repeat([]byte("x"), 10), 0o644)

	Evict(dir, 0, 0, false)
	if _, err := os.Stat(filepath.Join(dir, "legacy")); err != nil {
		t.Fatal("legacy file evicted while deferred")
	}
	if _, err := os.Stat(filepath.Join(dir, "a"+VariantSuffix)); err == nil {
		t.Fatal("variant not evicted")
	}
	Evict(dir, 0, 0, true)
	if _, err := os.Stat(filepath.Join(dir, "legacy")); err == nil {
		t.Fatal("legacy file kept after deferral")
	}
}

type fixedBody struct{ body []byte }

func (f fixedBody) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(f.body)), Request: req}, nil
}

func TestNoTIFF(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://x/a.jpg", nil)
	resp, _ := (&NoTIFF{Next: fixedBody{[]byte("II*\x00rest")}}).RoundTrip(req)
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("tiff passed with %d", resp.StatusCode)
	}
	jpeg := []byte("\xff\xd8\xff\xe0jpegdata")
	resp, _ = (&NoTIFF{Next: fixedBody{jpeg}}).RoundTrip(req)
	got, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !bytes.Equal(got, jpeg) {
		t.Fatalf("jpeg altered: %d %q", resp.StatusCode, got)
	}
}

func testJPEG(t *testing.T, w, h int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestScaleToWidth(t *testing.T) {
	small := testJPEG(t, 500, 300)
	if out, err := scaleTo(small, 700, 0, 80); err != nil || !bytes.Equal(out, small) {
		t.Fatal("narrow jpeg should pass through unchanged")
	}
	out, err := scaleTo(testJPEG(t, 1600, 900), 700, 0, 80)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 700 || cfg.Height != 393 {
		t.Fatalf("scaled to %dx%d (%v)", cfg.Width, cfg.Height, err)
	}
	var png bytes.Buffer
	pngEncode(&png, image.NewRGBA(image.Rect(0, 0, 1600, 900)))
	if _, err := scaleTo(png.Bytes(), 700, 0, 80); err == nil {
		t.Fatal("png should be left to imageproxy")
	}
}

var pngEncode = png.Encode

type countingTransport struct {
	body  []byte
	calls int32
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&c.calls, 1)
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(c.body)), Request: req}, nil
}

func cachedDump(t *testing.T, body []byte) []byte {
	resp := &http.Response{StatusCode: 200, ProtoMajor: 1, ProtoMinor: 1, Header: http.Header{"Content-Type": {"image/jpeg"}}, ContentLength: int64(len(body)), Body: io.NopCloser(bytes.NewReader(body))}
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatal(err)
	}
	return dump
}

func TestPassthroughReusesLargerCachedCover(t *testing.T) {
	c, _ := newTestCache(t, 1<<20)
	c.Set("http://x/a.jpg#1200x0,q80", cachedDump(t, testJPEG(t, 1200, 675)))
	next := &countingTransport{body: testJPEG(t, 1600, 900)}
	p := &Passthrough{Next: next, Cache: c}

	req, _ := http.NewRequest("GET", "http://x/a.jpg#700x0,q80", nil)
	resp, err := p.RoundTrip(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("got %v %v", resp, err)
	}
	cfg, err := jpeg.DecodeConfig(resp.Body)
	if err != nil || cfg.Width != 700 {
		t.Fatalf("width %d (%v)", cfg.Width, err)
	}
	if next.calls != 0 {
		t.Fatalf("original fetched %d times although a larger copy was cached", next.calls)
	}
	if !c.Has("http://x/a.jpg#700x0,q80") {
		t.Fatal("700 px copy not cached")
	}
}

func TestPassthroughHeightOnlyAndCancelledClient(t *testing.T) {
	c, _ := newTestCache(t, 1<<20)
	next := &countingTransport{body: testJPEG(t, 1200, 600)}
	p := &Passthrough{Next: next, Cache: c}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the browser already gave up
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://x/b.jpg#0x85,q80", nil)
	resp, err := p.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := jpeg.DecodeConfig(resp.Body)
	if cfg.Height != 85 {
		t.Fatalf("height %d, want 85", cfg.Height)
	}
	if !c.Has("http://x/b.jpg#0x85,q80") {
		t.Fatal("thumbnail not cached after the client cancelled")
	}
}
