package models

import (
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"

	"github.com/xbapps/xbvr/pkg/common"
)

const (
	sceneEventInterval = 2 * time.Second
	// above this many changes in one interval the UI reloads its list instead of patching it
	sceneEventMaxIDs = 500
)

var (
	sceneEventsMu   sync.Mutex
	sceneEventIDs   = map[uint]struct{}{}
	sceneEventsOnce sync.Once
)

// NotifySceneChanged queues a "scenes.changed" websocket message; changes are bundled every few seconds.
func NotifySceneChanged(ids ...uint) {
	sceneEventsOnce.Do(func() { go flushSceneEvents() })
	sceneEventsMu.Lock()
	defer sceneEventsMu.Unlock()
	for _, id := range ids {
		if id != 0 {
			sceneEventIDs[id] = struct{}{}
		}
	}
}

func flushSceneEvents() {
	for range time.Tick(sceneEventInterval) {
		sceneEventsMu.Lock()
		if len(sceneEventIDs) == 0 {
			sceneEventsMu.Unlock()
			continue
		}
		ids := make([]uint, 0, len(sceneEventIDs))
		for id := range sceneEventIDs {
			ids = append(ids, id)
		}
		sceneEventIDs = map[uint]struct{}{}
		sceneEventsMu.Unlock()

		if len(ids) > sceneEventMaxIDs {
			_ = common.PublishWS("scenes.changed", map[string]interface{}{"many": true})
		} else {
			_ = common.PublishWS("scenes.changed", map[string]interface{}{"ids": ids})
		}
	}
}

// AfterSave is a gorm hook: every saved scene reports itself, whatever changed it (scrape, rescan, edit).
func (i *Scene) AfterSave() error {
	NotifySceneChanged(i.ID)
	return nil
}

// SceneListPositions returns where each of ids sits in the full result of r (ids that do not match are absent).
func SceneListPositions(r RequestSceneList, ids []uint) map[uint]int {
	db, _ := GetDB()
	defer db.Close()
	return sceneListPositions(db, r, ids)
}

func sceneListPositions(db *gorm.DB, r RequestSceneList, ids []uint) map[uint]int {
	wanted := make(map[uint]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	r.Limit = optional.Int{}
	r.Offset = optional.Int{}
	_, finalTx := queryScenes(db, r)

	var all []uint
	finalTx.Pluck("scenes.id", &all)
	out := map[uint]int{}
	for pos, id := range all {
		if wanted[id] {
			out[id] = pos
		}
	}
	return out
}

// SceneCards loads scenes by id with what scene cards need (files and cuepoint counts).
func SceneCards(ids []uint) ([]Scene, map[uint]int) {
	db, _ := GetDB()
	defer db.Close()
	var scenes []Scene
	if len(ids) == 0 {
		return scenes, map[uint]int{}
	}
	db.Preload("Files").Where("id IN (?)", ids).Find(&scenes)
	return scenes, cuepointCounts(db, ids)
}
