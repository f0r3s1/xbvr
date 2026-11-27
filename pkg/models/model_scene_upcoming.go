package models

import (
	"database/sql"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// Upcoming scenes are scenes a site announces before release.

const (
	DefaultUpcomingDays = 14
	upcomingIDInfix     = "-upcoming-"
	// how long a scene must have been gone from a site's upcoming list before it is removed
	upcomingGrace = 14 * 24 * time.Hour
)

// UpcomingPlaceholderID is the scene id for an announced scene whose real id the site does not publish yet.
func UpcomingPlaceholderID(scraperID string, title string) string {
	return scraperID + upcomingIDInfix + titleKey(title)
}

// titleKey reduces a title to lowercase letters and digits joined by dashes.
func titleKey(title string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}), "-")
}

// matchKey is titleKey with the usual abbreviations spelled out, used to match a stand-in against the released scene.
func matchKey(title string) string {
	var out []string
	for _, w := range strings.Split(titleKey(title), "-") {
		switch w {
		case "pt":
			w = "part"
		case "vol":
			w = "volume"
		case "ep":
			w = "episode"
		case "and", "":
			continue
		}
		out = append(out, w)
	}
	return strings.Join(out, "-")
}

func isPlaceholderID(scraperID string, sceneID string) bool {
	return strings.HasPrefix(sceneID, scraperID+upcomingIDInfix)
}

// findPlaceholders returns the stand-in scenes of a scraper whose title matches.
func findPlaceholders(db *gorm.DB, scraperID string, title string) []Scene {
	var candidates, out []Scene
	db.Where("scraper_id = ? AND is_upcoming = ? AND scene_id LIKE ?", scraperID, true, scraperID+upcomingIDInfix+"%").
		Order("id").Find(&candidates)
	key := matchKey(title)
	for _, c := range candidates {
		if matchKey(c.Title) == key {
			out = append(out, c)
		}
	}
	return out
}

// adoptUpcomingPlaceholder hands a stand-in's record to the released scene so user data stays on one scene.
func adoptUpcomingPlaceholder(db *gorm.DB, ext ScrapedScene) {
	if ext.Upcoming || ext.ScraperID == "" || ext.Title == "" {
		return
	}
	stands := findPlaceholders(db, ext.ScraperID, ext.Title)
	if len(stands) == 0 {
		return
	}
	var target Scene
	if db.Where("scene_id = ?", ext.SceneID).First(&target).RecordNotFound() {
		target = stands[0]
		db.Model(&Scene{}).Where("id = ?", target.ID).Update("scene_id", ext.SceneID)
		stands = stands[1:]
	}
	for _, s := range stands {
		mergeStandIn(db, s, target.ID)
	}
}

// mergeStandIn moves everything the user attached to a stand-in onto the released scene and removes the stand-in.
func mergeStandIn(db *gorm.DB, from Scene, toID uint) {
	var to Scene
	if db.First(&to, toID).RecordNotFound() || from.ID == to.ID {
		return
	}
	err := inTransaction(db, func(tx *gorm.DB) error {
		for _, m := range []interface{}{&File{}, &History{}, &SceneCuepoint{}} {
			if err := tx.Model(m).Where("scene_id = ?", from.ID).Update("scene_id", to.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&Action{}).Where("scene_id = ?", from.SceneID).Update("scene_id", to.SceneID).Error; err != nil {
			return err
		}
		if err := tx.Model(&ExternalReferenceLink{}).Where("internal_table = ? AND internal_db_id = ?", "scenes", from.ID).Update("internal_db_id", to.ID).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"favourite":        to.Favourite || from.Favourite,
			"watchlist":        to.Watchlist || from.Watchlist,
			"wishlist":         to.Wishlist || from.Wishlist,
			"trailerlist":      to.Trailerlist || from.Trailerlist,
			"is_hidden":        to.IsHidden || from.IsHidden,
			"is_watched":       to.IsWatched || from.IsWatched,
			"is_available":     to.IsAvailable || from.IsAvailable,
			"total_watch_time": to.TotalWatchTime + from.TotalWatchTime,
		}
		if to.StarRating == 0 {
			updates["star_rating"] = from.StarRating
		}
		if from.LastOpened.After(to.LastOpened) {
			updates["last_opened"] = from.LastOpened
		}
		if err := tx.Model(&Scene{}).Where("id = ?", to.ID).Updates(updates).Error; err != nil {
			return err
		}
		return deleteScenes(tx, []uint{from.ID})
	})
	if err != nil {
		log.Errorf("Could not merge upcoming scene %v into %v: %v", from.ID, to.ID, err)
	}
}

// inTransaction runs fn in a new transaction, or in db itself when db already is one.
func inTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if _, isTx := db.CommonDB().(*sql.Tx); isTx {
		return fn(db)
	}
	return db.Transaction(fn)
}

