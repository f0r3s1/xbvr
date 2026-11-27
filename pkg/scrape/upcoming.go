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

// fillReleaseDates gives undated announcements the site's next release times (UTC) in list order.
func fillReleaseDates(scenes []models.ScrapedScene, sched releaseSchedule, recent []time.Time, now time.Time) {
	next := sched.after(recent, now)
	for i := range scenes {
		if scenes[i].Released != "" {
			continue
		}
		scenes[i].Released = next.UTC().Format(time.RFC3339)
		next = next.AddDate(0, 0, sched.EveryDays)
	}
}

// releaseSchedule is when a site publishes, in the site's own time zone.
type releaseSchedule struct {
	EveryDays    int
	Weekday      time.Weekday // regular releases fall on this day (unused for daily sites)
	Hour, Minute int
	Location     *time.Location
}

func (s releaseSchedule) at(t time.Time) time.Time {
	t = t.In(s.Location)
	return time.Date(t.Year(), t.Month(), t.Day(), s.Hour, s.Minute, 0, 0, s.Location)
}

// after returns the first scheduled release after now, counted from the latest regular release (extra releases on other days are ignored).
func (s releaseSchedule) after(recent []time.Time, now time.Time) time.Time {
	var anchor time.Time
	for _, t := range recent {
		if (s.EveryDays == 1 || t.In(s.Location).Weekday() == s.Weekday) && t.After(anchor) {
			anchor = t
		}
	}
	var next time.Time
	if anchor.IsZero() {
		next = s.at(now)
		if s.EveryDays > 1 {
			next = next.AddDate(0, 0, (int(s.Weekday)-int(next.Weekday())+7)%7)
		}
	} else {
		next = s.at(anchor).AddDate(0, 0, s.EveryDays)
	}
	for !next.After(now) {
		next = next.AddDate(0, 0, s.EveryDays)
	}
	return next
}
