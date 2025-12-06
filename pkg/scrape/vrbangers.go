package scrape

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

// fetchURLWithFlareSolverr fetches a URL using FlareSolverr if enabled for the domain, otherwise uses resty
func fetchURLWithFlareSolverr(url string, domain string) (string, error) {
	if IsFlareSolverrEnabled(domain) {
		return FlareSolverrGet(url)
	}
	r, err := resty.New().R().
		SetHeader("User-Agent", UserAgent).
		Get(url)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}

func VRBangersSite(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	// Extract site name from URL for API
	siteName := strings.TrimSuffix(strings.TrimPrefix(URL, "https://"), "/")
	contentBaseURL := "https://content." + siteName

	// Function to scrape a single scene from API
	scrapeScene := func(slug string) {
		sceneURL := URL + slug + "/"

		// Get scene data from API
		apiURL := contentBaseURL + "/api/content/v1/videos/" + slug
		jsonData, err := fetchURLWithFlareSolverr(apiURL, siteName)

		if err != nil {
			log.Warnf("Failed to fetch scene %s: %v", slug, err)
			return
		}

		if gjson.Get(jsonData, "status.message").String() != "Ok" {
			return
		}

		// Check if scene is published
		sceneStatus := gjson.Get(jsonData, "data.item.status").String()
		if sceneStatus != "published" {
			log.Debugf("Skipping unpublished scene %s (status: %s)", slug, sceneStatus)
			return
		}

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = siteID
		sc.HomepageURL = sceneURL

		// Scene ID - back 8 of the "id" via api response
		fullID := gjson.Get(jsonData, "data.item.id").String()
		if len(fullID) > 15 {
			sc.SiteID = strings.TrimSpace(fullID[15:])
		} else {
			sc.SiteID = fullID
		}
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(gjson.Get(jsonData, "data.item.title").String())

		// Filenames
		baseName := sc.Site + "_" + strings.TrimSpace(gjson.Get(jsonData, "data.item.videoSettings.videoShortName").String()) + "_"
		filenames := []string{"8K_180x180_3dh", "6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}
		for i := range filenames {
			filenames[i] = baseName + filenames[i] + ".mp4"
		}
		sc.Filenames = filenames

		// Date from API - publishedAt is Unix timestamp
		publishedAt := gjson.Get(jsonData, "data.item.publishedAt").Int()
		if publishedAt > 0 {
			t := time.Unix(publishedAt, 0).UTC()
			sc.Released = t.Format("2006-01-02")
		}

		// Duration from API (seconds to minutes)
		apiDuration := gjson.Get(jsonData, "data.item.videoSettings.duration").Int()
		if apiDuration > 0 {
			sc.Duration = int(apiDuration / 60)
		}

		// Cover from API - use poster with L size preview for better quality
		// The poster object has previews array with different sizes like gallery images
		posterURL := gjson.Get(jsonData, "data.item.poster.previews.#(sizeAlias==L).permalink").String()
		if posterURL == "" {
			posterURL = gjson.Get(jsonData, "data.item.poster.previews.#(sizeAlias==XL).permalink").String()
		}
		if posterURL == "" {
			posterURL = gjson.Get(jsonData, "data.item.poster.permalink").String()
		}
		if posterURL == "" {
			// Fallback to heroImg
			posterURL = gjson.Get(jsonData, "data.item.heroImg.previews.#(sizeAlias==L).permalink").String()
		}
		if posterURL == "" {
			posterURL = gjson.Get(jsonData, "data.item.heroImg.permalink").String()
		}
		if posterURL != "" {
			sc.Covers = append(sc.Covers, contentBaseURL+posterURL)
		}

		// Gallery from API - use L size (1400px) for good quality but faster loading
		// XL is 2000px which is overkill, L provides good balance
		galleryImages := gjson.Get(jsonData, "data.item.galleryImages").Array()
		for _, img := range galleryImages {
			// Prefer L size (1400px), fallback to XS (400px) if L not available
			imgURL := img.Get("previews.#(sizeAlias==L).permalink").String()
			if imgURL == "" {
				imgURL = img.Get("previews.#(sizeAlias==XS).permalink").String()
			}
			if imgURL != "" {
				sc.Gallery = append(sc.Gallery, contentBaseURL+imgURL)
			}
		}

		// Synopsis from API - strip HTML tags and unescape entities
		rawDesc := gjson.Get(jsonData, "data.item.description").String()
		htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
		cleanDesc := htmlTagRegex.ReplaceAllString(rawDesc, "")
		sc.Synopsis = strings.TrimSpace(html.UnescapeString(cleanDesc))

		// Tags from API - categories use "name" field
		ignoreTags := map[string]bool{"180 vr": true, "6k vr porn": true, "8k vr porn": true, "4k vr porn": true}
		categories := gjson.Get(jsonData, "data.item.categories").Array()
		for _, cat := range categories {
			tag := strings.ToLower(strings.TrimSpace(cat.Get("name").String()))
			if !ignoreTags[tag] && tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		}

		if scraperID == "vrbgay" {
			sc.Tags = append(sc.Tags, "Gay")
		}

		// Trailer setup
		if scraperID != "vrconk" {
			sc.TrailerType = "load_json"
			params := models.TrailerScrape{SceneUrl: apiURL, RecordPath: "data.item.videoPlayerSources.trailer", ContentPath: "src", QualityPath: "quality"}
			strParam, _ := json.Marshal(params)
			sc.TrailerSrc = string(strParam)
		}

		// Cast from API - models array with "title" field for name
		sc.ActorDetails = make(map[string]models.ActorDetails)
		actors := gjson.Get(jsonData, "data.item.models").Array()
		for _, actor := range actors {
			actorName := strings.TrimSpace(actor.Get("title").String())
			actorSlug := actor.Get("slug").String()
			if actorName != "" {
				sc.Cast = append(sc.Cast, actorName)
				sc.ActorDetails[actorName] = models.ActorDetails{
					Source:     scraperID + " scrape",
					ProfileUrl: URL + "model/" + actorSlug + "/",
				}
			}
		}

		// Validate required fields before sending
		// Check that cover URL actually has a path (not just base URL)
		hasCover := len(sc.Covers) > 0 && len(sc.Covers[0]) > len(contentBaseURL)+1
		if sc.Title == "" || sc.SiteID == "" || !hasCover {
			log.Warnf("Skipping scene %s - missing required data (title=%q, siteID=%q, hasCover=%v)", slug, sc.Title, sc.SiteID, hasCover)
			return
		}

		log.Infof("Scraped scene: %s", sc.Title)
		out <- sc
	}

	if singleSceneURL != "" {
		// Extract slug from URL
		parts := strings.Split(strings.TrimSuffix(singleSceneURL, "/"), "/")
		if len(parts) > 0 {
			slug := parts[len(parts)-1]
			scrapeScene(slug)
		}
	} else {
		// Use API to get video listing
		limit := 1000
		if limitScraping {
			limit = 50
		}

		apiURL := fmt.Sprintf("%s/api/content/v1/videos?page=1&type=videos&sort=latest&show_custom_video=1&bonus-video=1&limit=%d", contentBaseURL, limit)

		jsonResponse, err := fetchURLWithFlareSolverr(apiURL, siteName)

		if err != nil {
			log.Errorf("Failed to fetch video listing from API: %v", err)
		} else {
			items := gjson.Get(jsonResponse, "data.items")
			items.ForEach(func(_, scene gjson.Result) bool {
				slug := scene.Get("slug").String()
				if slug != "" {
					sceneURL := URL + slug + "/"
					if !funk.ContainsString(knownScenes, sceneURL) {
						scrapeScene(slug)
					}
				}
				return true
			})
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VRBangers(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrbangers", "VRBangers", "https://vrbangers.com/", limitScraping)
}
func VRBTrans(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrbtrans", "VRBTrans", "https://vrbtrans.com/", limitScraping)
}
func VRBGay(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrbgay", "VRBGay", "https://vrbgay.com/", limitScraping)
}
func VRConk(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrconk", "VRCONK", "https://vrconk.com/", limitScraping)
}
func BlowVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "blowvr", "BlowVR", "https://blowvr.com/", limitScraping)
}
func ARPorn(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "arporn", "ARPorn", "https://arporn.com/", limitScraping)
}

func init() {
	registerScraper("vrbangers", "VRBangers", "https://vrbangers.com/favicon/apple-touch-icon-144x144.png", "vrbangers.com", VRBangers)
	registerScraper("vrbtrans", "VRBTrans", "https://vrbtrans.com/favicon/apple-touch-icon-144x144.png", "vrbtrans.com", VRBTrans)
	registerScraper("vrbgay", "VRBGay", "https://vrbgay.com/favicon/apple-touch-icon-144x144.png", "vrbgay.com", VRBGay)
	registerScraper("vrconk", "VRCONK", "https://vrconk.com/favicon/apple-touch-icon-144x144.png", "vrconk.com", VRConk)
	registerScraper("blowvr", "BlowVR", "https://blowvr.com/favicon/apple-touch-icon-144x144.png", "blowvr.com", BlowVR)
	registerScraper("arporn", "ARPorn", "https://arporn.com/favicon/apple-touch-icon-144x144.png", "arporn.com", ARPorn)
}
