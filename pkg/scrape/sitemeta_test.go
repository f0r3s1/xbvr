package scrape

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const siteMetaFixture = `<html><head>
<meta property="og:image" content="/img/banner.jpg">
<meta name="description" content="Plain description">
<meta property="og:description" content="  Studio of VR films  ">
<link rel="icon" sizes="32x32" href="/f32.png">
<link rel="apple-touch-icon" sizes="180x180" href="https://cdn.example.com/a180.png">
<link rel="icon" sizes="192x192" href="/f192.png">
</head></html>`

func TestParseSiteMeta(t *testing.T) {
	base, _ := url.Parse("https://example.com/")
	m := ParseSiteMeta(strings.NewReader(siteMetaFixture), base)
	if m.Description != "Studio of VR films" || m.Logo != "https://example.com/f192.png" {
		t.Fatalf("unexpected meta: %+v", m)
	}
	if !IsBoilerplateDescription("Acme full VR porn videos - check out our collection", "Acme") || IsBoilerplateDescription("Acme makes films", "Acme") {
		t.Fatal("boilerplate detection failed")
	}
	if e := ParseSiteMeta(strings.NewReader("<html></html>"), base); e != (SiteMeta{}) {
		t.Fatalf("expected empty meta, got %+v", e)
	}
}

func TestValidLogo(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/blocked.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<html>Forbidden</html>"))
	})
	mux.HandleFunc("/html.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>ok</html>"))
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	})
	mux.HandleFunc("/fake.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("not an image"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for path, want := range map[string]bool{"/blocked.png": false, "/html.png": false, "/fake.png": false, "/logo.png": true} {
		if got := ValidLogo(srv.URL + path); got != want {
			t.Errorf("%s: got %v want %v", path, got, want)
		}
	}
}
