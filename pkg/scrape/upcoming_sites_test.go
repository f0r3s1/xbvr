package scrape

import (
	"os"
	"testing"
	"time"

	"github.com/xbapps/xbvr/pkg/models"
)

// Parser tests for the upcoming blocks of VRHush, FuckPassVR, Czech VR and WankzVR, run
// against trimmed pages saved in Nov 2026 (testdata/upcoming_*.html).

func readFixture(t *testing.T, name string) string {
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseVRHushUpcoming(t *testing.T) {
	got := parseVRHushUpcoming(readFixture(t, "upcoming_vrhush.html"))
	if len(got) != 1 {
		t.Fatalf("want 1 scene, got %d", len(got))
	}
	sc := got[0]
	if sc.SceneID != "vrhush-0812" || sc.SiteID != "0812" {
		t.Errorf("id must match the released scene: %q / %q", sc.SceneID, sc.SiteID)
	}
	if sc.HomepageURL != "https://vrhush.com/scenes/controller-and-kisses-72164" || sc.Title != "Controller And Kisses" {
		t.Errorf("url/title: %q %q", sc.HomepageURL, sc.Title)
	}
	if !sc.Upcoming || sc.Released != "2026-11-29" || sc.Duration != 33 {
		t.Errorf("upcoming=%v released=%q duration=%d", sc.Upcoming, sc.Released, sc.Duration)
	}
	if len(sc.Cast) != 1 || sc.Cast[0] != "Venus Valencia" || len(sc.Tags) == 0 || len(sc.Covers) != 1 {
		t.Errorf("cast/tags/covers: %v %v %v", sc.Cast, sc.Tags, sc.Covers)
	}
}

func TestParseFuckPassVRUpcoming(t *testing.T) {
	now := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	got := parseFuckPassVRUpcoming(readFixture(t, "upcoming_fuckpassvr.html"), now)
	if len(got) < 2 {
		t.Fatalf("want the feature and at least one more scene, got %d", len(got))
	}
	first := got[0]
	if first.SceneID != "fpvr-0452" || first.Title != "Living Room Premiere" || first.Released != "2026-11-27" {
		t.Errorf("feature: %+v", first)
	}
	if len(first.Cast) != 1 || first.Cast[0] != "Lindsay Lee" || !first.Upcoming {
		t.Errorf("cast/upcoming: %v %v", first.Cast, first.Upcoming)
	}
	if got[1].SceneID != "fpvr-0453" || got[1].Released != "2026-11-30" {
		t.Errorf("mini: %+v", got[1])
	}
}

func TestParseCzechVRUpcoming(t *testing.T) {
	page := readFixture(t, "upcoming_czechvr.html")
	got := parseCzechVRUpcoming(page, "czechvrfetish", "Czech VR Fetish")
	if len(got) != 1 {
		t.Fatalf("want only the fetish scene, got %d", len(got))
	}
	sc := got[0]
	if sc.SceneID != "czech-vr-fetish-544" || sc.SiteID != "544" || sc.Title != "Toys and Lingerie" {
		t.Errorf("id/title: %q %q %q", sc.SceneID, sc.SiteID, sc.Title)
	}
	if sc.HomepageURL != "https://www.czechvrnetwork.com/detail-2642-toys-and-lingerie" || sc.Released != "2026-11-25" {
		t.Errorf("url/date: %q %q", sc.HomepageURL, sc.Released)
	}
	if len(sc.Cast) != 1 || sc.Cast[0] != "Nikol Raklova" || len(sc.Covers) != 1 || !sc.Upcoming {
		t.Errorf("cast/covers: %v %v", sc.Cast, sc.Covers)
	}
	if other := parseCzechVRUpcoming(page, "czechvr", "Czech VR"); len(other) != 1 || other[0].SceneID != "czech-vr-919" {
		t.Errorf("czechvr scene: %+v", other)
	}
	if none := parseCzechVRUpcoming(page, "czechar", "Czech AR"); len(none) != 0 {
		t.Errorf("other studio must be empty: %+v", none)
	}
}

func TestParseWankzVRUpcoming(t *testing.T) {
	got := parseWankzVRUpcoming(readFixture(t, "upcoming_wankzvr.html"), "https://www.wankzvr.com/", "wankzvr", "WankzVR (POVR)", "POVR.COM")
	if len(got) != 1 {
		t.Fatalf("want 1 scene, got %d", len(got))
	}
	sc := got[0]
	if sc.SceneID != models.UpcomingPlaceholderID("wankzvr", "In Too Deep") || sc.Title != "In Too Deep" {
		t.Errorf("id/title: %q %q", sc.SceneID, sc.Title)
	}
	if !sc.Upcoming || sc.Released != "2026-11-27" || len(sc.Cast) != 1 || sc.Cast[0] != "Mira Luv" {
		t.Errorf("upcoming=%v released=%q cast=%v", sc.Upcoming, sc.Released, sc.Cast)
	}
	if len(sc.Covers) != 1 || sc.Covers[0] != "https://images.povr.com/wvr/6/6376/6376343/cover/large.jpg" {
		t.Errorf("cover: %v", sc.Covers)
	}
	// milfvr.com shares the markup; its block was empty when checked, so the wankzvr page stands in.
	milf := parseWankzVRUpcoming(readFixture(t, "upcoming_wankzvr.html"), "https://www.milfvr.com/", "milfvr", "MilfVR (POVR)", "POVR.COM")
	if len(milf) != 1 || milf[0].HomepageURL != "https://www.milfvr.com/#upcoming-in-too-deep" || milf[0].ScraperID != "milfvr" {
		t.Errorf("milfvr: %+v", milf)
	}
}

func TestParseUpCloseUpcoming(t *testing.T) {
	got := parseUpCloseUpcoming(readFixture(t, "upcoming_upclosevr.json"), "upclosevr", "UpCloseVR")
	if len(got) != 1 {
		t.Fatalf("want 1 scene, got %d", len(got))
	}
	sc := got[0]
	if sc.SceneID != "upclosevr-288697" || !sc.Upcoming || sc.Released != "2026-12-26" || sc.Duration != 62 {
		t.Errorf("id=%q upcoming=%v released=%q duration=%d", sc.SceneID, sc.Upcoming, sc.Released, sc.Duration)
	}
	if len(sc.Cast) != 1 || sc.Cast[0] != "Aleksa Mink" || len(sc.Covers) != 1 {
		t.Errorf("cast=%v covers=%v", sc.Cast, sc.Covers)
	}
}
