package tasks

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/simple"
	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

type Index struct {
	Bleve bleve.Index
}

type SceneIndexed struct {
	Description string    `json:"description"`
	Title       string    `json:"title"`
	Cast        string    `json:"cast"`
	Site        string    `json:"site"`
	Studio      string    `json:"studio"`
	Id          string    `json:"id"`
	Released    time.Time `json:"released"`
	Added       time.Time `json:"added"`
	Duration    int       `json:"duration"`
}

func NewIndex(name string) (*Index, error) {
	i := new(Index)

	path := filepath.Join(common.IndexDirV2, name)

	// the simple analyzer is more approriate for the title and cast
	// note this does not effect search unless the query includes cast: or title:
	titleFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping.Analyzer = simple.Name
	castFieldMapping := bleve.NewTextFieldMapping()
	castFieldMapping.Analyzer = simple.Name
	releaseFieldMapping := bleve.NewDateTimeFieldMapping()
	addedFieldMapping := bleve.NewDateTimeFieldMapping()
	durationFieldMapping := bleve.NewNumericFieldMapping()
	sceneMapping := bleve.NewDocumentMapping()
	sceneMapping.AddFieldMappingsAt("title", titleFieldMapping)
	sceneMapping.AddFieldMappingsAt("cast", castFieldMapping)
	sceneMapping.AddFieldMappingsAt("released", releaseFieldMapping)
	sceneMapping.AddFieldMappingsAt("added", addedFieldMapping)
	sceneMapping.AddFieldMappingsAt("duration", durationFieldMapping)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("_default", sceneMapping)

	idx, err := bleve.NewUsing(path, mapping, scorch.Name, scorch.Name, nil)
	if err != nil && err == bleve.ErrorIndexPathExists {
		idx, err = bleve.Open(path)
	}
	if err != nil {
		return nil, err
	}

	i.Bleve = idx
	return i, nil
}

func (i *Index) Exist(id string) bool {
	d, err := i.Bleve.Document(id)
	if err != nil || d == nil {
		return false
	}
	return true
}

func sceneDocument(scene models.Scene) SceneIndexed {
	var cast, castConcat strings.Builder
	for _, c := range scene.Cast {
		cast.WriteString(" " + c.Name)
		castConcat.WriteString(" " + strings.ReplaceAll(c.Name, " ", ""))
	}
	return SceneIndexed{
		Title:       scene.Title,
		Description: scene.Synopsis,
		Cast:        cast.String() + " " + castConcat.String(),
		Site:        scene.Site,
		Studio:      scene.Studio,
		Id:          scene.SceneID,
		Released:    time.Date(scene.ReleaseDate.Year(), scene.ReleaseDate.Month(), scene.ReleaseDate.Day(), 0, 0, 0, 0, time.UTC),
		Added:       scene.CreatedAt.Truncate(24 * time.Hour),
		Duration:    scene.Duration,
	}
}

func (i *Index) PutScene(scene models.Scene) error {
	return i.Bleve.Index(scene.SceneID, sceneDocument(scene))
}

const indexBatchSize = 500

