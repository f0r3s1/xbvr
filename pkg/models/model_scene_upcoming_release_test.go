package models

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func releaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := upcomingTestDB(t)
	if err := db.AutoMigrate(&Tag{}, &Actor{}, &SceneCuepoint{}, &File{}, &History{}, &Action{}, &Site{}, &ExternalReference{}, &ExternalReferenceLink{}, &KV{}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func save(t *testing.T, db *gorm.DB, ext ScrapedScene) {
	t.Helper()
	if err := SceneCreateUpdateFromExternal(db, ext); err != nil {
		t.Fatal(err)
	}
}

// Each case announces a scene, marks it with user data, then scrapes the release.
func TestUpcomingThenReleasedKeepsOneScene(t *testing.T) {
	day := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	cases := []struct {
		name     string
		scraper  string
		announce []ScrapedScene // announcements in order, the first gets the user data
		release  ScrapedScene
		before   bool // the released scene is scraped before the announcement (site keeps it listed)
	}{
		{"vrbangers same title", "vrb",
			[]ScrapedScene{{SceneID: UpcomingPlaceholderID("vrb", "Hot Night: Part 2"), Title: "Hot Night: Part 2"}},
			ScrapedScene{SceneID: "vrb-100", Title: "Hot Night: Part 2"}, false},
		{"vrbangers punctuation and Pt", "vrb",
			[]ScrapedScene{{SceneID: UpcomingPlaceholderID("vrb", "Hot Night Pt. 2"), Title: "Hot Night Pt. 2"}},
			ScrapedScene{SceneID: "vrb-100", Title: "Hot Night - Part 2!"}, false},
		{"vrbangers announcement retitled and listed twice", "vrb",
			[]ScrapedScene{
				{SceneID: UpcomingPlaceholderID("vrb", "Hot Night Pt. 2"), Title: "Hot Night Pt. 2"},
				{SceneID: UpcomingPlaceholderID("vrb", "Hot Night Part 2"), Title: "Hot Night Part 2"},
			},
			ScrapedScene{SceneID: "vrb-100", Title: "Hot Night Part 2"}, false},
		{"vrbangers released scene already stored", "vrb",
			[]ScrapedScene{{SceneID: UpcomingPlaceholderID("vrb", "Hot Night"), Title: "Hot Night"}},
			ScrapedScene{SceneID: "vrb-100", Title: "hot night"}, true},
		{"vrspy same id", "vrspy",
			[]ScrapedScene{{SceneID: "vrspy-123", Title: "Test Scene"}},
			ScrapedScene{SceneID: "vrspy-123", Title: "Test Scene"}, false},
		{"slr same id", "slr-studio",
			[]ScrapedScene{{SceneID: "slr-9", Title: "Test Scene"}},
			ScrapedScene{SceneID: "slr-9", Title: "Test Scene"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := releaseTestDB(t)
			c.release.ScraperID, c.release.Released = c.scraper, day
			if c.before {
				// the stand-in is created first, then the release arrives while it is still listed
				for _, a := range c.announce {
					a.ScraperID, a.Upcoming = c.scraper, true
					save(t, db, a)
				}
				var stand Scene
				db.Where("is_upcoming = ?", true).First(&stand)
				db.Create(&File{SceneID: stand.ID, Filename: "a.mp4"})
				db.Model(&stand).Update("favourite", true)
				// release arrives under a new id without the adoption path (plain insert), then
				// the announcement is listed again
				db.Delete(&Scene{}, stand.ID)
				save(t, db, c.release)
				db.Model(&Scene{}).Unscoped().Where("id = ?", stand.ID).Update("deleted_at", nil)
				a := c.announce[0]
				a.ScraperID, a.Upcoming = c.scraper, true
				save(t, db, a)
			} else {
				for _, a := range c.announce {
					a.ScraperID, a.Upcoming = c.scraper, true
					save(t, db, a)
				}
				var first Scene
				db.Where("is_upcoming = ?", true).Order("id").First(&first)
				db.Create(&File{SceneID: first.ID, Filename: "a.mp4"})
				db.Create(&SceneCuepoint{SceneID: first.ID, Name: "kiss"})
				db.Model(&first).Updates(map[string]interface{}{"favourite": true, "watchlist": true, "star_rating": 4.5})
				save(t, db, c.release)
			}

			var scenes []Scene
			db.Find(&scenes)
			if len(scenes) != 1 {
				t.Fatalf("want exactly one scene, got %d: %+v", len(scenes), scenes)
			}
			s := scenes[0]
			if s.SceneID != c.release.SceneID || s.IsUpcoming {
				t.Errorf("scene_id=%q upcoming=%v", s.SceneID, s.IsUpcoming)
			}
			var files, cues int
			db.Model(&File{}).Where("scene_id = ?", s.ID).Count(&files)
			db.Model(&SceneCuepoint{}).Where("scene_id = ?", s.ID).Count(&cues)
			if !s.Favourite || files != 1 {
				t.Errorf("user data lost: favourite=%v files=%d", s.Favourite, files)
			}
			if !c.before && (!s.Watchlist || s.StarRating != 4.5 || cues != 1) {
				t.Errorf("user data lost: watchlist=%v rating=%v cuepoints=%d", s.Watchlist, s.StarRating, cues)
			}
		})
	}
}

func TestMergeStandInIntoExistingReleasedScene(t *testing.T) {
	db := releaseTestDB(t)
	released := Scene{SceneID: "vrb-1", ScraperId: "vrb", Title: "Same Title", Watchlist: true}
	stand := Scene{SceneID: UpcomingPlaceholderID("vrb", "Same Title"), ScraperId: "vrb", Title: "Same Title", IsUpcoming: true, Favourite: true}
	db.Create(&released)
	db.Create(&stand)
	db.Create(&File{SceneID: stand.ID, Filename: "a.mp4"})
	db.Create(&Action{SceneID: stand.SceneID, ChangedColumn: "title"})

	adoptUpcomingPlaceholder(db, ScrapedScene{SceneID: "vrb-1", ScraperID: "vrb", Title: "Same Title"})

	var scenes []Scene
	db.Find(&scenes)
	if len(scenes) != 1 || !scenes[0].Favourite || !scenes[0].Watchlist {
		t.Fatalf("want one merged scene with both marks, got %+v", scenes)
	}
	var files, actions int
	db.Model(&File{}).Where("scene_id = ?", released.ID).Count(&files)
	db.Model(&Action{}).Where("scene_id = ?", "vrb-1").Count(&actions)
	if files != 1 || actions != 1 {
		t.Errorf("files=%d actions=%d", files, actions)
	}
}

func TestReleasedSceneIsNotTurnedBackIntoAnnouncement(t *testing.T) {
	db := releaseTestDB(t)
	save(t, db, ScrapedScene{SceneID: "vrspy-1", ScraperID: "vrspy", Title: "T"})
	save(t, db, ScrapedScene{SceneID: "vrspy-1", ScraperID: "vrspy", Title: "T", Upcoming: true})
	var s Scene
	db.Where("scene_id = ?", "vrspy-1").First(&s)
	if s.IsUpcoming {
		t.Error("released scene flipped back to upcoming")
	}
}

func TestMatchKeyAbbreviations(t *testing.T) {
	if matchKey("Hot Night Pt. 2") != matchKey("hot night - Part 2") || matchKey("Hot & Delicious") != matchKey("Hot and Delicious") {
		t.Error("abbreviations should match")
	}
	if matchKey("Part 1") == matchKey("Part 2") {
		t.Error("different titles must not match")
	}
}
