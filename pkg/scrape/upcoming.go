package scrape

import (
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