func SearchIndex() {
	if !models.CheckLock("index") {
		models.CreateLock("index")
		defer models.RemoveLock("index")

		tlog := log.WithFields(logrus.Fields{"task": "scrape"})

		idx, err := NewIndex("scenes")
		if err != nil {
			log.Error(err)
			models.RemoveLock("index")
			return
		}

		db, _ := models.GetDB()
		defer db.Close()

		total := 0
		current := 0
		var lastID uint
		db.Model(&models.Scene{}).Count(&total)

		tlog.Infof("Building search index...")

		for {
			var scenes []models.Scene
			db.Preload("Cast").Where("id > ?", lastID).Order("id").Limit(indexBatchSize).Find(&scenes)
			if len(scenes) == 0 {
				break
			}
			lastID = scenes[len(scenes)-1].ID

			batch := idx.Bleve.NewBatch()
			for i := range scenes {
				if !idx.Exist(scenes[i].SceneID) {
					if err := batch.Index(scenes[i].SceneID, sceneDocument(scenes[i])); err != nil {
						log.Error(err)
					}
				}
			}
			if err := idx.Bleve.Batch(batch); err != nil {
				log.Error(err)
			}
			current += len(scenes)
			tlog.Infof("Indexed %v/%v scenes", current, total)

			// Update migration status if migration is running
			if config.State.Migration.IsRunning {
				msg := fmt.Sprintf("Reindexing scenes: %v/%v", current, total)
				config.UpdateMigrationStatus(config.State.Migration.Current, current, total, msg)
			}

		}

		idx.Bleve.Close()
		// Release Bleve's segment memory back to the OS after indexing
		debug.FreeOSMemory()

		tlog.Infof("Search index built!")
	}
}

/**
 * Update search index for all of the specified scenes.
 */
func IndexScenes(scenes *[]models.Scene) {
	if !models.CheckLock("index") {
		models.CreateLock("index")
		defer models.RemoveLock("index")

		tlog := log.WithFields(logrus.Fields{"task": "scrape"})

		idx, err := NewIndex("scenes")
		if err != nil {
			log.Error(err)
			models.RemoveLock("index")
			return
		}

		tlog.Infof("Adding scraped scenes to search index...")

		total := 0
		batch := idx.Bleve.NewBatch()
		for i := range *scenes {
			scene := (*scenes)[i]
			if err := batch.Index(scene.SceneID, sceneDocument(scene)); err != nil {
				log.Error(err)
				continue
			}
			total++
			if batch.Size() >= indexBatchSize {
				if err := idx.Bleve.Batch(batch); err != nil {
					log.Error(err)
				}
				batch.Reset()
				tlog.Infof("Indexed %v of %v scenes", total, len(*scenes))
			}
		}
		if err := idx.Bleve.Batch(batch); err != nil {
			log.Error(err)
		}

		idx.Bleve.Close()

		tlog.Infof("Indexed %v scenes", total)
	}
}

func DeleteIndexScenes(scenes *[]models.Scene) {
	if !models.CheckLock("index") {
		models.CreateLock("index")
		defer models.RemoveLock("index")

		tlog := log.WithFields(logrus.Fields{"task": "scrape"})

		idx, err := NewIndex("scenes")
		if err != nil {
			log.Error(err)
			models.RemoveLock("index")
			return
		}

		tlog.Infof("Deleting scenes from search index...")

		total := 0
		lastMessage := time.Now()
		for i := range *scenes {
			if time.Since(lastMessage) > time.Duration(config.Config.Advanced.ProgressTimeInterval)*time.Second {
				tlog.Infof("Deleting scene index %v of %v scenes", total, len(*scenes))
				lastMessage = time.Now()
			}
			scene := (*scenes)[i]
			if idx.Exist(scene.SceneID) {
				// Remove old index, as data may have been updated
				idx.Bleve.Delete(scene.SceneID)
			}
		}

		idx.Bleve.Close()

		tlog.Infof("Indexed %v scenes", total)
	}
}

/**
 * Update search index for all of the specified scrapedScenes.
 * This method will first read the scraped scenes from the DB, after
 * which it calls IndexScenes.
 */
func IndexScrapedScenes(scrapedScenes *[]models.ScrapedScene) {
	// Map scrapedScenes to Scenes
	var scenes []models.Scene
	for i := range *scrapedScenes {
		var scene models.Scene
		scrapedScene := (*scrapedScenes)[i]
		// Read scraped scene from db, as we don't want to index it
		// if it doesn't exist in there
		err := scene.GetIfExist(scrapedScene.SceneID)
		if err == nil {
			scenes = append(scenes, scene)
		}
	}

	// Now update search index
	IndexScenes(&scenes)
}
