package tasks

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
)

var gpuFailOnce sync.Once

const (
	flatMoments    = 4
	flatMomentSecs = 2.4
	flatFadeSecs   = 0.4
)

// camera is where one moment of a trailer looks, in degrees.
type camera struct {
	yaw, pitch, hfov, vfov float64
}

// trailers are square like classic previews, so they fill the card
const trailerAspect = 1.0

// looking a little down and wide keeps people above the glass that covers the bottom of the card
var defaultCamera = camera{yaw: 0, pitch: -12, hfov: 100, vfov: 100}

// renderFlatPreview makes a short trailer: moments joined by crossfades, one eye shown as an undistorted 90° view.
func renderFlatPreview(inputFile, destFile, projection string, o PreviewOptions) error {
	starts, err := trailerMoments(inputFile, o)
	if err != nil {
		return err
	}
	cams := make([]camera, len(starts))
	for k := range cams {
		cams[k] = defaultCamera
	}
	return renderTrailer(inputFile, destFile, projection, o, starts, cams)
}

// trailerMoments picks where each moment starts: at the scene's cue points when it has some, else evenly spread.
func trailerMoments(inputFile string, o PreviewOptions) ([]float64, error) {
	probe, err := ffprobe.GetProbeData(inputFile, 10*time.Second)
	if err != nil {
		return nil, err
	}
	dur := probe.Format.DurationSeconds
	start := float64(o.StartTime)
	if start*2 >= dur {
		start = 0
	}
	if dur-start < flatMomentSecs*flatMoments {
		return nil, fmt.Errorf("video too short for a trailer preview")
	}
	if cues := cueMoments(o.Cuepoints, start, math.Min(dur-flatMomentSecs, dur*0.95)); len(cues) >= 2 {
		return cues, nil
	}
	step := (dur - start - flatMomentSecs) / float64(flatMoments)
	starts := make([]float64, flatMoments)
	for k := range starts {
		starts[k] = start + step*(float64(k)+0.5)
	}
	return starts, nil
}

// cueMoments chooses up to flatMoments cue points spread over the scene, preferring different positions.
func cueMoments(cues []models.SceneCuepoint, from, to float64) []float64 {
	const settle = 3.0 // seconds after a cue point, once the position is under way
	var all, distinct []float64
	seen := map[string]bool{}
	for _, c := range cues {
		at := c.TimeStart + settle
		if at < from || at > to || (len(all) > 0 && at-all[len(all)-1] < 20) {
			continue
		}
		all = append(all, at)
		if name := strings.ToLower(strings.TrimSpace(c.Name)); !seen[name] {
			seen[name] = true
			distinct = append(distinct, at)
		}
	}
	pool := all
	if len(distinct) >= flatMoments {
		pool = distinct
	}
	if len(pool) <= flatMoments {
		return pool
	}
	out := make([]float64, flatMoments)
	for k := range out {
		out[k] = pool[k*(len(pool)-1)/(flatMoments-1)]
	}
	return out
}

