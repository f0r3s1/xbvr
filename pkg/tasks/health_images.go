package tasks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xbapps/xbvr/pkg/models"
)

// imageKey treats links that differ only by scheme, spaces or a trailing slash as the same picture.
func imageKey(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	return strings.TrimSuffix(u, "/")
}

// dedupeURLs keeps the first copy of each picture (an actor's main image is listed first on purpose and stays).
func dedupeURLs(urls []string) ([]string, int) {
	seen := map[string]bool{}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		k := imageKey(u)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, u)
	}
	return out, len(urls) - len(out)
}

func dedupeSceneImages(images []models.Image) ([]models.Image, int) {
	seen := map[string]bool{}
	out := make([]models.Image, 0, len(images))
	for _, im := range images {
		k := imageKey(im.URL)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, im)
	}
	return out, len(images) - len(out)
}

type duplicateImages struct {
	actors, scenes []AffectedItem
	actorFix       map[uint]string
	sceneFix       map[uint]string
}

// findDuplicateImages lists actors and scenes whose image lists repeat a picture, with the cleaned lists ready to save.
func findDuplicateImages() duplicateImages {
	d := duplicateImages{actorFix: map[uint]string{}, sceneFix: map[uint]string{}}
	commonDb, _ := models.GetCommonDB()
	var actors []struct {
		ID       uint
		Name     string
		ImageArr string
	}
	commonDb.Model(&models.Actor{}).Select("id, name, image_arr").Where("image_arr != '' AND image_arr != '[]'").Scan(&actors)
	for _, a := range actors {
		var urls []string
		if json.Unmarshal([]byte(a.ImageArr), &urls) != nil {
			continue
		}
		clean, n := dedupeURLs(urls)
		if n == 0 {
			continue
		}
		b, _ := json.Marshal(clean)
		d.actorFix[a.ID] = string(b)
		d.actors = append(d.actors, AffectedItem{ID: a.ID, Label: a.Name, Extra: fmt.Sprintf("%d repeated", n)})
	}
	backgroundYield()
	var scenes []struct {
		ID     uint
		Title  string
		Images string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, images").Where("images != '' AND images != '[]'").Scan(&scenes)
	for _, s := range scenes {
		var images []models.Image
		if json.Unmarshal([]byte(s.Images), &images) != nil {
			continue
		}
		clean, n := dedupeSceneImages(images)
		if n == 0 {
			continue
		}
		b, _ := json.Marshal(clean)
		d.sceneFix[s.ID] = string(b)
		d.scenes = append(d.scenes, AffectedItem{ID: s.ID, Label: s.Title, Extra: fmt.Sprintf("%d repeated", n)})
	}
	return d
}

func duplicateImageIssues(d duplicateImages) []HealthIssue {
	var issues []HealthIssue
	add := func(id, what string, items []AffectedItem) {
		if len(items) == 0 {
			return
		}
		total := len(items)
		if len(items) > 50 {
			items = items[:50]
		}
		issues = append(issues, HealthIssue{
			ID: id, Category: what, Severity: "info",
			Description: fmt.Sprintf("%d %s list the same image more than once", total, what),
			Fixable:     true, FixAction: "dedupe-images", FixLabel: "Remove repeated images",
			AffectedItems: items,
		})
	}
	add("duplicate-actor-images", "actors", d.actors)
	add("duplicate-scene-images", "scenes", d.scenes)
	return issues
}

// RemoveDuplicateImages keeps the first copy of every picture in actor and scene image lists.
func RemoveDuplicateImages() {
	d := findDuplicateImages()
	db, _ := models.GetDB()
	defer db.Close()
	for id, arr := range d.actorFix {
		db.Model(&models.Actor{}).Where("id = ?", id).UpdateColumn("image_arr", arr)
	}
	for id, images := range d.sceneFix {
		db.Model(&models.Scene{}).Where("id = ?", id).UpdateColumn("images", images)
	}
	tlog := log.WithField("task", "health")
	tlog.Infof("Removed repeated images from %d actors and %d scenes", len(d.actorFix), len(d.sceneFix))
}
