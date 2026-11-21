package tasks

import (
	"math"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/xbapps/xbvr/pkg/models"
)

// ScriptStatsVersion is bumped when the way stats are computed changes, so every
// script is measured again on the next run.
const ScriptStatsVersion = 2

// pauses longer than this are left out of strokes per minute (a still section
// should not make a fast script look slow)
const scriptPauseMs = 5000

// ScriptStats are the numbers shown for a funscript and used for sorting.
//   - AvgSpeed: mean of the speeds between consecutive actions, in position units
//     (0-100) per second, over every interval of the script (pauses and equal
//     timestamps included, as 0 or slow intervals). This is the usual funscript
//     "average speed" so the numbers match other tools.
//   - MaxSpeed: fastest single movement, same unit
//   - StrokesPerMin: up+down cycles per minute (two turning points = one stroke)
//   - Depth: average travel between turning points, in percent of the full range
type ScriptStats struct {
	Actions       int
	AvgSpeed      float64
	MaxSpeed      float64
	StrokesPerMin float64
	Depth         float64
}

func computeScriptStats(actions []Action) ScriptStats {
	st := ScriptStats{Actions: len(actions)}
	if len(actions) < 2 {
		return st
	}
	var movingMs, speedSum float64
	turns := 0
	depthSum := 0.0
	lastTurnPos := actions[0].Pos
	prevDir := 0
	for i := 1; i < len(actions); i++ {
		dt := float64(actions[i].At - actions[i-1].At)
		dp := actions[i].Pos - actions[i-1].Pos
		if dt > 0 {
			speedSum += math.Abs(float64(dp)) / dt * 1000
		}
		// strokes per minute and the peak only look at real movement, not pauses
		if dt <= 0 || dt > scriptPauseMs {
			continue
		}
		movingMs += dt
		// ignore near-instant points when looking for the peak (rounding noise)
		if dt >= 50 {
			if v := math.Abs(float64(dp)) / (dt / 1000); v > st.MaxSpeed {
				st.MaxSpeed = v
			}
		}
		dir := 0
		if dp > 0 {
			dir = 1
		} else if dp < 0 {
			dir = -1
		}
		if dir != 0 && prevDir != 0 && dir != prevDir {
			turns++
			depthSum += math.Abs(float64(actions[i-1].Pos - lastTurnPos))
			lastTurnPos = actions[i-1].Pos
		}
		if dir != 0 {
			prevDir = dir
		}
	}
	st.AvgSpeed = speedSum / float64(len(actions)-1)
	if movingMs > 0 {
		st.StrokesPerMin = float64(turns) / 2 / (movingMs / 60000)
	}
	if turns > 0 {
		st.Depth = depthSum / float64(turns)
	}
	round := func(v float64) float64 { return math.Round(v*10) / 10 }
	st.AvgSpeed, st.MaxSpeed, st.StrokesPerMin, st.Depth = round(st.AvgSpeed), round(st.MaxSpeed), round(st.StrokesPerMin), round(st.Depth)
	return st
}

// applyScriptInfo copies stats and metadata of a parsed funscript onto its file.
func applyScriptInfo(f *models.File, s Script) {
	st := computeScriptStats(s.Actions)
	f.ScriptActions = st.Actions
	f.ScriptAvgSpeed = st.AvgSpeed
	f.ScriptMaxSpeed = st.MaxSpeed
	f.ScriptStrokesPerMin = st.StrokesPerMin
	f.ScriptDepth = st.Depth
	f.ScriptCreator, f.ScriptDescription, f.ScriptLicense = "", "", ""
	if m := s.Metadata; m != nil {
		f.ScriptCreator = strings.TrimSpace(m.Creator)
		f.ScriptDescription = strings.TrimSpace(m.Description)
		f.ScriptLicense = strings.TrimSpace(m.License)
	}
	f.ScriptStatsVersion = ScriptStatsVersion
}

// UpdateScriptStats measures every local funscript whose stats are missing or
// were computed by an older version. Unreadable files are marked so they are not
// retried on every run.
func UpdateScriptStats(tlog *logrus.Entry) {
	if !models.CheckLock("scriptstats") {
		models.CreateLock("scriptstats")
		defer models.RemoveLock("scriptstats")

		db, _ := models.GetDB()
		defer db.Close()

		var files []models.File
		db.Model(&models.File{}).Preload("Volume").
			Where("type = ? AND script_stats_version < ?", "script", ScriptStatsVersion).Find(&files)
		done := 0
		for i := range files {
			f := &files[i]
			if !f.Exists() || !strings.HasSuffix(strings.ToLower(f.Filename), ".funscript") {
				continue
			}
			s, err := LoadFunscriptData(f.GetPath())
			if err != nil {
				f.ScriptStatsVersion = ScriptStatsVersion
				f.Save()
				continue
			}
			applyScriptInfo(f, s)
			f.Save()
			done++
		}
		if tlog != nil && done > 0 {
			tlog.Infof("Measured %d funscript(s)", done)
		}
	}
}
