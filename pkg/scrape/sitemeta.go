package scrape

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	_ "golang.org/x/image/webp"
)

// SiteMeta is the public branding of a studio. Empty fields mean "not found".
type SiteMeta struct {
	Logo        string
	Description string
	Homepage    string
}

const siteMetaMaxBody = 2 << 20

var boilerplatePhrases = []string{"full vr porn videos", "check out our collection", "best vr"}

// IsBoilerplateDescription flags templated SEO text that names the site next to stock phrases.
func IsBoilerplateDescription(desc, siteName string) bool {
	d := strings.ToLower(desc)
	if !strings.Contains(d, strings.ToLower(strings.TrimSpace(siteName))) {
		return false
	}
	for _, p := range boilerplatePhrases {
		if strings.Contains(d, p) {
			return true
		}
	}
	return false
}

// ParseSiteMeta reads the description and the best icon from a homepage (og:image is a banner, never a logo). base resolves relative URLs.
func ParseSiteMeta(body io.Reader, base *url.URL) SiteMeta {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return SiteMeta{}
	}
	meta := func(attr, key string) string {
		v, _ := doc.Find("meta[" + attr + `="` + key + `"]`).First().Attr("content")
		return strings.TrimSpace(v)
	}
	abs := func(ref string) string {
		u, err := base.Parse(strings.TrimSpace(ref))
		if ref == "" || err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return ""
		}
		return u.String()
	}

	m := SiteMeta{Description: meta("property", "og:description")}
	if m.Description == "" {
		m.Description = meta("name", "description")
	}
	if len(m.Description) > 500 {
		m.Description = m.Description[:500]
	}

	m.Logo = abs(meta("property", "og:logo"))
	if m.Logo == "" {
		best := -1
		doc.Find(`link[rel~="apple-touch-icon"], link[rel~="icon"], link[rel="shortcut icon"]`).Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			sizes, _ := s.Attr("sizes")
			size, _ := strconv.Atoi(strings.SplitN(strings.ToLower(sizes), "x", 2)[0])
			if rel, _ := s.Attr("rel"); strings.Contains(rel, "apple-touch-icon") {
				size += 1 // tie-break towards the touch icon
			}
			if u := abs(href); u != "" && size > best {
				best, m.Logo = size, u
			}
		})
	}
	return m
}

// FetchSiteMeta looks up branding for a scraper: the SLR API for SLR studios, the homepage otherwise.
func FetchSiteMeta(scraperID string) (SiteMeta, error) {
	var pageURL string
	for _, l := range allScraperConfigs() {
		if l.ID == scraperID {
			pageURL = l.URL
			break
		}
	}
	domain := ""
	for _, s := range models.GetScrapers() {
		if s.ID == scraperID {
			domain = s.Domain
		}
	}
	if domain == "" {
		return SiteMeta{}, nil
	}
	if domain == "sexlikereal.com" && pageURL != "" {
		return fetchSLRSiteMeta(pageURL)
	}
	if pageURL == "" {
		pageURL = "https://" + domain
	}
	return fetchHomepageMeta(pageURL, domain)
}

// ValidLogo reports whether logoURL serves a real image: 2xx, image/* content type and a decodable body.
func ValidLogo(logoURL string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, logoURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	ctype := strings.ToLower(resp.Header.Get("Content-Type"))
	if resp.StatusCode < 200 || resp.StatusCode > 299 || !strings.HasPrefix(ctype, "image/") {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, siteMetaMaxBody+1))
	if err != nil || len(body) > siteMetaMaxBody {
		return false
	}
	switch {
	case strings.HasPrefix(ctype, "image/svg+xml"):
		return bytes.Contains(body, []byte("<svg"))
	case bytes.HasPrefix(body, []byte{0, 0, 1, 0}): // ICO, which the image package cannot decode
		return true
	}
	_, _, err = image.DecodeConfig(bytes.NewReader(body))
	return err == nil
}

func allScraperConfigs() []config.ScraperConfig {
	var l config.ScraperList
	l.Load()
	var all []config.ScraperConfig
	for _, g := range [][]config.ScraperConfig{
		l.XbvrScrapers.PovrScrapers, l.XbvrScrapers.SlrScrapers, l.XbvrScrapers.StashDbScrapers, l.XbvrScrapers.RealVRScrapers, l.XbvrScrapers.VrpornScrapers, l.XbvrScrapers.VrphubScrapers,
		l.CustomScrapers.PovrScrapers, l.CustomScrapers.SlrScrapers, l.CustomScrapers.StashDbScrapers, l.CustomScrapers.RealVRScrapers, l.CustomScrapers.VrpornScrapers, l.CustomScrapers.VrphubScrapers,
	} {
		all = append(all, g...)
	}
	return all
}

func siteMetaRequest(domain string) *resty.Request {
	ScraperRateLimiterWait(domain)
	req := resty.New().SetTimeout(10*time.Second).R().SetHeader("User-Agent", UserAgent)
	return SetupRestyRequest(GetCoreDomain(domain)+"-scraper", req)
}

func fetchHomepageMeta(pageURL, domain string) (SiteMeta, error) {
	resp, err := siteMetaRequest(domain).SetDoNotParseResponse(true).Get(pageURL)
	if err != nil {
		return SiteMeta{}, err
	}
	defer resp.RawBody().Close()
	if resp.StatusCode() != 200 {
		return SiteMeta{}, fmt.Errorf("%s returned %d", pageURL, resp.StatusCode())
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return SiteMeta{}, err
	}
	m := ParseSiteMeta(io.LimitReader(resp.RawBody(), siteMetaMaxBody), base)
	m.Homepage = pageURL
	return m, nil
}

// The SLR studio API only carries logo and website; its pages (banner, text) sit behind Cloudflare.
func fetchSLRSiteMeta(studioURL string) (SiteMeta, error) {
	code := slrStudioCode(studioURL)
	if code == "" {
		return SiteMeta{}, nil
	}
	project := "1"
	if strings.Contains(studioURL, "/gay/") {
		project = "4"
	} else if strings.Contains(studioURL, "/trans/") {
		project = "3"
	}
	resp, err := siteMetaRequest("sexlikereal.com").SetHeader("Client-Type", "web").SetHeader("project", project).
		Get("https://api.sexlikereal.com/v3/studios/" + code)
	if err != nil {
		return SiteMeta{}, err
	}
	if resp.StatusCode() != 200 {
		return SiteMeta{}, fmt.Errorf("SLR studio %s returned %d", code, resp.StatusCode())
	}
	d := gjson.GetBytes(resp.Body(), "data")
	m := SiteMeta{Logo: d.Get("logo").String(), Homepage: d.Get("url").String()}
	if m.Logo == "" {
		m.Logo = d.Get("thumbnailUrl").String()
	}
	if m.Homepage == "" {
		m.Homepage = studioURL
	}
	return m, nil
}

func slrStudioCode(studioURL string) string {
	last := studioURL[strings.LastIndex(studioURL, "/")+1:]
	if i := strings.LastIndex(last, "-"); i != -1 {
		if _, err := strconv.Atoi(last[i+1:]); err == nil {
			return last[i+1:]
		}
	}
	if code, ok := studioNameToCode[strings.ToLower(last)]; ok {
		return strconv.Itoa(code)
	}
	return ""
}
