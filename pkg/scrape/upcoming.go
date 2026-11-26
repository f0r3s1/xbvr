package scrape

import (
	"time"

	"github.com/xbapps/xbvr/pkg/models"
)

// emitUpcoming sends a site's announced scenes and drops the stored ones that left its list.
func emitUpcoming(scraperID string, scenes []models.ScrapedScene, out chan<- models.ScrapedScene) {
	listed := make([]string, 0, len(scenes))
	for _, sc := range scenes {
		listed = append(listed, sc.HomepageURL)
		out <- sc
	}
	pruneUpcoming(scraperID, listed)
}

// pruneUpcoming skips an empty list (a failed or changed page) and a cancelled scrape, which would otherwise drop every announcement.
func pruneUpcoming(scraperID string, listed []string) {
	if len(listed) == 0 || ScrapeCancelled() {
		return
	}
	models.PruneUpcoming(scraperID, listed)
}

// fillWeeklyDates gives undated announcements of a weekly site the next release days in list order; the real date replaces it on release.
func fillWeeklyDates(scenes []models.ScrapedScene, day time.Weekday, latestRelease, now time.Time) {
	from := now.Truncate(24 * time.Hour)
	if after := latestRelease.Truncate(24*time.Hour).AddDate(0, 0, 1); after.After(from) {
		from = after
	}
	next := from.AddDate(0, 0, (int(day)-int(from.Weekday())+7)%7)
	for i := range scenes {
		if scenes[i].Released != "" {
			continue
		}
		scenes[i].Released = next.Format("2006-01-02")
		next = next.AddDate(0, 0, 7)
	}
}
