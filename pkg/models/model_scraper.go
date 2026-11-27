package models

import (
	"encoding/json"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

var (
	scrapers   []Scraper
	scrapersMu sync.RWMutex
)

type ScraperFunc func(*ScrapeWG, bool, []string, chan<- ScrapedScene, string, string, bool) error

type Scraper struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	AvatarURL    string      `json:"avatar_url"`
	Domain       string      `json:"domain"`
	Scrape       ScraperFunc `json:"-"`
	MasterSiteId string      `json:"master_site_id"`
}

type ScrapedScene struct {
	SceneID           string   `json:"_id"`
	ScraperID         string   `json:"xbvr_site"`
	SiteID            string   `json:"scene_id"`
	SceneType         string   `json:"scene_type"`
	Title             string   `json:"title"`
	Studio            string   `json:"studio"`
	Site              string   `json:"site"`
	Covers            []string `json:"covers"`
	Gallery           []string `json:"gallery"`
	Tags              []string `json:"tags"`
	Cast              []string `json:"cast"`
	Filenames         []string `json:"filename"`
	Duration          int      `json:"duration"`
	Synopsis          string   `json:"synopsis"`
	Released          string   `json:"released"`
	HomepageURL       string   `json:"homepage_url"`
	MembersUrl        string   `json:"members_url"`
	TrailerType       string   `json:"trailer_type"`
	TrailerSrc        string   `json:"trailer_source"`
	ChromaKey         string   `json:"chromakey"`
	HasScriptDownload bool     `json:"has_script_Download"`
	AiScript          bool     `json:"ai_script"`
	HumanScript       bool     `json:"human_script"`
	Upcoming          bool     `json:"upcoming"` // announced by the site, not released yet

	OnlyUpdateScriptData bool `default:"false" json:"only_update_script_data"`
	InternalSceneId      uint `json:"internal_id"`

	ActorDetails map[string]ActorDetails `json:"actor_details"`
	MasterSiteId string                  `json:"master_site_id"`
	Timestamps   string                  `json:"timestamps"`
}

type ActorDetails struct {
	ImageUrl   string
	ProfileUrl string
	Source     string
	StashData  string
}
type TrailerScrape struct {
	SceneUrl       string `json:"scene_url"`        // url of the page to be scrapped
	HtmlElement    string `json:"html_element"`     // path to section of html (using colly)
	ExtractRegex   string `json:"extract_regex"`    // regex expression to extract the json, eg from a json variable assignment in javascript
	ContentBaseUrl string `json:"content_base_url"` // prefix for the url if the scrapped content urls are not abosolute
	RecordPath     string `json:"record_path"`      // points to a json array of video source (optional, there maybe a single video), uses jsonpath syntax
	ContentPath    string `json:"content_path"`     // points to the content url uses jsonpath syntax
	EncodingPath   string `json:"encoding_path"`    // optional, points to the encoding for the source using jsonpath syntax, eg h264, h265
	QualityPath    string `json:"quality_path"`     // points to the quality using jsonpath syntax, eg 1440p, 5k
	KVHttpConfig   string `json:"kv_http_config"`
}

func (s *ScrapedScene) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *ScrapedScene) Log() error {
	j, err := json.MarshalIndent(s, "", "  ")
	log.Debugf("\n%v", string(j))
	return err
}

func GetScrapers() []Scraper {
	scrapersMu.RLock()
	defer scrapersMu.RUnlock()
	return slices.Clone(scrapers)
}

// RegisterScraper adds a scraper, or replaces the one with the same id (custom sites can be edited while running).
func RegisterScraper(id string, name string, avatarURL string, domain string, f ScraperFunc, masterSiteId string) {
	s := Scraper{ID: id, Name: name, AvatarURL: avatarURL, Domain: domain, Scrape: f, MasterSiteId: masterSiteId}
	scrapersMu.Lock()
	defer scrapersMu.Unlock()
	for i := range scrapers {
		if scrapers[i].ID == id {
			scrapers[i] = s
			return
		}
	}
	scrapers = append(scrapers, s)
}

func UnregisterScraper(id string) {
	scrapersMu.Lock()
	defer scrapersMu.Unlock()
	scrapers = slices.DeleteFunc(scrapers, func(s Scraper) bool { return s.ID == id })
}

// Custom wg functions, to allow access to the current count of waitgroups. This allows running scrapers at max count always
type ScrapeWG struct {
	count int64
}

func (wg *ScrapeWG) Add(n int64) {
	atomic.AddInt64(&wg.count, n)
}

func (wg *ScrapeWG) Done() {
	wg.Add(-1)
	if atomic.LoadInt64(&wg.count) < 0 {
		panic("negative wait group counter")
	}
}

func (wg *ScrapeWG) Wait(n int64) {
	for atomic.LoadInt64(&wg.count) >= n && atomic.LoadInt64(&wg.count) != 0 {
		time.Sleep(10 * time.Millisecond)
	}
}

func (wg *ScrapeWG) Count() int64 {
	return atomic.LoadInt64(&wg.count)
}
