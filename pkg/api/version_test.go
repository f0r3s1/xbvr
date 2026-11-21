package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShouldNotifyUpdate(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.4.38", "0.4.39", true},
		{"v0.4.38", "0.4.39", true},
		{"0.4.38-next", "0.4.39", true},
		{"0.4.38-12-gdeadbee", "0.4.39", true},
		{"0.4.39", "0.4.39", false},
		{"0.4.39-next", "0.4.39", false},
		{"0.4.39-3-gadfe592", "0.4.39", false},
		{"0.4.40", "0.4.39", false},
		{"0.4.40-next", "0.4.39", false},
		{"CURRENT", "0.4.39", false},
		{"HEAD", "0.4.39", false},
		{"", "0.4.39", false},
		{"adfe592", "0.4.39", false},
		{"0.4.39", "", false},
	}
	for _, tc := range cases {
		got := shouldNotifyUpdate(tc.current, tc.latest)
		if got != tc.want {
			t.Errorf("shouldNotifyUpdate(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestVersionCore(t *testing.T) {
	cases := map[string]string{
		"0.4.39":            "0.4.39",
		"v0.4.39":           "0.4.39",
		"0.4.39-next":       "0.4.39",
		"0.4.39-3-gadfe592": "0.4.39",
		"CURRENT":           "",
		"adfe592":           "",
		"":                  "",
	}
	for in, want := range cases {
		if got := versionCore(in); got != want {
			t.Errorf("versionCore(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLatestReleaseTagIsCached(t *testing.T) {
	hits := 0
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"tag_name":"0.4.41"}`))
	}))
	defer srv.Close()

	origURL := latestReleaseURL
	latestReleaseURL = srv.URL
	latestRelease.tag, latestRelease.checked = "", time.Time{}
	defer func() { latestReleaseURL = origURL; latestRelease.tag, latestRelease.checked = "", time.Time{} }()

	for i := 0; i < 5; i++ {
		if got := latestReleaseTag(); got != "0.4.41" {
			t.Fatalf("call %d: got %q, want 0.4.41", i, got)
		}
	}
	if hits != 1 {
		t.Fatalf("expected one upstream request, got %d", hits)
	}

	// Once the cache expires a failing upstream keeps the last good tag.
	status = http.StatusForbidden
	latestRelease.checked = time.Now().Add(-2 * latestReleaseTTL)
	if got := latestReleaseTag(); got != "0.4.41" {
		t.Fatalf("after upstream failure got %q, want last good 0.4.41", got)
	}
	if hits != 2 {
		t.Fatalf("expected a retry request after expiry, got %d hits", hits)
	}
}
