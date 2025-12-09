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
		sceneCollector = createCollector(domain)
		siteCollector  = createCollector(domain)
		mu             sync.Mutex
		processed      sync.Map
	)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
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
		if storedTitle, exists := processed.Load(e.Request.URL.String() + "_title"); exists {
			sc.Title = storedTitle.(string)
		} else {
			sc.Title = strings.TrimSpace(e.ChildText(`div.video-title-container h1`))
		}

		// Cover image
		if storedCover, exists := processed.Load(e.Request.URL.String()); exists {
			sc.Covers = append(sc.Covers, storedCover.(string))
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
			// Clean CDN resize directives to get original quality image
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

	// Scene discovery
	siteCollector.OnHTML(`div.item-wrapper`, func(e *colly.HTMLElement) {
		// Find scene URL
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

		// Get cover and title
		coverImg := e.ChildAttr("img.cover", "src")
		if coverImg == "" {
			coverImg = e.ChildAttr("img", "src")
		}
		// Clean CDN resize directives from cover image
		coverImg = cleanCDNURL(coverImg)

		title := strings.TrimSpace(e.ChildText("div.title"))
		if title == "" {
			title = strings.TrimSpace(e.ChildText(".title"))
		}

		mu.Lock()
		defer mu.Unlock()

		if !funk.ContainsString(knownScenes, sceneURL) {
			if _, exists := processed.Load(sceneURL); !exists {
				processed.Store(sceneURL, coverImg)
				processed.Store(sceneURL+"_title", title)
				sceneCollector.Visit(sceneURL)
			}
		}
	})

	// Pagination - only if not limit scraping
	if !limitScraping {
		siteCollector.OnHTML(`#video-section`, func(e *colly.HTMLElement) {
			if e.ChildText(`div.data-notfound-message`) != "" {
				return
			}

			pageNum := 1
			if page := e.Request.URL.Query().Get("page"); page != "" {
				pageNum, _ = strconv.Atoi(page)
			}

			nextPage := fmt.Sprintf("%s/videos?sort=new&page=%d", baseURL, pageNum+1)

			mu.Lock()
			defer mu.Unlock()

			if _, exists := processed.Load(nextPage); !exists {
				processed.Store(nextPage, true)
				siteCollector.Visit(nextPage)
			}
		})
	}

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(baseURL + "/videos")
	}

	siteCollector.Wait()
	sceneCollector.Wait()

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper(scraperID, siteID, baseURL+"/favicon.ico", domain, VRSpy)
}
