package models

import (
	"testing"
	"time"

	"github.com/markphelps/optional"
)

func TestQueryScenesFiltersAndCounts(t *testing.T) {
	db := upcomingTestDB(t)
	if err := db.AutoMigrate(&KV{}, &Tag{}, &Actor{}, &SceneCuepoint{}).Error; err != nil {
		t.Fatal(err)
	}
	beach, sunset := Tag{Name: "beach"}, Tag{Name: "sunset"}
	ava, mia := Actor{Name: "Ava"}, Actor{Name: "Mia"}
	for _, v := range []interface{}{&beach, &sunset, &ava, &mia} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	scenes := []Scene{
		{SceneID: "a", Tags: []Tag{beach, sunset}, Cast: []Actor{ava, mia}, IsAvailable: true, IsAccessible: true},
		{SceneID: "b", Tags: []Tag{beach}, Cast: []Actor{ava}, IsAvailable: true},
		{SceneID: "c", Tags: []Tag{sunset}},
		{SceneID: "d", IsHidden: true, Tags: []Tag{beach}},
	}
	for i := range scenes {
		if err := db.Create(&scenes[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Create(&SceneCuepoint{SceneID: scenes[0].ID, Name: "kiss"})
	db.Create(&SceneCuepoint{SceneID: scenes[0].ID, Name: "kiss later"})

	str := func(v ...string) []optional.String {
		out := make([]optional.String, len(v))
		for i, s := range v {
			out[i] = optional.NewString(s)
		}
		return out
	}
	cases := []struct {
		name    string
		r       RequestSceneList
		results int
		any     int
		hidden  int
	}{
		{"all", RequestSceneList{}, 3, 3, 1},
		{"any of two tags (no duplicates)", RequestSceneList{Tags: str("beach", "sunset")}, 3, 3, 1},
		{"must have both tags", RequestSceneList{Tags: str("&beach", "&sunset")}, 1, 1, 0},
		{"exclude tag", RequestSceneList{Tags: str("!beach")}, 1, 1, 0},
		{"any of two actors (no duplicates)", RequestSceneList{Cast: str("Ava", "Mia")}, 2, 2, 0},
		{"two matching cuepoints (no duplicates)", RequestSceneList{Cuepoint: str("kiss")}, 1, 1, 0},
		{"available", RequestSceneList{IsAvailable: optional.NewBool(true)}, 2, 3, 1},
		{"available and accessible", RequestSceneList{IsAvailable: optional.NewBool(true), IsAccessible: optional.NewBool(true)}, 1, 3, 1},
		{"hidden tab", RequestSceneList{DlState: optional.NewString("hidden")}, 1, 3, 1},
	}
	for _, c := range cases {
		pre, final := queryScenes(db, c.r)
		var out ResponseSceneList
		countScenes(pre, c.r, &out)
		var rows []Scene
		final.Find(&rows)
		if out.Results != c.results || len(rows) != c.results || out.CountAny != c.any || out.CountHidden != c.hidden {
			t.Errorf("%s: results=%d rows=%d any=%d hidden=%d, want results=%d any=%d hidden=%d",
				c.name, out.Results, len(rows), out.CountAny, out.CountHidden, c.results, c.any, c.hidden)
		}
	}
}

func TestSceneListPositions(t *testing.T) {
	db := upcomingTestDB(t)
	if err := db.AutoMigrate(&KV{}, &Tag{}, &Actor{}, &SceneCuepoint{}).Error; err != nil {
		t.Fatal(err)
	}
	day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
	scenes := []Scene{
		{SceneID: "old", ReleaseDate: day(1), IsAvailable: true},
		{SceneID: "new", ReleaseDate: day(20), IsAvailable: true},
		{SceneID: "mid", ReleaseDate: day(10), IsAvailable: true},
		{SceneID: "missing", ReleaseDate: day(15)},
	}
	for i := range scenes {
		if err := db.Create(&scenes[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := RequestSceneList{Sort: optional.NewString("release_desc"), IsAvailable: optional.NewBool(true), Limit: optional.NewInt(1)}
	got := sceneListPositions(db, r, []uint{scenes[0].ID, scenes[2].ID, scenes[3].ID})
	if got[scenes[2].ID] != 1 || got[scenes[0].ID] != 2 {
		t.Errorf("positions follow the sort: got %v", got)
	}
	if _, ok := got[scenes[3].ID]; ok {
		t.Error("a scene outside the filter got a position")
	}
}

func TestUndatedUpcomingSortsAsNewest(t *testing.T) {
	db := upcomingTestDB(t)
	if err := db.AutoMigrate(&KV{}, &Tag{}, &Actor{}, &SceneCuepoint{}).Error; err != nil {
		t.Fatal(err)
	}
	scenes := []Scene{
		{SceneID: "released", ReleaseDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{SceneID: "announced", IsUpcoming: true},
	}
	for i := range scenes {
		if err := db.Create(&scenes[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	for sort, first := range map[string]string{"release_desc": "announced", "release_asc": "released"} {
		_, final := queryScenes(db, RequestSceneList{Sort: optional.NewString(sort)})
		var rows []Scene
		final.Find(&rows)
		if len(rows) != 2 || rows[0].SceneID != first {
			t.Errorf("%s: first is %v, want %s", sort, rows, first)
		}
	}
}