// renderTrailer joins the moments with crossfades, each seen through its own camera, in the video's real colours.
// Moments are cut one at a time into small clips, so only one full-size decoder is ever in memory.
func renderTrailer(inputFile, destFile, projection string, o PreviewOptions, starts []float64, cams []camera) error {
	h := o.Resolution / 2 * 2
	w := int(float64(h)*trailerAspect) / 2 * 2
	var clips []string
	defer func() {
		for _, c := range clips {
			os.Remove(c)
		}
	}()
	for k, at := range starts {
		clip := fmt.Sprintf("%s.m%d.mp4", destFile, k)
		clips = append(clips, clip)
		vf := view(projection, cams[k], w, h) + ",scale=in_color_matrix=bt709:out_color_matrix=bt709,fps=30,format=yuv420p"
		if err := runWithGPU([]string{"-y", "-threads", "2", "-ss", fmt.Sprintf("%.2f", at), "-t", fmt.Sprintf("%.2f", flatMomentSecs), "-i", inputFile,
			"-vf", vf, "-an", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "12", "-threads", "2", clip}); err != nil {
			return err
		}
	}

	var g strings.Builder
	args := []string{"-y"}
	for k, c := range clips {
		args = append(args, "-i", c)
		fmt.Fprintf(&g, "[%d:v]null[m%d];", k, k)
	}
	// the end fades back into the opening, whose first frames are trimmed, so the loop has no seam
	fmt.Fprintf(&g, "[m0]split[a0][h0];[h0]trim=0:%.1f,setpts=PTS-STARTPTS[head];", flatFadeSecs)
	prev := "a0"
	for k := 1; k < len(clips); k++ {
		out := fmt.Sprintf("x%d", k)
		fmt.Fprintf(&g, "[%s][m%d]xfade=transition=fade:duration=%.1f:offset=%.2f[%s];", prev, k, flatFadeSecs, float64(k)*(flatMomentSecs-flatFadeSecs), out)
		prev = out
	}
	total := float64(len(clips))*flatMomentSecs - float64(len(clips)-1)*flatFadeSecs
	fmt.Fprintf(&g, "[%s][head]xfade=transition=fade:duration=%.1f:offset=%.2f[loop];", prev, flatFadeSecs, total-flatFadeSecs)
	fmt.Fprintf(&g, "[loop]trim=start=%.1f,setpts=PTS-STARTPTS,setparams=color_primaries=bt709:color_trc=bt709:colorspace=bt709:range=tv[out]", flatFadeSecs)

	enc := []string{"-c:v", "libx265", "-crf", "26", "-preset", "fast", "-tag:v", "hvc1", "-x265-params", "log-level=error:pools=2:frame-threads=1"}
	if o.Codec == "h264" {
		enc = []string{"-c:v", "libx264", "-crf", "23", "-preset", "fast"}
	}
	tmp := destFile + ".tmp.mp4"
	defer os.Remove(tmp)
	args = append(args, "-filter_complex", g.String(), "-map", "[out]", "-an", "-threads", "2")
	args = append(args, enc...)
	args = append(args, "-pix_fmt", "yuv420p", "-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", "-color_range", "tv", "-movflags", "+faststart", tmp)
	if err := buildCmd(GetBinPath("ffmpeg"), args...).Run(); err != nil {
		return err
	}
	return os.Rename(tmp, destFile)
}

// lens returns how the recording is stored: its v360 input and stereo layout; flat videos return "".
func lens(projection string) (input, stereo string) {
	p := strings.ToLower(projection)
	if p == "" || p == "flat" {
		return "", ""
	}
	stereo = "sbs"
	if strings.Contains(p, "tb") {
		stereo = "tb"
	} else if strings.Contains(p, "mono") {
		stereo = "2d"
	}
	input = "hequirect"
	switch {
	case strings.HasPrefix(p, "360"):
		input = "equirect"
	case strings.Contains(p, "fisheye"), strings.HasPrefix(p, "mkx"), strings.HasPrefix(p, "rf52"), strings.HasPrefix(p, "vrca"):
		fov := 190
		for _, n := range []int{200, 220} {
			if strings.Contains(p, fmt.Sprint(n)) {
				fov = n
			}
		}
		input = fmt.Sprintf("fisheye:ih_fov=%d:iv_fov=%d", fov, fov)
	}
	return input, stereo
}

// view is the filter that shows one eye of the recording through a flat camera.
func view(projection string, c camera, w, h int) string {
	input, stereo := lens(projection)
	if input == "" {
		return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", w, h, w, h)
	}
	return fmt.Sprintf("v360=input=%s:in_stereo=%s:output=flat:out_stereo=2d:yaw=%.1f:pitch=%.1f:h_fov=%.1f:v_fov=%.1f:w=%d:h=%d", input, stereo, c.yaw, c.pitch, c.hfov, c.vfov, w, h)
}

// runWithGPU runs ffmpeg decoding every input on the GPU when one is passed in, and on the CPU otherwise.
func runWithGPU(args []string) error {
	if _, err := os.Stat(gpuDevice); err == nil {
		var gpu []string
		for _, a := range args {
			if a == "-ss" {
				gpu = append(gpu, "-hwaccel", "vaapi", "-hwaccel_device", gpuDevice)
			}
			gpu = append(gpu, a)
		}
		out, err := buildCmd(GetBinPath("ffmpeg"), gpu...).CombinedOutput()
		if err == nil {
			return nil
		}
		gpuFailOnce.Do(func() {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			log.WithField("task", "previews").Warnf("GPU decoding failed, using the CPU: %s", lines[len(lines)-1])
		})
	}
	return buildCmd(GetBinPath("ffmpeg"), args...).Run()
}
