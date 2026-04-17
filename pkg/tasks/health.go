package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type AffectedItem struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
	Extra string `json:"extra,omitempty"`
}

type HealthIssue struct {
	ID            string         `json:"id"`
	Category      string         `json:"category"`
	Severity      string         `json:"severity"`
	Description   string         `json:"description"`
	Detail        string         `json:"detail,omitempty"`
	Fixable       bool           `json:"fixable"`
	FixAction     string         `json:"fix_action,omitempty"`
	FixLabel      string         `json:"fix_label,omitempty"`
	AffectedItems []AffectedItem `json:"affected_items"`
}

type HealthReport struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Duration    string                 `json:"duration"`
	Summary     map[string]int         `json:"summary"`
	Issues      []HealthIssue          `json:"issues"`
	Stats       map[string]interface{} `json:"stats"`
}

type HealthProgress struct {
	Running    bool    `json:"running"`
	Step       string  `json:"step"`
	StepNum    int     `json:"step_num"`
	TotalSteps int     `json:"total_steps"`
	Percent    float64 `json:"percent"`
}

var (
	healthRunning  int32
	healthCancel   int32
	lastReport     *HealthReport
	lastReportLock sync.RWMutex

	// imageErrorsMu guards imageErrors map (sceneID → timestamp of last failure)
	imageErrorsMu sync.Mutex
	imageErrors   = map[uint]time.Time{}
)

const totalHealthSteps = 16

func publishProgress(step string, stepNum int) {
	pct := float64(stepNum) / float64(totalHealthSteps) * 100
	common.PublishWS("health.progress", map[string]interface{}{
		"running":     true,
		"step":        step,
		"step_num":    stepNum,
		"total_steps": totalHealthSteps,
		"percent":     pct,
	})
}

func publishDone() {
	common.PublishWS("health.progress", map[string]interface{}{
		"running":     false,
		"step":        "done",
		"step_num":    totalHealthSteps,
		"total_steps": totalHealthSteps,
		"percent":     100,
	})
}

func publishCancelled() {
	common.PublishWS("health.progress", map[string]interface{}{
		"running":     false,
		"step":        "cancelled",
		"step_num":    0,
		"total_steps": totalHealthSteps,
		"percent":     0,
	})
}

func cancelled() bool {
	return atomic.LoadInt32(&healthCancel) == 1
}

func IsHealthRunning() bool {
	return atomic.LoadInt32(&healthRunning) == 1
}

func CancelHealthCheck() {
	atomic.StoreInt32(&healthCancel, 1)
}

func GetLastHealthReport() *HealthReport {
	lastReportLock.RLock()
	defer lastReportLock.RUnlock()
	return lastReport
}

// ReportImageError is called by the API when the browser reports a cover image 404.
func ReportImageError(sceneID uint) {
	imageErrorsMu.Lock()
	imageErrors[sceneID] = time.Now()
	imageErrorsMu.Unlock()
}

// getLastReportItemIDs returns the scene IDs from a specific issue in the last report.
func getLastReportItemIDs(issueID string) []uint {
	lastReportLock.RLock()
	defer lastReportLock.RUnlock()
	if lastReport == nil {
		return nil
	}
	for _, issue := range lastReport.Issues {
		if issue.ID == issueID {
			ids := make([]uint, 0, len(issue.AffectedItems))
			for _, item := range issue.AffectedItems {
				if item.ID > 0 {
					ids = append(ids, item.ID)
				}
			}
			return ids
		}
	}
	return nil
}

