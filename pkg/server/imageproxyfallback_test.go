package server

import "testing"

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
