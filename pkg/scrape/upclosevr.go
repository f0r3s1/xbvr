package scrape

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func UpCloseVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	// this scraper is non-standard in that it gathers info via an api rather than scraping html pages
	defer wg.Done()
	scraperID := "upclosevr"
	siteID := "UpCloseVR"
	logScrapeStart(scraperID, siteID)

	siteCollector := createCollector("www.upclosevr.com")

	siteCollector.OnHTML(`script`, func(e *colly.HTMLElement) {
		apiKeyRegex := regexp.MustCompile(`"apiKey":"(.+)"}},"site`)
		applicationIDRegex := regexp.MustCompile(`"applicationID":"(.+)","apiKey`)
		apiKey := apiKeyRegex.FindStringSubmatch(e.Text)
		applicationID := applicationIDRegex.FindStringSubmatch(e.Text)

		if len(apiKey) > 0 && len(applicationID) > 0 {
			pageTotal := 1
			client := resty.New()

			for page := 0; page < pageTotal; page++ {

				var payloadStr string
				if singleSceneURL != "" {
					tmp := strings.Split(singleSceneURL, "/")
					sceneID := tmp[len(tmp)-1]
					payloadStr = `{"requests":[{"indexName":"all_scenes","params":"clickAnalytics=true&facetFilters=%5B%5B%22availableOnSite%3Aupclosevr%22%5D%2C%5B%22clip_id%3A` + sceneID + `%22%5D%5D&facets=%5B%5D&hitsPerPage=1&tagFilters="}]}`
				} else {
					payloadStr = `{"requests":[{"indexName":"all_scenes_latest_desc","params":"analytics=true&analyticsTags=%5B%22component%3Asearchlisting%22%2C%22section%3Afreetour%22%2C%22site%3Aupclosevr%22%2C%22context%3Avideos%22%2C%22device%3Adesktop%22%5D&clickAnalytics=true&facetingAfterDistinct=true&facets=%5B%22categories.name%22%5D&filters=(upcoming%3A'0')%20AND%20availableOnSite%3Aupclosevr&highlightPostTag=__%2Fais-highlight__&highlightPreTag=__ais-highlight__&hitsPerPage=60&maxValuesPerFacet=1000&page=` + strconv.Itoa(page) + `&query=&tagFilters="}]}`
				}

				var payload = strings.NewReader(payloadStr)
				resp, err := client.R().
					SetHeader("Origin", "https://www.upclosevr.com").
					SetHeader("Referer", "https://www.upclosevr.com/").
					SetHeader("User-Agent", UserAgent).
					SetHeader("x-algolia-api-key", apiKey[1]).
					SetHeader("x-algolia-application-id", applicationID[1]).
					SetBody(payload).
					Post("https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20JavaScript%20(4.22.1)%3B%20Browser%3B%20instantsearch.js%20(4.64.3)%3B%20react%20(18.2.0)%3B%20react-instantsearch%20(7.5.5)%3B%20react-instantsearch-core%20(7.5.5)%3B%20JS%20Helper%20(3.16.2)")

				if err != nil {
					log.Errorln("UpCloseVR encourtned an error on the API Call", err)
					return
				}

				// Convert the resp into a json string for gjson usability
				jsonString := resp.String()

				// Check to see if there are multiple pages of results
				if pageTotal == 1 && singleSceneURL == "" && !limitScraping {
					pageTotal = int(gjson.Get(jsonString, "results.0.nbPages").Int())
				}

				// Make sure we are getting valid response. If the hits array is zero something went wrong
				if len(gjson.Get(jsonString, "results.0.hits").Array()) == 0 {
					log.Errorln("No Results found for UpCloseVR message:", gjson.Get(jsonString, "message").String(), "response code:", gjson.Get(jsonString, "status").String())
				}

				for _, hit := range gjson.Get(jsonString, "results.0.hits").Array() {
					sc := parseUpCloseHit(hit, scraperID, siteID)
					// Check to make sure we don't update scenes we have already collected
					if !funk.ContainsString(knownScenes, sc.HomepageURL) || singleSceneURL != "" {
						log.Infoln(`Scraping ` + sc.Title)
						out <- sc
					}
				}
			}

			if singleSceneURL == "" && config.Config.Web.ShowUpcoming {
				payload := `{"requests":[{"indexName":"all_scenes_latest_desc","params":"filters=(upcoming%3A'1')%20AND%20availableOnSite%3Aupclosevr&hitsPerPage=60"}]}`
				resp, err := client.R().
					SetHeader("Origin", "https://www.upclosevr.com").
					SetHeader("User-Agent", UserAgent).
					SetHeader("x-algolia-api-key", apiKey[1]).
					SetHeader("x-algolia-application-id", applicationID[1]).
					SetBody(payload).
					Post("https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries")
				if err != nil {
					log.Debugf("%s: upcoming scenes not available: %v", siteID, err)
				} else {
					emitUpcoming(scraperID, parseUpCloseUpcoming(resp.String(), scraperID, siteID), out)
				}
			}
		}
	})

	siteCollector.Visit("https://www.upclosevr.com/en/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

// parseUpCloseUpcoming reads announced scenes from an Algolia response; they share the clip_id of the later release.
func parseUpCloseUpcoming(body string, scraperID string, siteID string) []models.ScrapedScene {
	var scenes []models.ScrapedScene
	for _, hit := range gjson.Get(body, "results.0.hits").Array() {
		sc := parseUpCloseHit(hit, scraperID, siteID)
		sc.Upcoming = true
		scenes = append(scenes, sc)
	}
	return scenes
}

func parseUpCloseHit(hit gjson.Result, scraperID string, siteID string) models.ScrapedScene {
	sc := models.ScrapedScene{}
	sc.ScraperID = scraperID
	sc.SceneType = "VR"
	sc.Studio = siteID
	sc.Site = siteID
	sc.SiteID = hit.Get("clip_id").String()
	sc.HomepageURL = `https://www.upclosevr.com/en/video/upclosevr/` + hit.Get("url_title").String() + `/` + sc.SiteID
	sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

	tmpDate, _ := goment.New(hit.Get("release_date").String(), "YYYY-MM-DD")
	sc.Released = tmpDate.Format("YYYY-MM-DD")
	sc.Covers = append(sc.Covers, `https://transform.gammacdn.com/movies/`+hit.Get("pictures.1920x1080").String())
	sc.Synopsis = strings.TrimSpace(strings.Replace(hit.Get("description").String(), "</br></br>", " ", -1))
	sc.Title = strings.TrimSpace(hit.Get("title").String())

	// Cast - Females Only can be update to include males if wanted
	sc.ActorDetails = make(map[string]models.ActorDetails)
	for _, actor := range hit.Get("female_actors").Array() {
		name := actor.Get("name").String()
		sc.Cast = append(sc.Cast, name)
		sc.ActorDetails[name] = models.ActorDetails{
			Source:     scraperID + " scrape",
			ProfileUrl: `https://www.upclosevr.com/en/pornstar/view/` + actor.Get("url_name").String() + `/` + actor.Get("actor_id").String(),
		}
	}

	// Junk Tags we don't want to add to scene data
	skiptags := map[string]bool{
		"Original Series":     true,
		"Adult Time Original": true,
	}
	for _, name := range hit.Get("categories.#.name").Array() {
		if !skiptags[name.String()] {
			sc.Tags = append(sc.Tags, name.String())
		}
	}

	// Duration is in total seconds
	sc.Duration = int(hit.Get("length").Int()) / 60
	return sc
}

func init() {
	registerScraper("upclosevr", "Up Close VR", "https://static01-cms-fame.gammacdn.com/upclosevr/m/3ixx4xg65im880g8/UpClose-VR_Favicon_114x114.png", "upclosevr.com", UpCloseVR)
}