func RunHealthCheck() {
	if !atomic.CompareAndSwapInt32(&healthRunning, 0, 1) {
		return // already running
	}
	defer atomic.StoreInt32(&healthRunning, 0)
	atomic.StoreInt32(&healthCancel, 0)

	start := time.Now()
	issues := make([]HealthIssue, 0)
	stats := make(map[string]interface{})

	commonDb, _ := models.GetCommonDB()

	// -- Step 1: Stats --
	publishProgress("Counting records", 1)
	var totalScenes, totalFiles, totalActors, totalTags int
	commonDb.Model(&models.Scene{}).Count(&totalScenes)
	commonDb.Model(&models.File{}).Count(&totalFiles)
	commonDb.Model(&models.Actor{}).Count(&totalActors)
	commonDb.Model(&models.Tag{}).Count(&totalTags)
	stats["total_scenes"] = totalScenes
	stats["total_files"] = totalFiles
	stats["total_actors"] = totalActors
	stats["total_tags"] = totalTags

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 2: Orphaned files --
	publishProgress("Checking orphaned files", 2)
	var orphanedFiles []models.File
	commonDb.Where("scene_id = 0").Limit(50).Find(&orphanedFiles)
	var orphanedCount int
	commonDb.Model(&models.File{}).Where("scene_id = 0").Count(&orphanedCount)
	if orphanedCount > 0 {
		items := make([]AffectedItem, 0, len(orphanedFiles))
		for _, f := range orphanedFiles {
			items = append(items, AffectedItem{ID: f.ID, Label: f.Filename, Extra: f.Path})
		}
		issues = append(issues, HealthIssue{
			ID: "orphaned-files", Category: "files", Severity: "warning",
			Description: fmt.Sprintf("%d files not matched to any scene", orphanedCount),
			Detail:      "Rescan volumes or manually match these files", Fixable: true,
			FixAction: "rescan", FixLabel: "Rescan Volumes", AffectedItems: items,
		})
	}
	stats["orphaned_files"] = orphanedCount

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 3: Stale available --
	publishProgress("Checking stale availability flags", 3)
	var staleAvailableScenes []struct {
		ID    uint
		Title string
		Site  string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, site").
		Where("is_available = ? AND id NOT IN (SELECT DISTINCT scene_id FROM files WHERE type = ? AND scene_id > 0)", true, "video").
		Limit(50).Scan(&staleAvailableScenes)
	var staleAvailableCount int
	commonDb.Model(&models.Scene{}).
		Where("is_available = ? AND id NOT IN (SELECT DISTINCT scene_id FROM files WHERE type = ? AND scene_id > 0)", true, "video").
		Count(&staleAvailableCount)
	if staleAvailableCount > 0 {
		items := make([]AffectedItem, 0, len(staleAvailableScenes))
		for _, s := range staleAvailableScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.Site})
		}
		issues = append(issues, HealthIssue{
			ID: "stale-available", Category: "scenes", Severity: "critical",
			Description: fmt.Sprintf("%d scenes marked available but have no video files", staleAvailableCount),
			Detail:      "Will reset is_available flag to false", Fixable: true,
			FixAction: "refresh-status", FixLabel: "Refresh Status", AffectedItems: items,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 4: Stale scripted --
	publishProgress("Checking stale script flags", 4)
	var staleScriptedScenes []struct {
		ID    uint
		Title string
		Site  string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, site").
		Where("is_scripted = ? AND id NOT IN (SELECT DISTINCT scene_id FROM files WHERE type = ? AND scene_id > 0)", true, "script").
		Limit(50).Scan(&staleScriptedScenes)
	var staleScriptedCount int
	commonDb.Model(&models.Scene{}).
		Where("is_scripted = ? AND id NOT IN (SELECT DISTINCT scene_id FROM files WHERE type = ? AND scene_id > 0)", true, "script").
		Count(&staleScriptedCount)
	if staleScriptedCount > 0 {
		items := make([]AffectedItem, 0, len(staleScriptedScenes))
		for _, s := range staleScriptedScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.Site})
		}
		issues = append(issues, HealthIssue{
			ID: "stale-scripted", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d scenes marked scripted but have no script files", staleScriptedCount),
			Detail:      "Will reset is_scripted flag to false", Fixable: true,
			FixAction: "refresh-status", FixLabel: "Refresh Status", AffectedItems: items,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 5: Missing covers --
	publishProgress("Checking missing covers", 5)
	var noCoverScenes []struct {
		ID    uint
		Title string
		Site  string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, site").
		Where("(cover_url = '' OR cover_url IS NULL) AND is_available = ?", true).
		Limit(50).Scan(&noCoverScenes)
	if len(noCoverScenes) > 0 {
		items := make([]AffectedItem, 0, len(noCoverScenes))
		for _, s := range noCoverScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.Site})
		}
		var noCoverCount int
		commonDb.Model(&models.Scene{}).
			Where("(cover_url = '' OR cover_url IS NULL) AND is_available = ?", true).Count(&noCoverCount)
		issues = append(issues, HealthIssue{
			ID: "missing-covers", Category: "scenes", Severity: "info",
			Description: fmt.Sprintf("%d available scenes have no cover image", noCoverCount),
			Detail:      "Will trigger a metadata re-scrape for these scenes", Fixable: true,
			FixAction: "rescrape-scenes", FixLabel: "Re-scrape Scenes",
			AffectedItems: items,
		})
		stats["missing_covers"] = noCoverCount
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 6: Missing cast --
	publishProgress("Checking missing cast", 6)
	var noCastScenes []struct {
		ID    uint
		Title string
		Site  string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, site").
		Where("id NOT IN (SELECT scene_id FROM scene_cast) AND is_hidden = ?", false).
		Limit(50).Scan(&noCastScenes)
	var noCastCount int
	commonDb.Model(&models.Scene{}).
		Where("id NOT IN (SELECT scene_id FROM scene_cast) AND is_hidden = ?", false).Count(&noCastCount)
	if noCastCount > 0 {
		items := make([]AffectedItem, 0, len(noCastScenes))
		for _, s := range noCastScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.Site})
		}
		issues = append(issues, HealthIssue{
			ID: "missing-cast", Category: "scenes", Severity: "info",
			Description:   fmt.Sprintf("%d scenes have no cast information", noCastCount),
			AffectedItems: items,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 7: Missing tags --
	publishProgress("Checking missing tags", 7)
	var noTagScenes []struct {
		ID    uint
		Title string
		Site  string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, site").
		Where("id NOT IN (SELECT scene_id FROM scene_tags) AND is_hidden = ?", false).
		Limit(50).Scan(&noTagScenes)
	var noTagCount int
	commonDb.Model(&models.Scene{}).
		Where("id NOT IN (SELECT scene_id FROM scene_tags) AND is_hidden = ?", false).Count(&noTagCount)
	if noTagCount > 0 {
		items := make([]AffectedItem, 0, len(noTagScenes))
		for _, s := range noTagScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.Site})
		}
		issues = append(issues, HealthIssue{
			ID: "missing-tags", Category: "scenes", Severity: "info",
			Description:   fmt.Sprintf("%d scenes have no tags", noTagCount),
			AffectedItems: items,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 8: Files missing on disk --
	publishProgress("Checking files on disk", 8)
	var filesToCheck []models.File
	commonDb.Preload("Volume").Where("scene_id > 0").Find(&filesToCheck)
	var missingOnDisk []AffectedItem
	for _, f := range filesToCheck {
		if cancelled() {
			publishCancelled()
			return
		}
		if f.Volume.Type == "local" {
			fullPath := filepath.Join(f.Path, f.Filename)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				missingOnDisk = append(missingOnDisk, AffectedItem{ID: f.ID, Label: f.Filename, Extra: f.Path})
			}
		}
	}
	if len(missingOnDisk) > 0 {
		issues = append(issues, HealthIssue{
			ID: "files-missing-on-disk", Category: "files", Severity: "warning",
			Description: fmt.Sprintf("%d matched files are missing from disk", len(missingOnDisk)),
			Detail:      "Rescan volumes to update file statuses", Fixable: true,
			FixAction: "rescan", FixLabel: "Rescan Volumes", AffectedItems: missingOnDisk,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 9: Duplicate scene IDs --
	publishProgress("Checking duplicate scene IDs", 9)
	type dupResult struct {
		SceneID string
		Cnt     int
	}
	var dups []dupResult
	commonDb.Model(&models.Scene{}).
		Select("scene_id, count(*) as cnt").Where("scene_id != ''").
		Group("scene_id").Having("count(*) > 1").Limit(50).Scan(&dups)
	if len(dups) > 0 {
		items := make([]AffectedItem, 0, len(dups))
		total := 0
		for _, d := range dups {
			total += d.Cnt
			items = append(items, AffectedItem{Label: d.SceneID, Extra: fmt.Sprintf("%d copies", d.Cnt)})
		}
		issues = append(issues, HealthIssue{
			ID: "duplicate-scene-ids", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d duplicate scene IDs (%d total records)", len(dups), total),
			Detail:      "Will keep the record with the most associated files and delete duplicates",
			Fixable:     true, FixAction: "remove-duplicates", FixLabel: "Remove Duplicates",
			AffectedItems: items,
		})
	}
	stats["duplicate_scene_ids"] = len(dups)

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 10: Orphaned tags --
	publishProgress("Checking orphaned tags", 10)
	var orphanedTagsList []models.Tag
	commonDb.Where("id NOT IN (SELECT tag_id FROM scene_tags)").Limit(50).Find(&orphanedTagsList)
	var orphanedTagCount int
	commonDb.Model(&models.Tag{}).Where("id NOT IN (SELECT tag_id FROM scene_tags)").Count(&orphanedTagCount)
	if orphanedTagCount > 0 {
		items := make([]AffectedItem, 0, len(orphanedTagsList))
		for _, t := range orphanedTagsList {
			items = append(items, AffectedItem{ID: t.ID, Label: t.Name})
		}
		issues = append(issues, HealthIssue{
			ID: "orphaned-tags", Category: "tags", Severity: "info",
			Description: fmt.Sprintf("%d tags not used by any scene", orphanedTagCount),
			Detail:      "Will delete unused tags", Fixable: true,
			FixAction: "clean-tags", FixLabel: "Clean Tags", AffectedItems: items,
		})
	}
	stats["orphaned_tags"] = orphanedTagCount

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 11: Blank titles --
	publishProgress("Checking blank titles", 11)
	var blankTitleScenes []struct {
		ID      uint
		SceneID string
		Site    string
	}
	commonDb.Model(&models.Scene{}).Select("id, scene_id, site").
		Where("title = '' OR title IS NULL").Limit(50).Scan(&blankTitleScenes)
	if len(blankTitleScenes) > 0 {
		items := make([]AffectedItem, 0, len(blankTitleScenes))
		for _, s := range blankTitleScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.SceneID, Extra: s.Site})
		}
		issues = append(issues, HealthIssue{
			ID: "blank-titles", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d scenes have no title", len(blankTitleScenes)),
			Detail:      "Will delete scenes with no title and no associated files",
			Fixable:     true, FixAction: "delete-blank-scenes", FixLabel: "Delete Untitled (no files)",
			AffectedItems: items,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 12: Missing previews --
	publishProgress("Checking missing previews", 12)
	var availableScenes []models.Scene
	commonDb.Select("id, scene_id").Where("is_available = ? AND has_preview = ?", true, false).Find(&availableScenes)
	var missingPreviewItems []AffectedItem
	for _, s := range availableScenes {
		previewPath := filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", s.SceneID))
		if _, err := os.Stat(previewPath); os.IsNotExist(err) {
			missingPreviewItems = append(missingPreviewItems, AffectedItem{ID: s.ID, Label: s.SceneID})
		}
	}
	if len(missingPreviewItems) > 0 {
		issues = append(issues, HealthIssue{
			ID: "missing-previews", Category: "scenes", Severity: "info",
			Description: fmt.Sprintf("%d available scenes have no video preview", len(missingPreviewItems)),
			Fixable:     true, FixAction: "generate-previews", FixLabel: "Generate Previews",
			AffectedItems: missingPreviewItems,
		})
	}
	stats["missing_previews"] = len(missingPreviewItems)

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 13: Broken images JSON --
	publishProgress("Checking image data integrity", 13)
	var allSceneImages []struct {
		ID     uint
		Title  string
		Images string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, images").
		Where("images != '' AND images IS NOT NULL").Scan(&allSceneImages)
	var brokenImageItems []AffectedItem
	for _, s := range allSceneImages {
		var imgs []models.Image
		if err := json.Unmarshal([]byte(s.Images), &imgs); err != nil {
			brokenImageItems = append(brokenImageItems, AffectedItem{ID: s.ID, Label: s.Title})
		}
	}
	if len(brokenImageItems) > 0 {
		issues = append(issues, HealthIssue{
			ID: "broken-images-json", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d scenes have malformed image data", len(brokenImageItems)),
			Detail:      "Will clear the broken image data; re-scrape to restore",
			Fixable:     true, FixAction: "clear-broken-images", FixLabel: "Clear Broken Image Data",
			AffectedItems: brokenImageItems,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 14: Future dates & zero duration --
	publishProgress("Checking metadata quality", 14)
	var zeroDurScenes []struct {
		ID    uint
		Title string
		Site  string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, site").
		Where("is_available = ? AND duration = 0", true).Limit(50).Scan(&zeroDurScenes)
	if len(zeroDurScenes) > 0 {
		items := make([]AffectedItem, 0, len(zeroDurScenes))
		for _, s := range zeroDurScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.Site})
		}
		issues = append(issues, HealthIssue{
			ID: "zero-duration", Category: "scenes", Severity: "info",
			Description:   fmt.Sprintf("%d available scenes have no duration", len(zeroDurScenes)),
			AffectedItems: items,
		})
	}

	var futureDateScenes []struct {
		ID          uint
		Title       string
		ReleaseDate time.Time
	}
	commonDb.Model(&models.Scene{}).Select("id, title, release_date").
		Where("release_date > ? AND release_date != '0001-01-01 00:00:00+00:00'", time.Now().AddDate(0, 1, 0)).
		Limit(50).Scan(&futureDateScenes)
	if len(futureDateScenes) > 0 {
		items := make([]AffectedItem, 0, len(futureDateScenes))
		for _, s := range futureDateScenes {
			items = append(items, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.ReleaseDate.Format("2006-01-02")})
		}
		issues = append(issues, HealthIssue{
			ID: "future-dates", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d scenes have release dates far in the future", len(futureDateScenes)),
			Detail:      "Possibly bad scrape data", AffectedItems: items,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 15: Dead cover URLs (all scenes, capped at 300) --
	publishProgress("Checking cover URLs", 15)
	var coverScenes []struct {
		ID       uint
		Title    string
		CoverURL string
	}
	commonDb.Model(&models.Scene{}).Select("id, title, cover_url").
		Where("cover_url != '' AND cover_url IS NOT NULL AND cover_url LIKE 'http%'").
		Order("RANDOM()").Limit(300).Scan(&coverScenes)
	var deadCoverItems []AffectedItem
	client := &http.Client{Timeout: 4 * time.Second}
	for _, s := range coverScenes {
		if cancelled() {
			publishCancelled()
			return
		}
		resp, err := client.Head(s.CoverURL)
		if err != nil || resp.StatusCode >= 400 {
			deadCoverItems = append(deadCoverItems, AffectedItem{ID: s.ID, Label: s.Title, Extra: s.CoverURL})
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	if len(deadCoverItems) > 0 {
		issues = append(issues, HealthIssue{
			ID: "dead-cover-urls", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d of %d sampled cover URLs are unreachable", len(deadCoverItems), len(coverScenes)),
			Detail:      "Will clear dead cover URLs so the next scrape can re-fetch them",
			Fixable:     true, FixAction: "clear-dead-covers", FixLabel: "Clear Dead Covers",
			AffectedItems: deadCoverItems,
		})
	}

	if cancelled() {
		publishCancelled()
		return
	}

	// -- Step 16: Browser-reported broken cover images --
	publishProgress("Checking browser-reported image errors", 16)
	imageErrorsMu.Lock()
	reportedErrors := make(map[uint]time.Time, len(imageErrors))
	for k, v := range imageErrors {
		reportedErrors[k] = v
	}
	imageErrorsMu.Unlock()

	if len(reportedErrors) > 0 {
		// Look up scene titles for the reported IDs
		ids := make([]uint, 0, len(reportedErrors))
		for id := range reportedErrors {
			ids = append(ids, id)
		}
		var reportedScenes []struct {
			ID    uint
			Title string
		}
		commonDb.Model(&models.Scene{}).Select("id, title").Where("id IN (?)", ids).Scan(&reportedScenes)
		titleMap := make(map[uint]string, len(reportedScenes))
		for _, s := range reportedScenes {
			titleMap[s.ID] = s.Title
		}
		items := make([]AffectedItem, 0, len(reportedErrors))
		for id, ts := range reportedErrors {
			items = append(items, AffectedItem{ID: id, Label: titleMap[id], Extra: ts.Format("15:04:05")})
		}
		issues = append(issues, HealthIssue{
			ID: "browser-image-errors", Category: "scenes", Severity: "warning",
			Description: fmt.Sprintf("%d scenes had cover image load failures in this session", len(items)),
			Detail:      "Will clear cover URLs so the next scrape re-fetches them",
			Fixable:     true, FixAction: "clear-reported-covers", FixLabel: "Clear Failed Covers",
			AffectedItems: items,
		})
	}

	// -- Build summary & store --
	summary := map[string]int{"critical": 0, "warning": 0, "info": 0}
	for _, issue := range issues {
		summary[issue.Severity]++
	}

	report := HealthReport{
		GeneratedAt: time.Now(),
		Duration:    time.Since(start).Round(time.Millisecond).String(),
		Summary:     summary,
		Issues:      issues,
		Stats:       stats,
	}

	lastReportLock.Lock()
	lastReport = &report
	lastReportLock.Unlock()

	publishDone()
}

func FixHealthIssue(action string) error {
	switch action {
	case "refresh-status":
		go RefreshSceneStatuses()

	case "rescan":
		go RescanVolumes(-1)

	case "clean-tags":
		go CleanTags()

	case "generate-previews":
		go GeneratePreviews(nil)

	case "rescrape-scenes":
		go Scrape("_enabled", "", "", false)

	case "clear-broken-images":
		go func() {
			ids := getLastReportItemIDs("broken-images-json")
			if len(ids) == 0 {
				return
			}
			db, _ := models.GetDB()
			defer db.Close()
			db.Model(&models.Scene{}).Where("id IN (?)", ids).Updates(map[string]interface{}{"images": ""})
		}()

	case "clear-dead-covers":
		go func() {
			ids := getLastReportItemIDs("dead-cover-urls")
			if len(ids) == 0 {
				return
			}
			db, _ := models.GetDB()
			defer db.Close()
			db.Model(&models.Scene{}).Where("id IN (?)", ids).Updates(map[string]interface{}{"cover_url": ""})
		}()

	case "clear-reported-covers":
		go func() {
			ids := getLastReportItemIDs("browser-image-errors")
			if len(ids) == 0 {
				return
			}
			db, _ := models.GetDB()
			defer db.Close()
			db.Model(&models.Scene{}).Where("id IN (?)", ids).Updates(map[string]interface{}{"cover_url": ""})
			// Also clear from in-memory error tracking
			imageErrorsMu.Lock()
			for _, id := range ids {
				delete(imageErrors, id)
			}
			imageErrorsMu.Unlock()
		}()

	case "remove-duplicates":
		go func() {
			db, _ := models.GetDB()
			defer db.Close()
			// Find all duplicate scene_ids
			type dupResult struct {
				SceneID string
				Cnt     int
			}
			var dups []dupResult
			db.Model(&models.Scene{}).
				Select("scene_id, count(*) as cnt").Where("scene_id != ''").
				Group("scene_id").Having("count(*) > 1").Scan(&dups)
			for _, d := range dups {
				// Get all records for this scene_id, ordered by file count desc then id asc
				var sceneIDs []uint
				db.Raw(`SELECT s.id FROM scenes s
					LEFT JOIN (SELECT scene_id, count(*) as fc FROM files GROUP BY scene_id) f ON f.scene_id = s.id
					WHERE s.scene_id = ?
					ORDER BY COALESCE(f.fc, 0) DESC, s.id ASC`, d.SceneID).Pluck("id", &sceneIDs)
				if len(sceneIDs) <= 1 {
					continue
				}
				// Keep the first (most files, oldest), delete the rest
				toDelete := sceneIDs[1:]
				db.Where("id IN (?)", toDelete).Delete(&models.Scene{})
			}
		}()

	case "delete-blank-scenes":
		go func() {
			db, _ := models.GetDB()
			defer db.Close()
			// Only delete scenes with no title AND no files
			db.Exec(`DELETE FROM scenes WHERE (title = '' OR title IS NULL)
				AND id NOT IN (SELECT DISTINCT scene_id FROM files WHERE scene_id > 0)`)
		}()

	default:
		return fmt.Errorf("unknown fix action: %s", action)
	}
	return nil
}
