package tasks

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

const (
	// the size scene and actor cards ask for (XS to M cards); larger cards reuse the original
	warmSize = "700x"
	// pause after each cover fetched from a site, so a site never sees a burst of requests
	warmPause = 750 * time.Millisecond
	// a site refusing this many covers in a row is skipped for the rest of the run
	warmHostFailures    = 5
	warmCacheCheckEvery = 200
)

var warmMu sync.Mutex

// WarmCovers caches card covers one at a time in the background (cached ones cost nothing) and stops before the cache size limit.
func WarmCovers(urls []string) {
	if len(urls) == 0 || !warmMu.TryLock() {
		return
	}
	defer warmMu.Unlock()
	tlog := log.WithField("task", "coverwarm")

	client := &http.Client{Timeout: 60 * time.Second}
	base := fmt.Sprintf("http://127.0.0.1:%v/img/%s/", config.Config.Server.Port, warmSize)
	failures := map[string]int{}
	fetched, failed := 0, 0

	for n, cover := range urls {
		u, err := url.Parse(cover)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || failures[u.Host] >= warmHostFailures {
			continue
		}
		if n%warmCacheCheckEvery == 0 && cacheNearlyFull() {
			tlog.Infof("Image cache nearly full, stopped preparing covers after %d", fetched)
			return
		}

		start := time.Now()
		resp, err := client.Get(base + strings.ReplaceAll(cover, " ", "%20"))
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		if err != nil || resp.StatusCode != http.StatusOK {
			failures[u.Host]++
			failed++
			continue
		}
		failures[u.Host] = 0
		// a cache hit answers in milliseconds and needs no pause; a real fetch went to the site
		if time.Since(start) > 50*time.Millisecond {
			fetched++
			time.Sleep(warmPause)
		}
	}
	if fetched > 0 || failed > 0 {
		tlog.Infof("Prepared %d covers (%d unavailable)", fetched, failed)
	}
}

func cacheNearlyFull() bool {
	maxGB := config.Config.Library.ImageCacheMaxGB
	if maxGB <= 0 {
		return false
	}
	size, err := common.DirSize(common.ImgDir)
	return err == nil && size > int64(maxGB)<<30*9/10
}

// WarmScrapedCovers prepares the covers of scenes a scrape just added or updated.
func WarmScrapedCovers(scenes []models.ScrapedScene) {
	urls := make([]string, 0, len(scenes))
	for _, s := range scenes {
		if len(s.Covers) > 0 {
			urls = append(urls, s.Covers[0])
		}
	}
	WarmCovers(urls)
}

// WarmAllCovers prepares every scene and actor card cover, newest scenes first.
func WarmAllCovers() {
	db, _ := models.GetDB()
	var urls []string
	db.Model(&models.Scene{}).Where("cover_url <> ''").Order("release_date desc").Pluck("cover_url", &urls)
	var actorURLs []string
	db.Model(&models.Actor{}).Where("image_url <> ''").Order("count desc").Pluck("image_url", &actorURLs)
	db.Close()
	WarmCovers(append(urls, actorURLs...))
}
