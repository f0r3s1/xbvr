package scrape

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"golang.org/x/net/html"
)

var log = &common.Log

var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"

// DomainToSiteID maps domain to site ID for FlareSolverr lookup
var DomainToSiteID = make(map[string]string)

// RegisterDomainSiteMapping maps a domain to a site ID
func RegisterDomainSiteMapping(domain string, siteID string) {
	DomainToSiteID[domain] = siteID
}

// IsFlareSolverrEnabled checks if FlareSolverr is enabled for a domain
func IsFlareSolverrEnabled(domain string) bool {
	// Check if FlareSolverr address is configured
	if config.Config.Advanced.FlareSolverrAddress == "" {
		return false
	}
	// Get site ID from domain mapping
	siteID, exists := DomainToSiteID[domain]
	if !exists {
		return false
	}
	// Check per-site setting in database
	db, _ := models.GetDB()
	defer db.Close()
	var site models.Site
	if err := db.Where("id = ?", siteID).First(&site).Error; err != nil {
		return false
	}
	return site.UseFlareSolverr
}

func createCollector(domains ...string) *colly.Collector {
	// Check if any domain has FlareSolverr enabled
	for _, domain := range domains {
		if IsFlareSolverrEnabled(domain) {
			log.Debugf("Using FlareSolverr for domain: %s", domain)
			return createFlareSolverrCollector(domains...)
		}
	}

	return createBasicCollector(domains...)
}

// createBasicCollector creates a collector without FlareSolverr (for API-based scrapers)
func createBasicCollector(domains ...string) *colly.Collector {
	// Expand domains to include www. variants
	expandedDomains := make([]string, 0, len(domains)*2)
	for _, domain := range domains {
		expandedDomains = append(expandedDomains, domain)
		if !strings.HasPrefix(domain, "www.") {
			expandedDomains = append(expandedDomains, "www."+domain)
		}
	}

	c := colly.NewCollector(
		colly.AllowedDomains(expandedDomains...),
		colly.CacheDir(getScrapeCacheDir()),
		colly.UserAgent(UserAgent),
	)
	// use proxy if configured
	if config.Config.Advanced.ScraperProxy != "" {
		common.Log.Infof("Using proxy for scraping: %s.", config.Config.Advanced.ScraperProxy)
		c.SetProxy(config.Config.Advanced.ScraperProxy)
	}

	c.OnError(func(r *colly.Response, err error) {
		log.Errorf("Error visiting %s %s", r.Request.URL, err)
	})

	c = createCallbacks(c)
	c = setRateLimits(c, domains...)

	return c
}

func setRateLimits(c *colly.Collector, domains ...string) *colly.Collector {
	if Limiters == nil {
		LoadScraperRateLimits()
	}

	for _, domain := range domains {
		SetupCollector(GetCoreDomain(domain)+"-scraper", c)
		log.Debugf("Using Header/Cookies from %s", GetCoreDomain(domain)+"-scraper")
		limiter := GetRateLimiter(domain)
		if limiter != nil {
			randomDelay := limiter.maxDelay - limiter.minDelay
			delay := limiter.minDelay
			c.Limit(&colly.LimitRule{
				DomainGlob:  "*",
				Delay:       delay,       // Delay between requests to domains matching the glob
				RandomDelay: randomDelay, // Max additional random delay added to the delay
			})
			break
		}
	}
	return c
}

func cloneCollector(c *colly.Collector) *colly.Collector {
	x := c.Clone()
	x = createCallbacks(x)
	return x
}

