package scrape

import (
	"github.com/mozillazg/go-slugify"
	"github.com/xbapps/xbvr/pkg/config"
)

func init() {
	RegisterCustomScrapers()
}

// RegisterCustomScrapers registers the custom sites of scrapers.json; called again after a site is added, so no restart is needed.
func RegisterCustomScrapers() {
	var list config.ScraperList
	list.Load()
	c := list.CustomScrapers
	for _, s := range c.PovrScrapers {
		addPOVRScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
	}
	for _, s := range c.SlrScrapers {
		addSLRScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
	}
	for _, s := range c.StashDbScrapers {
		addStashScraper(slugify.Slugify(s.Name), s.Name, s.AvatarUrl, s.URL, s.MasterSiteId)
	}
	for _, s := range c.VrphubScrapers {
		addVRPHubScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, noop)
	}
	for _, s := range c.VrpornScrapers {
		addVRPornScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
	}
	for _, s := range c.RealVRScrapers {
		addRealVRScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
	}
}
