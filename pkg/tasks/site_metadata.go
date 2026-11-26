package tasks

import (
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

const siteMetaMaxAge = 30 * 24 * time.Hour

// RefreshSiteMetadata fetches logo, description and homepage for sites with scenes whose info is missing or older than 30 days.
func RefreshSiteMetadata() { refreshSiteMetadata(false) }

// ForceRefreshSiteMetadata ignores the 30 day age check.
func ForceRefreshSiteMetadata() { refreshSiteMetadata(true) }

func refreshSiteMetadata(force bool) {
	if models.CheckLock("site-metadata") {
		return
	}
	models.CreateLock("site-metadata")
	defer models.RemoveLock("site-metadata")

	db, _ := models.GetDB()
	var sites, all []models.Site
	q := db.Where("id IN (SELECT DISTINCT scraper_id FROM scenes WHERE deleted_at IS NULL)")
	if !force {
		q = q.Where("meta_updated_at IS NULL OR meta_updated_at < ?", time.Now().Add(-siteMetaMaxAge))
	}
	q.Find(&sites)
	db.Where("description <> ''").Find(&all)
	db.Close()

	results := make([]scrape.SiteMeta, len(sites))
	var wg sync.WaitGroup
	slots := make(chan struct{}, 2)
	for i, site := range sites {
		wg.Add(1)
		slots <- struct{}{}
		go func(i int, id string) {
			defer func() { <-slots; wg.Done() }()
			m, err := scrape.FetchSiteMeta(id)
			if err != nil {
				log.WithField("task", "site-metadata").Debugf("%s: %v", id, err)
			}
			if m.Logo != "" && !scrape.ValidLogo(m.Logo) {
				m.Logo = "" // keep the stored logo rather than save a dead or non-image URL
			}
			results[i] = m
		}(i, site.ID)
	}
	wg.Wait()

	// a description shared by two or more sites is templated text, not studio info
	seen := map[string]map[string]bool{}
	note := func(desc, id string) {
		if seen[desc] == nil {
			seen[desc] = map[string]bool{}
		}
		seen[desc][id] = true
	}
	for _, s := range all {
		note(s.Description, s.ID)
	}
	for i, site := range sites {
		note(results[i].Description, site.ID)
	}
	for i, site := range sites {
		m := results[i]
		if m.Description != "" && (len(seen[m.Description]) > 1 || scrape.IsBoilerplateDescription(m.Description, site.Name)) {
			m.Description = ""
		}
		saveSiteMetadata(site.ID, m)
	}
	log.WithField("task", "site-metadata").Infof("Site info refreshed for %d sites", len(sites))
}

func saveSiteMetadata(id string, m scrape.SiteMeta) {
	// only non-empty values are written, so a failed or partial fetch never erases existing info
	fields := map[string]interface{}{"meta_updated_at": time.Now()}
	for col, v := range map[string]string{"logo_url": m.Logo, "description": m.Description, "homepage_url": m.Homepage} {
		if v != "" {
			fields[col] = v
		}
	}
	db, _ := models.GetDB()
	defer db.Close()
	db.Model(&models.Site{}).Where("id = ?", id).Updates(fields)
}
