package scrape

import (
	"testing"
	"time"

	"github.com/xbapps/xbvr/pkg/models"
)

// Hermetic tests for the upcoming-scene parsers. The JSON mirrors the shape the sites
// returned in Nov 2026, with made-up content.

func TestParseVRBangersUpcoming(t *testing.T) {
	builder := `{"status":{"message":"Ok"},"data":{"homePageBuilder":{"upcomingScenes":[
		{"text":"Test Scene: Part &amp; One","starring":"Actor One, Actor Two","releaseDate":0,"isActive":true,
		 "image":{"permalink":"/uploads/2026/11/a.jpg"}},
		{"text":"Hidden","starring":"","releaseDate":0,"isActive":false,"image":{"permalink":"/uploads/b.jpg"}},
		{"text":"Dated","starring":"Actor Three","releaseDate":1796000000,"isActive":true,"image":{"permalink":"/uploads/c.jpg"}}
	]}}}`

	got := parseVRBangersUpcoming(builder, "vrbangers", "VRBangers", "https://vrbangers.com/", "https://content.vrbangers.com")
	if len(got) != 2 {
		t.Fatalf("want 2 active scenes, got %d", len(got))
	}
	first := got[0]
	if first.Title != "Test Scene: Part & One" {
		t.Errorf("title not unescaped: %q", first.Title)
	}
	if first.SceneID != models.UpcomingPlaceholderID("vrbangers", first.Title) || first.SceneID != "vrbangers-upcoming-test-scene-part-one" {
		t.Errorf("unexpected placeholder id %q", first.SceneID)
	}
	if !first.Upcoming || first.Released != "" {
		t.Errorf("want upcoming without date, got upcoming=%v released=%q", first.Upcoming, first.Released)
	}
	if len(first.Cast) != 2 || first.Cast[1] != "Actor Two" {
		t.Errorf("cast not split: %v", first.Cast)
	}
	if len(first.Covers) != 1 || first.Covers[0] != "https://content.vrbangers.com/uploads/2026/11/a.jpg" {
		t.Errorf("cover: %v", first.Covers)
	}
	if want := time.Unix(1796000000, 0).Format("2006-01-02"); got[1].Released != want {
		t.Errorf("release date %q, want %q", got[1].Released, want)
	}
}

func TestParseVRSpyUpcoming(t *testing.T) {
	resp := `{"data":{"upcomingVideos":{"content":[
		{"hrc":"test-scene","name":"Test Scene","cover":"https://cdn.vrspy.com/videos/123/images/cover-temp.jpg",
		 "publishedDate":1796000000,"state":"ComingSoon","actors":[{"hrc":"actor-one","name":"Actor One"}]},
		{"hrc":"no-id","name":"No Id","cover":"https://cdn.vrspy.com/other.jpg","publishedDate":0,"actors":[]}
	]}}}`

	got := parseVRSpyUpcoming(resp)
	if len(got) != 1 {
		t.Fatalf("want 1 scene with an id, got %d", len(got))
	}
	sc := got[0]
	if sc.SceneID != "vrspy-123" || sc.SiteID != "123" {
		t.Errorf("ids must match the released scene: %q / %q", sc.SceneID, sc.SiteID)
	}
	if sc.HomepageURL != "https://www.vrspy.com/video/test-scene" {
		t.Errorf("scene URL must match the regular scraper: %q", sc.HomepageURL)
	}
	if !sc.Upcoming || sc.Released != time.Unix(1796000000, 0).Format("2006-01-02") {
		t.Errorf("upcoming=%v released=%q", sc.Upcoming, sc.Released)
	}
	if len(sc.Cast) != 1 || sc.Cast[0] != "Actor One" {
		t.Errorf("cast: %v", sc.Cast)
	}
}
