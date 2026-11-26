package session

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var (
	sessionSource      string
	isPlaying          bool
	currentPosition    float64
	currentFileID      int
	currentSceneID     uint
	lastSessionID      uint
	lastSessionSceneID uint
	lastSessionStart   time.Time
	lastSessionEnd     time.Time
	sessionPlayLimit   = 2 * time.Hour
)

var currentSessionHeatmap []int

func HasActiveSession() bool {
	return lastSessionID != 0
}

func TrackSessionFromFile(f models.File, doNotTrack string) {
	sessionSource = "file"

	if f.SceneID != 0 && doNotTrack != "true" {
		if lastSessionSceneID != f.SceneID {
			newWatchSession(f.SceneID)
		}

		lastSessionEnd = time.Now()
	}
}

func FinishTrackingFromFile(doNotTrack string) {
	lastSessionEnd = time.Now()
	if doNotTrack != "true" {
		watchSessionFlush()
	}
}

func TrackSessionFromRemote(packet DeoPacket) {
	if packet.Path == "" || packet.Duration == 0 {
		return
	}

	sessionSource = "deovr"
	isPlaying = packet.PlayerState == PLAYING
	currentPosition = packet.CurrentTime

	tmpPath, err := url.Parse(packet.Path)
	if err != nil {
		return
	}
	tmp := strings.Split(tmpPath.Path, "/")
	tmpCurrentFileID, err := strconv.Atoi(tmp[len(tmp)-1])
	if err != nil {
		return
	}

	// Currently playing file has changed
	if tmpCurrentFileID != currentFileID {
		// Get scene ID
		currentFileID = tmpCurrentFileID

		f := models.File{}
		db, _ := models.GetDB()
		_ = db.First(&f, currentFileID).Error
		defer db.Close()

		// Create new session
		if lastSessionSceneID != f.SceneID {
			newWatchSession(f.SceneID)
		}

		currentSessionHeatmap = make([]int, int(packet.Duration))
	}

	// Keep session alive if Deo is playing
	if packet.PlayerState == PLAYING {
		lastSessionEnd = time.Now()

		position := int(packet.CurrentTime)
		if position > 0 && position < len(currentSessionHeatmap) {
			currentSessionHeatmap[position] = currentSessionHeatmap[position] + 1
		}
	}
}

const (
	HereSphereEventOpen = iota
	HereSphereEventPlay
	HereSphereEventPause
	HereSphereEventClose
)

func TrackSessionFromHereSphere(sceneID uint, event int) {
	if sceneID == 0 {
		return
	}
	sessionSource = "heresphere"
	switch event {
	case HereSphereEventOpen, HereSphereEventPlay:
		if lastSessionSceneID != sceneID {
			newWatchSession(sceneID)
		}
		isPlaying = event == HereSphereEventPlay
	case HereSphereEventPause:
		isPlaying = false
	case HereSphereEventClose:
		isPlaying = false
		if lastSessionSceneID == sceneID && HasActiveSession() {
			lastSessionEnd = time.Now()
			watchSessionFlush()
		}
		return
	}
	lastSessionEnd = time.Now()
}

func CheckForDeadSession() {
	var timeout float64
	switch {
	case sessionSource == "file":
		timeout = 60
	case sessionSource == "heresphere" && isPlaying:
		timeout = sessionPlayLimit.Seconds()
	case sessionSource == "heresphere":
		timeout = 600
	default:
		timeout = 5
	}

	if time.Since(lastSessionEnd).Seconds() > timeout && lastSessionSceneID != 0 && HasActiveSession() {
		if sessionSource == "heresphere" && isPlaying {
			lastSessionEnd = lastSessionEnd.Add(sessionPlayLimit)
			isPlaying = false
		}
		watchSessionFlush()
		lastSessionID = 0
		lastSessionSceneID = 0
	}
}

func newWatchSession(sceneID uint) {
	if HasActiveSession() {
		watchSessionFlush()
	}

	lastSessionSceneID = sceneID
	lastSessionStart = time.Now()
	sessionPlayLimit = 2 * time.Hour

	obj := models.History{SceneID: sceneID, TimeStart: lastSessionStart}
	obj.Save()

	var scene models.Scene
	err := scene.GetIfExistByPK(sceneID)
	if err == nil {
		scene.LastOpened = time.Now()
		scene.Save()
		if scene.Duration > 0 {
			sessionPlayLimit = time.Duration(scene.Duration)*time.Minute + 5*time.Minute
		}
	} else {
		return
	}

	lastSessionID = obj.ID
	currentSceneID = scene.ID

	common.Log.Infof("New session #%v for scene #%v from %v", lastSessionID, lastSessionSceneID, sessionSource)
}

func watchSessionFlush() {
	var obj models.History
	err := obj.GetIfExist(lastSessionID)
	if err == nil {
		obj.TimeEnd = lastSessionEnd
		obj.Duration = lastSessionEnd.Sub(lastSessionStart).Seconds()
		obj.Save()

		var scene models.Scene
		err := scene.GetIfExistByPK(lastSessionSceneID)
		if err == nil {
			if !scene.IsWatched {
				scene.IsWatched = true
				scene.Save()
			}
		}

		common.Log.Infof("Session #%v duration for scene #%v is %v", lastSessionID, lastSessionSceneID, time.Since(lastSessionStart).Seconds())

		// Dump heatmap
		// TODO: handle multipart scenes
		if !scene.IsMultipart && sessionSource == "deovr" {
			err = dumpHeatmap(lastSessionSceneID, currentSessionHeatmap)
			if err != nil {
				common.Log.Error("Error while writing heatmap data", err)
			}
		}
	}

	currentFileID = 0
	currentSceneID = 0
	lastSessionID = 0
	lastSessionSceneID = 0
}

func dumpHeatmap(sceneID uint, data []int) error {
	path := path.Join(common.HeatmapDir, fmt.Sprintf("%v.json", sceneID))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create new heatmap
		dataOut, _ := json.Marshal(data)
		os.WriteFile(path, dataOut, 0644)
	} else {
		// Update existing heatmap
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpHeatmap := make([]int, len(data))
		err = json.Unmarshal(b, &tmpHeatmap)
		if err != nil {
			return err
		}

		for k, v := range tmpHeatmap {
			data[k] = data[k] + v
		}

		dataOut, _ := json.Marshal(data)
		os.WriteFile(path, dataOut, 0644)
	}
	return nil
}
