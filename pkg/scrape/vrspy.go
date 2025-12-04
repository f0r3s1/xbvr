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

func VRSpy(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singleScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	var (
		flareWG        sync.WaitGroup
		sceneCollector = createCollector(domain)
		siteCollector  = createCollector(domain)
		mu             sync.Mutex
		processed      sync.Map
	)

	sceneCollector.OnScraped(func(r *colly.Response) {
		log.Infof("🏁 Finished processing scene: %s", r.Request.URL)
	})

	siteCollector.OnScraped(func(r *colly.Response) {
		log.Infof("🏁 Finished processing listing: %s", r.Request.URL)
	})

	trackRequests := func(c *colly.Collector) {
		c.OnRequest(func(r *colly.Request) {
			flareWG.Add(1)
			log.Infof("🌐 Starting request: %s", r.URL)
		})
		c.OnResponse(func(r *colly.Response) {
			defer flareWG.Done()
			log.Infof("✅ Received %d bytes from %s", len(r.Body), r.Request.URL)
		})
		c.OnError(func(r *colly.Response, err error) {
			defer flareWG.Done()
			log.Errorf("❌ Error fetching %s: %v", r.Request.URL, err)
		})
	}

	trackRequests(sceneCollector)
	trackRequests(siteCollector)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		log.Infof("🔍 Processing scene page: %s", e.Request.URL)

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
			log.Warnf("❌ Could not determine SiteID for %s", sc.HomepageURL)
			return
		}
		sc.SceneID = scraperID + "-" + sc.SiteID

		// Title - Use the stored title from listing page if available
		if storedTitle, exists := processed.Load(e.Request.URL.String() + "_title"); exists {
			sc.Title = storedTitle.(string)
		} else {
			sc.Title = strings.TrimSpace(e.ChildText(`div.video-title-container h1`))
		}

		// Cover image - Use the stored cover from listing page
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

			// Check for duration
			if strings.Contains(infoText, "Duration") {
				durationText := e.ChildText("span")
				if durationText == "" && strings.Contains(infoText, ":") {
					durationText = strings.TrimSpace(strings.Split(infoText, ":")[1])
				}

				if durationText != "" {
					// Simplified duration extraction
					parts := strings.Split(durationText, ":")
					var hours, mins, secs int
					if len(parts) == 3 {
						hours, _ = strconv.Atoi(parts[0])
						mins, _ = strconv.Atoi(parts[1])
						secs, _ = strconv.Atoi(parts[2])
					} else if len(parts) == 2 {
						mins, _ = strconv.Atoi(parts[0])
						secs, _ = strconv.Atoi(parts[1])
					}
					sc.Duration = (hours*3600 + mins*60 + secs) / 60
				}
			}
		})

		// Date & Duration
		e.ForEach(`div.video-details-info-items div.video-details-info-item`, func(id int, e *colly.HTMLElement) {
			label := strings.TrimSpace(e.Text)
			if strings.Contains(label, "Release date:") {
				dateStr := strings.TrimSpace(e.ChildText("span"))
				if tmpDate, err := goment.New(dateStr, "DD MMMM YYYY"); err == nil {
					sc.Released = tmpDate.Format("YYYY-MM-DD")
				}
			}
			if strings.Contains(label, "Duration:") {
				durationStr := strings.TrimSpace(e.ChildText("span"))
				parts := strings.Split(durationStr, ":")
				if len(parts) == 3 {
					hours, _ := strconv.Atoi(parts[0])
					minutes, _ := strconv.Atoi(parts[1])
					// We're ignoring seconds since the Duration field is in minutes
					sc.Duration = hours*60 + minutes
				}
			}
		})

		// Gallery
		e.ForEach(`div.video-gallery img.thumbnail-cover`, func(id int, e *colly.HTMLElement) {
			imgURL := e.Request.AbsoluteURL(e.Attr("src"))
			// Remove width parameter and add webp format
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

	// Scene discovery with thumbnail and title capture
	siteCollector.OnHTML(`div.item-wrapper`, func(e *colly.HTMLElement) {
		sceneURL := e.ChildAttr("div.item div.photo a.photo-preview", "href")
		if !strings.HasPrefix(sceneURL, "/video/") {
			return
		}
		sceneURL = e.Request.AbsoluteURL(sceneURL)
		coverImg := e.ChildAttr("div.item div.photo a.photo-preview img.cover", "src")
		title := strings.TrimSpace(e.ChildText("div.item div.info.info--grid div.top a div.title"))

		mu.Lock()
		defer mu.Unlock()

		if !funk.ContainsString(knownScenes, sceneURL) {
			if _, exists := processed.Load(sceneURL); !exists {
				// Store both cover image and title from list page
				processed.Store(sceneURL, coverImg)
				processed.Store(sceneURL+"_title", title)
				log.Infof("🎬 Found scene: %s (Cover: %s, Title: %s)", sceneURL, coverImg, title)
				sceneCollector.Visit(sceneURL)
			}
		}
	})

	siteCollector.OnHTML(`#video-section`, func(e *colly.HTMLElement) {
		// Check if we have a "no videos found" message
		if e.ChildText(`div.data-notfound-message`) != "" {
			return // Stop if no more videos
		}

		pageNum := 1

		// Extract current page number if it exists
		if page := e.Request.URL.Query().Get("page"); page != "" {
			pageNum, _ = strconv.Atoi(page)
		}

		// Construct next page URL
		nextPage := fmt.Sprintf("%s/videos?sort=new&page=%d", baseURL, pageNum+1)

		mu.Lock()
		defer mu.Unlock()

		if _, exists := processed.Load(nextPage); !exists && !limitScraping {
			processed.Store(nextPage, true)
			log.Infof("⏭️ Trying next page: %s", nextPage)
			siteCollector.Visit(nextPage)
		}
	})

	if singleSceneURL != "" {
		processed.Store(singleSceneURL, true)
		sceneCollector.Visit(singleSceneURL)
	} else {
		initialPage := baseURL + "/videos"
		processed.Store(initialPage, true)
		siteCollector.Visit(initialPage)
	}

	// Proper synchronization
	siteCollector.Wait()
	sceneCollector.Wait()
	flareWG.Wait()

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	RegisterFlareSolverrSite("vrspy.com")
	registerScraper(scraperID, siteID, baseURL+"/favicon.ico", domain, VRSpy)
}
