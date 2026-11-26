package scrape

import (
	"time"
	_ "time/tzdata" // the runtime image has no zone database; release times are site-local

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

// fillWeeklyDates gives undated announcements of a weekly site the next release times (UTC) in list order.
func fillWeeklyDates(scenes []models.ScrapedScene, release weeklyRelease, latestRelease, now time.Time) {
	from := now
	if latestRelease.After(from) {
		from = latestRelease
	}
	local := from.In(release.Location)
	next := time.Date(local.Year(), local.Month(), local.Day(), release.Hour, release.Minute, 0, 0, release.Location)
	next = next.AddDate(0, 0, (int(release.Day)-int(next.Weekday())+7)%7)
	if !next.After(from) {
		next = next.AddDate(0, 0, 7)
	}
	for i := range scenes {
		if scenes[i].Released != "" {
			continue
		}
		scenes[i].Released = next.UTC().Format(time.RFC3339)
		next = next.AddDate(0, 0, 7)
	}
}

// weeklyRelease is when a site publishes its weekly scene, in the site's own time zone.
type weeklyRelease struct {
	Day          time.Weekday
	Hour, Minute int
	Location     *time.Location
}
