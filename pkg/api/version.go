package api

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"

	"github.com/xbapps/xbvr/pkg/common"
)

// latestReleaseURL is a var so tests can point it at a local server.
var latestReleaseURL = "https://api.github.com/repos/f0r3s1/xbvr/releases/latest"

const (
	latestReleaseTimeout = 5 * time.Second
	latestReleaseTTL     = time.Hour
	latestReleaseRetry   = 10 * time.Minute
)

// The navbar calls version-check on every page load. Hitting GitHub each time
// burns the 60 req/h unauthenticated limit and, without a timeout, a slow or
// rate-limited reply stalled the request past the UI's own timeout so the
// version label never rendered. Cache the answer and bound the lookup.
var latestRelease struct {
	sync.Mutex
	tag     string
	checked time.Time
	ttl     time.Duration
}

// latestReleaseTag returns the newest published release tag, or "" when it
// is unknown. At most one GitHub request is made per TTL window.
func latestReleaseTag() string {
	latestRelease.Lock()
	defer latestRelease.Unlock()

	if !latestRelease.checked.IsZero() && time.Since(latestRelease.checked) < latestRelease.ttl {
		return latestRelease.tag
	}

	latestRelease.checked = time.Now()
	latestRelease.ttl = latestReleaseRetry
	r, err := resty.New().SetTimeout(latestReleaseTimeout).R().
		SetHeader("User-Agent", "XBVR/"+common.CurrentVersion).
		SetHeader("Accept", "application/vnd.github.v3+json").
		Get(latestReleaseURL)
	if err != nil || r.StatusCode() != 200 {
		return latestRelease.tag // keep the last good answer, retry sooner
	}
	if tag := gjson.Get(r.String(), "tag_name").String(); tag != "" {
		latestRelease.tag = tag
		latestRelease.ttl = latestReleaseTTL
	}
	return latestRelease.tag
}

// versionCoreRe captures the leading numeric version, ignoring snapshot
// suffixes such as goreleaser's "{{.Tag}}-next" and git-describe's
// "-N-gHASH", which would otherwise rank older than the tag they were built from.
var versionCoreRe = regexp.MustCompile(`(?i)^v?(\d+(?:\.\d+){1,3})`)

func isDevVersion(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "", "CURRENT", "HEAD", "DEV":
		return true
	default:
		return false
	}
}

func versionCore(v string) string {
	v = strings.TrimSpace(v)
	if isDevVersion(v) {
		return ""
	}
	m := versionCoreRe.FindStringSubmatch(v)
	if m == nil {
		return ""
	}
	return m[1]
}

// shouldNotifyUpdate reports whether current is a released build older than
// latest. Dev sentinels, bare git hashes, and snapshot stamps of the same
// or newer release do not notify.
func shouldNotifyUpdate(current, latest string) bool {
	currentCore := versionCore(current)
	latestCore := versionCore(latest)
	if currentCore == "" || latestCore == "" {
		return false
	}
	return compareDotted(currentCore, latestCore) < 0
}

// compareDotted compares dotted numeric versions ("0.4.40" < "0.4.40.5"); missing parts count as 0.
func compareDotted(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
