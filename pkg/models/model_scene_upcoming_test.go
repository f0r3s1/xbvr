package models

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestTitleKeyMatchesSmallDifferences(t *testing.T) {
	pairs := [][2]string{
		{"Sorority Hookup: Sticky Fingers", "sorority hookup - sticky fingers"},
		{"Hot & Delicious", "Hot  &  Delicious!"},
	}
	for _, p := range pairs {
		if titleKey(p[0]) != titleKey(p[1]) {
			t.Errorf("%q and %q should match: %q vs %q", p[0], p[1], titleKey(p[0]), titleKey(p[1]))
		}
	}
	if titleKey("Part 1") == titleKey("Part 2") {
		t.Error("different titles must not match")
	}
}

func upcomingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.AutoMigrate(&Scene{}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpcomingPlaceholderIsAdoptedByReleasedScene(t *testing.T) {
	db := upcomingTestDB(t)
	placeholder := Scene{SceneID: UpcomingPlaceholderID("site", "A Scene: Title"), ScraperId: "site", Title: "A Scene: Title", IsUpcoming: true, Watchlist: true}
	db.Create(&placeholder)

	ext := ScrapedScene{SceneID: "site-abc123", ScraperID: "site", Title: "A Scene - Title"}
	adoptUpcomingPlaceholder(db, ext)

	var got Scene
	db.First(&got, placeholder.ID)
	if got.SceneID != "site-abc123" || !got.Watchlist {
		t.Fatalf("placeholder not adopted: scene_id=%q watchlist=%v", got.SceneID, got.Watchlist)
	}

	// an announcement for the already released title must not create a new placeholder
	db.Model(&got).Update("is_upcoming", false).Update("release_date", time.Now())
	again := ScrapedScene{SceneID: UpcomingPlaceholderID("site", "A Scene: Title"), ScraperID: "site", Title: "A Scene: Title", Upcoming: true}
	if !placeholderAlreadyReleased(db, again) {
		t.Error("released title should be detected")
	}
}

func TestUpcomingFilter(t *testing.T) {
	db := upcomingTestDB(t)
	db.Create(&Scene{SceneID: "released", ReleaseDate: time.Now().AddDate(0, 0, -3)})
	db.Create(&Scene{SceneID: "soon", IsUpcoming: true, ReleaseDate: time.Now().AddDate(0, 0, 3)})
	db.Create(&Scene{SceneID: "later", IsUpcoming: true, ReleaseDate: time.Now().AddDate(0, 0, 40)})
	db.Create(&Scene{SceneID: "undated", IsUpcoming: true})

	count := func(c Config) int {
		var n int
		upcomingFilter(db.Model(&Scene{}), c).Count(&n)
		return n
	}
	show, hide, all, month := true, false, 0, 30
	var c Config
	if n := count(c); n != 3 {
		t.Errorf("defaults (show, 14 days): want 3, got %d", n)
	}
	c.Web.ShowUpcoming = &hide
	if n := count(c); n != 1 {
		t.Errorf("hidden: want 1, got %d", n)
	}
	c.Web.ShowUpcoming, c.Web.UpcomingDays = &show, &all
	if n := count(c); n != 4 {
		t.Errorf("any time: want 4, got %d", n)
	}
	c.Web.UpcomingDays = &month
	if n := count(c); n != 3 {
		t.Errorf("30 days: want 3, got %d", n)
	}
}