// deleteScenes soft-deletes scenes and drops their cast, tag and external link rows so counts stay right.
func deleteScenes(db *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	steps := []func() *gorm.DB{
		func() *gorm.DB { return db.Exec("DELETE FROM scene_cast WHERE scene_id IN (?)", ids) },
		func() *gorm.DB { return db.Exec("DELETE FROM scene_tags WHERE scene_id IN (?)", ids) },
		func() *gorm.DB {
			return db.Where("internal_table = ? AND internal_db_id IN (?)", "scenes", ids).Delete(&ExternalReferenceLink{})
		},
		func() *gorm.DB { return db.Where("id IN (?)", ids).Delete(&Scene{}) },
	}
	for _, step := range steps {
		if err := step().Error; err != nil {
			return err
		}
	}
	NotifySceneChanged(ids...)
	return nil
}

// canonicalPlaceholderID keeps an announcement whose title changed slightly on its stored stand-in instead of creating a second one.
func canonicalPlaceholderID(db *gorm.DB, ext ScrapedScene) ScrapedScene {
	if !ext.Upcoming || !isPlaceholderID(ext.ScraperID, ext.SceneID) {
		return ext
	}
	stands := findPlaceholders(db, ext.ScraperID, ext.Title)
	for _, s := range stands {
		if s.SceneID == ext.SceneID {
			return ext
		}
	}
	if len(stands) > 0 {
		ext.SceneID = stands[0].SceneID
	}
	return ext
}

// placeholderAlreadyReleased reports a stand-in whose scene is already stored under its real id, merging any stored stand-in.
func placeholderAlreadyReleased(db *gorm.DB, ext ScrapedScene) bool {
	if !ext.Upcoming || !isPlaceholderID(ext.ScraperID, ext.SceneID) {
		return false
	}
	var released []Scene
	db.Where("scraper_id = ? AND is_upcoming = ?", ext.ScraperID, false).
		Where("release_date > ?", time.Now().AddDate(0, -3, 0)).Order("id").Find(&released)
	key := matchKey(ext.Title)
	for _, r := range released {
		if matchKey(r.Title) == key {
			for _, s := range findPlaceholders(db, ext.ScraperID, ext.Title) {
				mergeStandIn(db, s, r.ID)
			}
			return true
		}
	}
	return false
}

// upcomingFilter applies the listing settings:
func upcomingFilter(tx *gorm.DB, c Config) *gorm.DB {
	if c.Web.ShowUpcoming != nil && !*c.Web.ShowUpcoming {
		return tx.Where("scenes.is_upcoming = ?", false)
	}
	days := DefaultUpcomingDays
	if c.Web.UpcomingDays != nil {
		days = *c.Web.UpcomingDays
	}
	if days <= 0 {
		return tx
	}
	return tx.Where("scenes.is_upcoming = ? OR scenes.release_date <= ?", false, time.Now().AddDate(0, 0, days))
}

// UpcomingReleaseDates maps each stored upcoming scene URL of a scraper to its release date.
func UpcomingReleaseDates(scraperID string) map[string]time.Time {
	db, _ := GetCommonDB()
	var scenes []Scene
	db.Select("scene_url, release_date").Where("scraper_id = ? AND is_upcoming = ?", scraperID, true).Find(&scenes)
	out := make(map[string]time.Time, len(scenes))
	for _, s := range scenes {
		out[s.SceneURL] = s.ReleaseDate
	}
	return out
}

// PruneUpcoming removes announcements that left the site's upcoming list without being released (renamed or cancelled).
func PruneUpcoming(scraperID string, listedSceneURLs []string) {
	if len(listedSceneURLs) == 0 {
		return
	}
	db, _ := GetCommonDB()
	var ids []uint
	db.Model(&Scene{}).
		Where("scraper_id = ? AND is_upcoming = ? AND updated_at < ?", scraperID, true, time.Now().Add(-upcomingGrace)).
		Where("is_available = ? AND favourite = ? AND watchlist = ? AND wishlist = ?", false, false, false, false).
		Where("scene_url NOT IN (?)", listedSceneURLs).
		Pluck("id", &ids)
	if err := deleteScenes(db, ids); err != nil {
		log.Errorf("Could not prune upcoming scenes of %v: %v", scraperID, err)
	}
}

// RecentReleaseDates are the newest release dates of a scraper's released scenes.
func RecentReleaseDates(scraperID string, n int) []time.Time {
	db, _ := GetCommonDB()
	var scenes []Scene
	db.Select("release_date").Where("scraper_id = ? AND is_upcoming = ?", scraperID, false).Order("release_date desc").Limit(n).Find(&scenes)
	out := make([]time.Time, len(scenes))
	for i, s := range scenes {
		out[i] = s.ReleaseDate
	}
	return out
}
