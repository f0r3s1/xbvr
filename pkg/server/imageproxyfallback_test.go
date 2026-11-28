package server

import (
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"willnorris.com/go/imageproxy"
)

func TestWithDefaultQuality(t *testing.T) {
	cases := map[string]string{
		"/700x/https:/a/b.jpg":     "/700x,q80/https:/a/b.jpg",
		"/200x,q50/https:/a/b.jpg": "/200x,q50/https:/a/b.jpg",
		"/https:/a/b.jpg":          "/https:/a/b.jpg",
		"/64x/https:/a/x/b.jpg":    "/64x,q80/https:/a/x/b.jpg",
	}
	for in, want := range cases {
		if got := withDefaultQuality(in); got != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
}

func TestServeStoredSkipsProxy(t *testing.T) {
	img := []byte("\xff\xd8\xff\xe0cached-jpeg")
	dump := "HTTP/1.1 200 OK\r\nContent-Type: image/jpeg\r\nContent-Length: " + strconv.Itoa(len(img)) + "\r\n\r\n" + string(img)
	asked := ""
	base, _ := url.Parse("http://127.0.0.1:9999")
	h := &ImageProxyFallbackHandler{
		ImageProxy: &imageproxy.Proxy{DefaultBaseURL: base},
		Stored:     func(key string) ([]byte, bool) { asked = key; return []byte(dump), true },
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/300x/https://example.com/a.jpg", nil))
	if w.Code != 200 || w.Body.String() != string(img) || w.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("got %d %q %v", w.Code, w.Body.String(), w.Header())
	}
	if asked != "https://example.com/a.jpg#300x0,q80" {
		t.Fatalf("looked up %q", asked)
	}
}
