package tasks

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/xbapps/xbvr/pkg/models"
)

// Second matching pass, run after the exact filename match in RescanVolumes.
// It only looks at files that are still unmatched and only links a file when the
// answer is unambiguous, so it never overrides or guesses between scenes.
//
//  1. Normalised name: file and scene filenames are compared after dropping the
//     extension, quality/projection/codec tags, script variant suffixes (_ai, _2d,
//     _ow) and repeated words. Links when exactly one scene has that name.
//  2. Sibling video: a script/hsp/subtitle file is linked to the scene of a matched
//     video in the same folder whose normalised name its own name starts with
//     ("clip.mkv" + "clip.mp4.funscript", "clip - Script v2.funscript", ...).

var (
	matchSplitRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	// tokens that describe the encode, not the scene
	matchNoiseRe = regexp.MustCompile(`^(\d+k|\d{3,4}p|\d+x\d+|\d+fps|h26[45]|x26[45]|hevc|avc|av1|vp9|hdr|sdr|` +
		`180|360|3d|3dh|3dv|2d|lr|tb|sbs|ou|mono|fisheye\d*|mkx\d*|rf\d+|vrca\d*|fb360|` +
		`oculus\w*|smartphone|gear|hq|lq|uhq|original|ai|ow|mp4|mkv|funscript|cmscript|hsp|srt|ssa|ass|script)$`)
	scriptLikeTypes = map[string]bool{"script": true, "hsp": true, "subtitles": true}
)

// matchTokens returns the meaningful, lower-cased words of a file name.
func matchTokens(name string) []string {
	base := strings.ToLower(path.Base(name))
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	var out []string
	for _, t := range matchSplitRe.Split(base, -1) {
		if t == "" || matchNoiseRe.MatchString(t) {
			continue
		}
		if len(out) > 0 && out[len(out)-1] == t { // "Golden_Golden_Coast" -> golden coast
			continue
		}
		out = append(out, t)
	}
	return out
}

// matchKey is the normalised name, or "" when too little is left to be safe.
func matchKey(name string) string {
	toks := matchTokens(name)
	key := strings.Join(toks, " ")
	if len(toks) < 2 || len(key) < 10 {
		return ""
	}
	return key
}

// hasTokenPrefix reports whether toks starts with all of prefix.
func hasTokenPrefix(toks, prefix []string) bool {
	if len(prefix) == 0 || len(prefix) > len(toks) {
		return false
	}
	for i := range prefix {
		if toks[i] != prefix[i] {
			return false
		}
	}
	return true
}

func matchUnmatchedFiles(db *gorm.DB, tlog *logrus.Entry) {
	var files []models.File
	db.Where("scene_id = 0").Find(&files)
	if len(files) == 0 {
		return
	}

	// normalised scene filename -> scene ids
	byKey := map[string]map[uint]bool{}
	var scenes []models.Scene
	db.Select("id, filenames_arr").Where("filenames_arr <> '' AND filenames_arr <> '[]'").Find(&scenes)
	for _, s := range scenes {
		var names []string
		if json.Unmarshal([]byte(s.FilenamesArr), &names) != nil {
			continue
		}
		for _, n := range names {
			if k := matchKey(n); k != "" {
				if byKey[k] == nil {
					byKey[k] = map[uint]bool{}
				}
				byKey[k][s.ID] = true
			}
		}
	}

	linked := map[uint]bool{}
	link := func(f *models.File, sceneID uint, how string) {
		f.SceneID = sceneID
		f.Save()
		linked[sceneID] = true
		tlog.Infof("Matched %s to scene %d (%s)", f.Filename, sceneID, how)
	}

	byName := 0
	for i := range files {
		k := matchKey(files[i].Filename)
		if k == "" || len(byKey[k]) != 1 {
			continue
		}
		for id := range byKey[k] {
			link(&files[i], id, "normalised name")
			byName++
		}
	}

	bySibling := 0
	videosByPath := map[string][]models.File{}
	for i := range files {
		f := &files[i]
		if f.SceneID != 0 || !scriptLikeTypes[f.Type] {
			continue
		}
		if _, ok := videosByPath[f.Path]; !ok {
			var vids []models.File
			db.Where("path = ? AND type = ? AND scene_id > 0", f.Path, "video").Find(&vids)
			videosByPath[f.Path] = vids
		}
		toks := matchTokens(f.Filename)
		best, bestLen, ambiguous := uint(0), 0, false
		for _, v := range videosByPath[f.Path] {
			vt := matchTokens(v.Filename)
			if len(vt) < 2 || !hasTokenPrefix(toks, vt) {
				continue
			}
			switch {
			case len(vt) > bestLen:
				best, bestLen, ambiguous = v.SceneID, len(vt), false
			case len(vt) == bestLen && v.SceneID != best:
				ambiguous = true
			}
		}
		if best != 0 && !ambiguous {
			link(f, best, "next to its video")
			bySibling++
		}
	}

	for id := range linked {
		var s models.Scene
		if s.GetIfExistByPK(id) == nil {
			s.UpdateStatus()
		}
	}
	if byName+bySibling > 0 {
		tlog.Infof("Second matching pass linked %d file(s) by normalised name and %d by sibling video", byName, bySibling)
	}
}
