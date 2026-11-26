package session

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var (
	mu                 sync.Mutex
	sessionSource      string
	isPlaying          bool
	currentPosition    float64
	currentFileID      int
	currentSceneID     uint
	lastSessionID      uint
	lastSessionSceneID uint
	lastSessionStart   time.Time
	lastSessionEnd     time.Time
	sceneLength        time.Duration
	// HereSphere playback reported at the last event, used to credit time when events stop
	playPosition time.Duration
	playSpeed    float64
)

var currentSessionHeatmap []int

const (
	defaultSceneLength = 2 * time.Hour
	playGrace          = 5 * time.Minute
	pausedTimeout      = 10 * time.Minute
)

func HasActiveSession() bool {
	mu.Lock()
	defer mu.Unlock()
	return active()
}

func active() bool {
	return lastSessionID != 0
}

// enterSession flushes the previous session under its own source before switching to the new one.
func enterSession(sceneID uint, source string) {
	if lastSessionSceneID != sceneID {
		if active() {
			watchSessionFlush()
		}
		sessionSource = source
		newWatchSession(sceneID)
	}
	sessionSource = source
}

func TrackSessionFromFile(f models.File, doNotTrack string) {
	if f.SceneID == 0 || doNotTrack == "true" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	enterSession(f.SceneID, "file")
	lastSessionEnd = time.Now()
}

func FinishTrackingFromFile(doNotTrack string) {
	mu.Lock()
	defer mu.Unlock()
	lastSessionEnd = time.Now()
	if doNotTrack != "true" {
		watchSessionFlush()
	}
}

func TrackSessionFromRemote(packet DeoPacket) {
	if packet.Path == "" || packet.Duration == 0 {
		return
	}
	tmpPath, err := url.Parse(packet.Path)
	if err != nil {
		return
	}
	tmp := strings.Split(tmpPath.Path, "/")
	fileID, err := strconv.Atoi(tmp[len(tmp)-1])
	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	isPlaying = packet.PlayerState == PLAYING
	currentPosition = packet.CurrentTime

	if fileID != currentFileID {
		f := models.File{}
		db, _ := models.GetDB()
		_ = db.First(&f, fileID).Error
		db.Close()

		enterSession(f.SceneID, "deovr")
		currentFileID = fileID
		currentSessionHeatmap = make([]int, int(packet.Duration))
	}
	sessionSource = "deovr"

	if isPlaying {
		lastSessionEnd = time.Now()
		position := int(packet.CurrentTime)
		if position > 0 && position < len(currentSessionHeatmap) {
			currentSessionHeatmap[position]++
		}
	}
}

const (
	HereSphereEventOpen = iota
	HereSphereEventPlay
	HereSphereEventPause
	HereSphereEventClose
)

// TrackSessionFromHereSphere handles a playback event; position is the playback time in milliseconds.
func TrackSessionFromHereSphere(sceneID uint, event int, position float64, speed float64) {
	if sceneID == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	switch event {
	case HereSphereEventOpen, HereSphereEventPlay:
		enterSession(sceneID, "heresphere")
		isPlaying = event == HereSphereEventPlay
		playPosition = time.Duration(position) * time.Millisecond
		playSpeed = speed
		if playSpeed <= 0 {
			playSpeed = 1
		}
	case HereSphereEventPause:
		isPlaying = false
	case HereSphereEventClose:
		isPlaying = false
		if lastSessionSceneID == sceneID && active() {
			lastSessionEnd = time.Now()
			watchSessionFlush()
		}
		return
	}
	lastSessionEnd = time.Now()
}

// remainingPlayback is how long HereSphere keeps playing after the last event if nothing stops it.
func remainingPlayback() time.Duration {
	left := sceneLength - playPosition
	if left < 0 {
		left = 0
	}
	return time.Duration(float64(left) / playSpeed)
}

func CheckForDeadSession() {
	mu.Lock()
	defer mu.Unlock()
	if lastSessionSceneID == 0 || !active() {
		return
	}
	var timeout time.Duration
	switch {
	case sessionSource == "file":
		timeout = time.Minute
	case sessionSource == "heresphere" && isPlaying:
		timeout = remainingPlayback() + playGrace
	case sessionSource == "heresphere":
		timeout = pausedTimeout
	default:
		timeout = 5 * time.Second
	}
	if time.Since(lastSessionEnd) <= timeout {
		return
	}
	if sessionSource == "heresphere" && isPlaying {
		lastSessionEnd = lastSessionEnd.Add(remainingPlayback())
		isPlaying = false
	}
	watchSessionFlush()
}

// RemoteState is the playback state shown in the UI.
func RemoteState() map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()
	return map[string]interface{}{
		"isPlaying":       isPlaying,
		"currentPosition": currentPosition,
		"sessionStart":    lastSessionStart,
		"sessionEnd":      lastSessionEnd,
		"currentFileID":   currentFileID,
		"currentSceneID":  currentSceneID,
	}
}

func newWatchSession(sceneID uint) {
	lastSessionSceneID = sceneID
	lastSessionStart = time.Now()
	sceneLength = defaultSceneLength

	obj := models.History{SceneID: sceneID, TimeStart: lastSessionStart}
	obj.Save()

	var scene models.Scene
	if err := scene.GetIfExistByPK(sceneID); err != nil {
		return
	}
	scene.LastOpened = time.Now()
	scene.Save()
	if scene.Duration > 0 {
		sceneLength = time.Duration(scene.Duration) * time.Minute
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
