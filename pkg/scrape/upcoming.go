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
	models.PruneUpcoming(scraperID, listed)
}
