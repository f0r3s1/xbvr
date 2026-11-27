package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darwayne/go-timecode/timecode"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
)

func GeneratePreviews(endTime *time.Time) {
	tlog := log.WithField("task", "previews")
	if models.CheckLock("previews") {
		tlog.Info("Preview generation is already running")
		return
	}
	models.CreateLock("previews")
	defer models.RemoveLock("previews")
	tlog.Infof("Generating previews")
	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Model(&models.Scene{}).Where("is_available = ?", true).Where("has_video_preview = ?", false).Order("release_date desc").Find(&scenes)

	for _, scene := range scenes {
		files, _ := scene.GetFiles()
		if len(files) > 0 {
			if endTime != nil && time.Now().After(*endTime) {
				return
			}
			i := 0
			for i < len(files) && files[i].Exists() {
				if files[i].Type == "video" {
					tlog.Infof("Rendering %v", scene.SceneID)
					destFile := filepath.Join(common.VideoPreviewDir, scene.SceneID+".mp4")
					method, err := RenderPreviewWith(config.Config.Library.Preview.Method, files[i].GetPath(), destFile, files[i].VideoProjection, PreviewOptionsFromConfig())
					if err == nil {
						scene.HasVideoPreview = true
						scene.PreviewMethod = method
						scene.Save()
						break
					} else {
						tlog.Warn(err)
					}
				}
				i++
			}
		}
	}
	tlog.Infof("Previews generated")
}

func RenderPreview(inputFile string, destFile string, videoProjection string, startTime int, snippetLength float64, snippetAmount int, resolution int, extraSnippet bool) error {
	tmpPath := filepath.Join(common.VideoPreviewDir, "tmp")
	os.MkdirAll(tmpPath, os.ModePerm)
	defer os.RemoveAll(tmpPath)

	// Get video duration
	ffdata, err := ffprobe.GetProbeData(inputFile, time.Second*10)
	if err != nil {
		return err
	}
	vs := ffdata.GetFirstVideoStream()
	dur := ffdata.Format.DurationSeconds

	crop := "iw/2:ih:iw/2:ih" // LR videos
	if vs.Height == vs.Width {
		crop = "iw/2:ih/2:iw/4:ih/2" // TB videos
	}
	if videoProjection == "flat" {
		crop = "iw:ih:iw:ih" // LR videos
	}
	// Mono 360 crop args: (no way of accurately determining)
	// "iw/2:ih:iw/4:ih"
	vfArgs := fmt.Sprintf("crop=%v,scale=%v:%v", crop, resolution, resolution)

	// Prepare snippets; short clips start at 0 and the last snippet must end before the video does
	if float64(startTime)*2 >= dur {
		startTime = 0
	}
	interval := (dur - float64(startTime) - snippetLength) / float64(snippetAmount)
	for i := 1; i <= snippetAmount; i++ {
		start := time.Duration(float64(i)*interval+float64(startTime)) * time.Second
		if err := renderSnippet(inputFile, filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", i)), start, snippetLength, vfArgs); err != nil {
			return err
		}
	}

	// Ensure ending is always in preview
	if extraSnippet && dur/float64(snippetAmount) > float64(150) {
		snippetAmount++
		start := time.Duration(dur-float64(150)) * time.Second
		if err := renderSnippet(inputFile, filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", snippetAmount)), start, snippetLength, vfArgs); err != nil {
			return err
		}
	}

	// Prepare concat file
	concatFile := filepath.Join(tmpPath, "concat.txt")
	f, err := os.Create(concatFile)
	if err != nil {
		return err
	}
	for i := 1; i <= snippetAmount; i++ {
		f.WriteString(fmt.Sprintf("file '%v.mp4'\n", i))
	}
	f.Close()

	// Save result
	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", filepath.ToSlash(concatFile),
		"-c", "copy",
		filepath.ToSlash(destFile),
	}
	cmd := buildCmd(GetBinPath("ffmpeg"), args...)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

const gpuDevice = "/dev/dri/renderD128"

// renderSnippet cuts one preview snippet, decoding on the GPU when one is available and on the CPU otherwise.
func renderSnippet(inputFile, outFile string, start time.Duration, length float64, vf string) error {
	args := []string{
		"-y",
		"-ss", strings.TrimSuffix(timecode.New(start, timecode.IdentityRate).String(), ":00"),
		"-i", inputFile,
		"-vf", vf,
		"-pix_fmt", "yuv420p",
		"-t", fmt.Sprintf("%v", length),
		"-an", outFile,
	}
	if _, err := os.Stat(gpuDevice); err == nil {
		gpu := append([]string{"-hwaccel", "vaapi", "-hwaccel_device", gpuDevice}, args...)
		if buildCmd(GetBinPath("ffmpeg"), gpu...).Run() == nil {
			return nil
		}
	}
	return buildCmd(GetBinPath("ffmpeg"), args...).Run()
}

// RegeneratePreview remakes one scene's preview with the given method, replacing the current one.
func RegeneratePreview(sceneID uint, method string) error {
	tlog := log.WithField("task", "previews")
	var scene models.Scene
	if err := scene.GetIfExistByPK(sceneID); err != nil {
		return err
	}
	files, _ := scene.GetFiles()
	for _, f := range files {
		if f.Type != "video" || !f.Exists() {
			continue
		}
		tlog.Infof("Rendering %v (%s)", scene.SceneID, method)
		used, err := RenderPreviewWith(method, f.GetPath(), filepath.Join(common.VideoPreviewDir, scene.SceneID+".mp4"), f.VideoProjection, PreviewOptionsFromConfig())
		if err != nil {
			tlog.Warn(err)
			return err
		}
		scene.HasVideoPreview = true
		scene.PreviewMethod = used
		scene.Save()
		tlog.Infof("Preview ready for %v", scene.SceneID)
		return nil
	}
	return fmt.Errorf("scene has no video file")
}
