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

func TestFillReleaseDates(t *testing.T) {
	at := func(m time.Month, d, h, min int) time.Time { return time.Date(2026, m, d, h, min, 0, 0, time.UTC) }
	// VRBangers: Fridays, plus a Monday compilation that must not shift the schedule
	vrb := []time.Time{at(11, 20, 8, 1), at(11, 16, 8, 1), at(11, 13, 8, 1)}
	cases := []struct {
		name   string
		site   string
		recent []time.Time
		now    time.Time
		want   []string
	}{
		{"weekly, thursday evening", "vrbangers", vrb, at(11, 26, 22, 0), []string{"2026-11-27T08:01:00Z", "2026-12-04T08:01:00Z"}},
		{"weekly, friday before 09:01 Paris", "vrbangers", vrb, at(11, 27, 7, 30), []string{"2026-11-27T08:01:00Z", "2026-12-04T08:01:00Z"}},
		{"weekly, friday after release", "vrbangers", vrb, at(11, 27, 9, 0), []string{"2026-12-04T08:01:00Z", "2026-12-11T08:01:00Z"}},
		{"weekly, latest is a monday extra", "vrbangers", []time.Time{at(11, 23, 8, 1), at(11, 20, 8, 1)}, at(11, 24, 12, 0), []string{"2026-11-27T08:01:00Z", "2026-12-04T08:01:00Z"}},
		{"every other friday", "vrbtrans", []time.Time{at(11, 20, 8, 1)}, at(11, 26, 22, 0), []string{"2026-12-04T08:01:00Z", "2026-12-18T08:01:00Z"}},
		{"daily", "arporn", []time.Time{at(11, 26, 8, 1)}, at(11, 26, 22, 0), []string{"2026-11-27T08:01:00Z", "2026-11-28T08:01:00Z"}},
		{"no history yet", "vrbangers", nil, at(11, 26, 22, 0), []string{"2026-11-27T08:01:00Z", "2026-12-04T08:01:00Z"}},
	}
	for _, c := range cases {
		scenes := []models.ScrapedScene{{Title: "a"}, {Title: "b"}}
		fillReleaseDates(scenes, vrbangersSchedules[c.site], c.recent, c.now)
		if scenes[0].Released != c.want[0] || scenes[1].Released != c.want[1] {
			t.Errorf("%s: got %s %s", c.name, scenes[0].Released, scenes[1].Released)
		}
	}
	// summer time: 09:01 Paris is 07:01 UTC; a dated announcement keeps its date
	summer := []models.ScrapedScene{{Title: "a"}, {Title: "dated", Released: "2026-07-20"}}
	fillReleaseDates(summer, vrbangersSchedules["vrbangers"], []time.Time{at(6, 26, 7, 1)}, at(6, 30, 12, 0))
	if summer[0].Released != "2026-07-03T07:01:00Z" || summer[1].Released != "2026-07-20" {
		t.Errorf("summer: got %s %s", summer[0].Released, summer[1].Released)
	}
}

func TestVRBangersUpcomingOrderAndDates(t *testing.T) {
	builder := `{"data":{"homePageBuilder":{"upcomingScenes":[
		{"text":"Oohmakase with Lucy Mochi","sort":36,"releaseDate":0,"isActive":true},
		{"text":"Sorority Hookup: Sticky Fingers","sort":35,"releaseDate":0,"isActive":true},
		{"text":"Bikini Car Wash","sort":37,"releaseDate":0,"isActive":true}]}}}`
	scenes := parseVRBangersUpcoming(builder, "vrbangers", "VRBangers", "https://vrbangers.com/", "https://content.vrbangers.com")
	fillReleaseDates(scenes, vrbangersSchedules["vrbangers"], []time.Time{time.Date(2026, 11, 20, 8, 1, 0, 0, time.UTC)}, time.Date(2026, 11, 26, 22, 0, 0, 0, time.UTC))
	want := []string{"Sorority Hookup: Sticky Fingers 2026-11-27T08:01:00Z", "Oohmakase with Lucy Mochi 2026-12-04T08:01:00Z", "Bikini Car Wash 2026-12-11T08:01:00Z"}
	for i, w := range want {
		if got := scenes[i].Title + " " + scenes[i].Released; got != w {
			t.Errorf("%d: got %q want %q", i, got, w)
		}
	}
}
