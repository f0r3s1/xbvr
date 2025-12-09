package scrape

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

const (
	scraperID = "vrspy"
	siteID    = "VRSpy"
	domain    = "vrspy.com"
	baseURL   = "https://www." + domain
)

// cleanCDNURL removes Cloudflare and other CDN resize directives from URLs
// to get the original full-quality image
func cleanCDNURL(url string) string {
	// Remove Cloudflare cdn-cgi/image resize directive
	// e.g., https://vrspy.com/cdn-cgi/image/w=480/https://cdn.vrspy.com/...
	// becomes https://cdn.vrspy.com/...
	re := regexp.MustCompile(`https?://[^/]+/cdn-cgi/image/[^/]+/`)
	url = re.ReplaceAllString(url, "")
	return url
}

func VRSpy(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singleScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	var (
		collector = createCollector(domain)
		mu        sync.Mutex
		processed sync.Map
		sceneURLs []string
		sceneMeta = make(map[string]struct{ cover, title string })
	)

	// Handler for scene pages (individual video pages)
	collector.OnHTML(`html`, func(e *colly.HTMLElement) {
		// Only process scene pages, not listing pages
		if !strings.Contains(e.Request.URL.Path, "/video/") {
			return
		}

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = siteID
		sc.Site = siteID
		sc.HomepageURL = e.Request.URL.String()

		// SiteID from meta tag
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			ogimage := e.Attr("content")
			re := regexp.MustCompile(`/videos/(\d+)/`)
			if matches := re.FindStringSubmatch(ogimage); len(matches) > 1 {
				sc.SiteID = matches[1]
			}
		})

		if sc.SiteID == "" {
			return
		}
		sc.SceneID = scraperID + "-" + sc.SiteID

		// Title
		if meta, exists := sceneMeta[e.Request.URL.String()]; exists && meta.title != "" {
			sc.Title = meta.title
		} else {
			sc.Title = strings.TrimSpace(e.ChildText(`div.video-title-container h1`))
		}

		// Cover image
		if meta, exists := sceneMeta[e.Request.URL.String()]; exists && meta.cover != "" {
			sc.Covers = append(sc.Covers, meta.cover)
		}

		// Description
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.video-description-container p`))

		// Tags
		sc.Tags = e.ChildTexts(`div.video-categories a`)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.video-actors div.video-actor-item`, func(id int, e *colly.HTMLElement) {
			actorName := strings.TrimSpace(e.ChildText("span"))
			if actorName != "" {
				sc.Cast = append(sc.Cast, actorName)
				sc.ActorDetails[actorName] = models.ActorDetails{
					Source:     scraperID,
					ProfileUrl: e.Request.AbsoluteURL(e.ChildAttr("a", "href")),
				}
			}
		})

		// Date & Duration
		e.ForEach(`div.video-details-info-items div.video-details-info-item`, func(id int, e *colly.HTMLElement) {
			text := strings.TrimSpace(e.Text)

			if strings.Contains(text, "Release date:") {
				dateStr := strings.TrimSpace(e.ChildText("span"))
				if tmpDate, err := goment.New(dateStr, "DD MMMM YYYY"); err == nil {
					sc.Released = tmpDate.Format("YYYY-MM-DD")
				}
			}

			if strings.Contains(text, "Duration:") {
				durationStr := strings.TrimSpace(e.ChildText("span"))
				parts := strings.Split(durationStr, ":")
				if len(parts) == 3 {
					hours, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
					minutes, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
					if errH == nil && errM == nil {
						sc.Duration = hours*60 + minutes
					}
				} else if len(parts) == 2 {
					minutes, errM := strconv.Atoi(strings.TrimSpace(parts[0]))
					if errM == nil {
						sc.Duration = minutes
					}
				}
			}
		})

		// Gallery
		e.ForEach(`div.video-gallery img.thumbnail-cover`, func(id int, e *colly.HTMLElement) {
			imgURL := e.Request.AbsoluteURL(e.Attr("src"))
			imgURL = cleanCDNURL(imgURL)
			if strings.Contains(imgURL, "?width=") {
				baseURL := strings.Split(imgURL, "?")[0]
				imgURL = baseURL + "?format=webp"
			}
			sc.Gallery = append(sc.Gallery, imgURL)
		})

		// Trailer
		if trailerSrc := e.ChildAttr(`video#preview-player source`, "src"); trailerSrc != "" {
			sc.TrailerType = "direct"
			sc.TrailerSrc = trailerSrc
		}

		out <- sc
	})

	// Scene discovery from listing pages - collect URLs, don't visit yet
	collector.OnHTML(`div.item-wrapper`, func(e *colly.HTMLElement) {
		var sceneURL string
		e.ForEach("a", func(i int, el *colly.HTMLElement) {
			href := el.Attr("href")
			if strings.Contains(href, "/video/") && sceneURL == "" {
				sceneURL = href
			}
		})

		if !strings.HasPrefix(sceneURL, "/video/") {
			return
		}
		sceneURL = e.Request.AbsoluteURL(sceneURL)

		coverImg := e.ChildAttr("img.cover", "src")
		if coverImg == "" {
			coverImg = e.ChildAttr("img", "src")
		}
		coverImg = cleanCDNURL(coverImg)

		title := strings.TrimSpace(e.ChildText("div.title"))
		if title == "" {
			title = strings.TrimSpace(e.ChildText(".title"))
		}

		mu.Lock()
		defer mu.Unlock()

		if !funk.ContainsString(knownScenes, sceneURL) {
			if _, exists := processed.Load(sceneURL); !exists {
				processed.Store(sceneURL, true)
				sceneURLs = append(sceneURLs, sceneURL)
				sceneMeta[sceneURL] = struct{ cover, title string }{coverImg, title}
			}
		}
	})

	// Pagination for listing pages - only if not limit scraping
	// We need to track items found per page to know when to stop
	var lastPageItemCount int

	if !limitScraping {
		collector.OnHTML(`html`, func(e *colly.HTMLElement) {
			// Only process pagination for listing pages
			if !strings.Contains(e.Request.URL.Path, "/videos") {
				return
			}

			// Count items on this page
			itemCount := 0
			e.ForEach(`div.item-wrapper`, func(id int, el *colly.HTMLElement) {
				itemCount++
			})

			// Stop if no items found (we've gone past the last page)
			if itemCount == 0 {
				log.Infof("VRSpy: Page %s has no items, stopping pagination", e.Request.URL.String())
				return
			}

			lastPageItemCount = itemCount

			pageNum := 1
			if page := e.Request.URL.Query().Get("page"); page != "" {
				pageNum, _ = strconv.Atoi(page)
			}

			nextPage := fmt.Sprintf("%s/videos?sort=new&page=%d", baseURL, pageNum+1)

			_, exists := processed.Load(nextPage)
			if !exists {
				processed.Store(nextPage, true)
				log.Infof("VRSpy: Page %d had %d items, queuing next page: %s", pageNum, itemCount, nextPage)
				e.Request.Visit(nextPage)
			}
		})
	}

	// Suppress the lastPageItemCount unused warning
	_ = lastPageItemCount

	if singleSceneURL != "" {
		// Single scene scrape
		if err := collector.Visit(singleSceneURL); err != nil {
			log.Errorf("VRSpy: failed to visit scene URL %s: %v", singleSceneURL, err)
		}
		collector.Wait()
	} else {
		// First, collect all scene URLs from listing pages
		if err := collector.Visit(baseURL + "/videos?sort=new"); err != nil {
			log.Errorf("VRSpy: failed to visit site URL: %v", err)
		}
		collector.Wait()

		// Now visit all collected scene URLs
		log.Infof("VRSpy: collected %d scene URLs, now fetching details", len(sceneURLs))
		for i, sceneURL := range sceneURLs {
			if err := collector.Visit(sceneURL); err != nil {
				log.Errorf("VRSpy: failed to visit scene %s: %v", sceneURL, err)
			}
			// Log progress every 10 scenes
			if (i+1)%10 == 0 {
				log.Infof("VRSpy: processed %d/%d scenes", i+1, len(sceneURLs))
			}
		}
		collector.Wait()
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper(scraperID, siteID, baseURL+"/favicon.ico", domain, VRSpy)
}
