package api

import (
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

type SceneSite struct {
	Name          string `json:"name"`
	ScraperID     string `json:"scraper_id"`
	AvatarURL     string `json:"avatar_url"`
	LogoURL       string `json:"logo_url"`
	HomepageURL   string `json:"homepage_url"`
	SceneCount    int    `json:"scene_count"`
	Available     int    `json:"available"`
	Upcoming      int    `json:"upcoming"`
	LatestRelease string `json:"latest_release"`
}

func (i SceneResource) getSites(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var sites []SceneSite
	db.Raw(`SELECT s.site AS name, s.scraper_id,
			COALESCE(st.avatar_url, '') AS avatar_url,
			COALESCE(NULLIF(MAX(st.logo_url), ''), st.avatar_url, '') AS logo_url,
			COALESCE(MAX(st.homepage_url), '') AS homepage_url,
			COUNT(*) AS scene_count,
			SUM(CASE WHEN s.is_available THEN 1 ELSE 0 END) AS available,
			SUM(CASE WHEN s.is_upcoming AND s.release_date <= ? THEN 1 ELSE 0 END) AS upcoming,
			MAX(CASE WHEN s.is_upcoming OR s.release_date < ? THEN NULL ELSE s.release_date END) AS latest_release
		FROM scenes s
		LEFT JOIN sites st ON st.id = s.scraper_id
		WHERE s.deleted_at IS NULL AND s.site <> ''
		GROUP BY s.site, s.scraper_id, st.avatar_url
		ORDER BY s.site`, upcomingCutoff(), time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)).Scan(&sites)
	for i := range sites {
		if len(sites[i].LatestRelease) > 10 {
			sites[i].LatestRelease = sites[i].LatestRelease[:10]
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, sites)
}

// upcomingCutoff mirrors the scene list: upcoming scenes count only inside the Web UI window.
func upcomingCutoff() time.Time {
	switch {
	case !config.Config.Web.ShowUpcoming:
		return time.Time{}
	case config.Config.Web.UpcomingDays <= 0:
		return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Now().AddDate(0, 0, config.Config.Web.UpcomingDays)
	}
}