func createCallbacks(c *colly.Collector) *colly.Collector {
	const maxRetries = 15

	c.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")

		if attempt == nil {
			r.Ctx.Put("attempt", 1)
		}

		log.Infoln("visiting", r.URL.String())
	})

	c.OnResponse(func(r *colly.Response) {
		log.Debugf("Response from %s: %d bytes, status %d", r.Request.URL, len(r.Body), r.StatusCode)
	})

	c.OnError(func(r *colly.Response, err error) {
		attempt := r.Ctx.GetAny("attempt").(int)

		if r.StatusCode == 429 {
			log.Errorln("Error:", r.StatusCode, err)

			if attempt <= maxRetries {
				unCache(r.Request.URL.String(), c.CacheDir)
				log.Errorln("Waiting 2 seconds before next request...")
				r.Ctx.Put("attempt", attempt+1)
				time.Sleep(2 * time.Second)
				r.Request.Retry()
			}
		}
	})

	return c
}

func DeleteScrapeCache() error {
	return os.RemoveAll(getScrapeCacheDir())
}

func getScrapeCacheDir() string {
	return common.ScrapeCacheDir
}

func registerScraper(id string, name string, avatarURL string, domain string, f models.ScraperFunc) {
	models.RegisterScraper(id, name, avatarURL, domain, f, "")
	// Register domain to site ID mapping for FlareSolverr lookup
	if domain != "" {
		RegisterDomainSiteMapping(domain, id)
	}
}

func registerAlternateScraper(id string, name string, avatarURL string, domain string, masterSiteId string, f models.ScraperFunc) {
	// alternate scrapers are to scrape scenes available at other sites to match against a scenes from the studio's site, eg scrape VRHush scenes from SLR and match to scenes from VRHush
	models.RegisterScraper(id, name, avatarURL, domain, f, masterSiteId)
	// Register domain to site ID mapping for FlareSolverr lookup
	if domain != "" {
		RegisterDomainSiteMapping(domain, id)
	}
}

func logScrapeStart(id string, name string) {
	log.WithFields(logrus.Fields{
		"task":      "scraperProgress",
		"scraperID": id,
		"progress":  0,
		"started":   true,
		"completed": false,
	}).Infof("Starting %v scraper", name)
}

func logScrapeFinished(id string, name string) {
	log.WithFields(logrus.Fields{
		"task":      "scraperProgress",
		"scraperID": id,
		"progress":  0,
		"started":   false,
		"completed": true,
	}).Infof("Finished %v scraper", name)
}

func unCache(URL string, cacheDir string) {
	sum := sha1.Sum([]byte(URL))
	hash := hex.EncodeToString(sum[:])
	dir := path.Join(cacheDir, hash[:2])
	filename := path.Join(dir, hash)
	if err := os.Remove(filename); err != nil {
		log.Fatal(err)
	}
}

func updateSiteLastUpdate(id string) {
	var site models.Site
	err := site.GetIfExist(id)
	if err != nil {
		log.Error(err)
		return
	}
	site.LastUpdate = time.Now()
	site.Save()
}

func traverseNodes(node *html.Node, fn func(*html.Node)) {
	if node == nil {
		return
	}

	fn(node)

	for cur := node.FirstChild; cur != nil; cur = cur.NextSibling {
		traverseNodes(cur, fn)
	}
}

func findComments(sel *goquery.Selection) []string {
	comments := []string{}
	for _, node := range sel.Nodes {
		traverseNodes(node, func(node *html.Node) {
			if node.Type == html.CommentNode {
				comments = append(comments, node.Data)
			}
		})
	}
	return comments
}

func getFilenameFromURL(u string) string {
	p, _ := url.Parse(u)
	return path.Base(p.Path)
}

func getTextFromHTMLWithSelector(data string, sel string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(doc.Find(sel).Text())
}

func CreateCollector(domains ...string) *colly.Collector {
	return createCollector(domains...)
}

func GetCoreDomain(domain string) string {
	if strings.HasPrefix(domain, "http") {
		parsedURL, _ := url.Parse(domain)
		domain = parsedURL.Hostname()
	}
	parts := strings.Split(domain, ".")
	if len(parts) > 2 && parts[0] == "www" {
		parts = parts[1:]
	}

	return strings.Join(parts[:len(parts)-1], ".")
}
