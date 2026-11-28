package tasks

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/ffprobe"
)

const (
	flatMoments    = 4
	flatMomentSecs = 2.4
	flatFadeSecs   = 0.4
)

// camera is where one moment of a trailer looks, in degrees.
type camera struct {
	yaw, pitch, hfov, vfov float64
}

var defaultCamera = camera{yaw: 0, pitch: -8, hfov: 90, vfov: 59}

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

// trailerMoments spreads the moments evenly over the video after the start time.
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
	step := (dur - start - flatMomentSecs) / float64(flatMoments)
	starts := make([]float64, flatMoments)
	for k := range starts {
		starts[k] = start + step*(float64(k)+0.5)
	}
	return starts, nil
}

// renderTrailer joins the moments with crossfades, each seen through its own camera, in the video's real colours.
func renderTrailer(inputFile, destFile, projection string, o PreviewOptions, starts []float64, cams []camera) error {
	h := o.Resolution
	w := (h*16/9 + 1) / 2 * 2
	// each moment is its own input seeking straight to its spot, so only those seconds are decoded
	var inputs []string
	var g strings.Builder
	for k, at := range starts {
		inputs = append(inputs, "-ss", fmt.Sprintf("%.2f", at), "-t", fmt.Sprintf("%.2f", flatMomentSecs), "-i", inputFile)
		fmt.Fprintf(&g, "[%d:v]setpts=PTS-STARTPTS,%s,scale=in_color_matrix=bt709:out_color_matrix=bt709,fps=30,format=yuv420p[m%d];", k, view(projection, cams[k], w, h), k)
	}
	prev := "m0"
	for k := 1; k < len(starts); k++ {
		out := fmt.Sprintf("x%d", k)
		fmt.Fprintf(&g, "[%s][m%d]xfade=transition=fade:duration=%.1f:offset=%.2f[%s];", prev, k, flatFadeSecs, float64(k)*(flatMomentSecs-flatFadeSecs), out)
		prev = out
	}
	fmt.Fprintf(&g, "[%s]setparams=color_primaries=bt709:color_trc=bt709:colorspace=bt709:range=tv[out]", prev)

	enc := []string{"-c:v", "libx265", "-crf", "26", "-preset", "fast", "-tag:v", "hvc1", "-x265-params", "log-level=error:pools=2:frame-threads=1"}
	if o.Codec == "h264" {
		enc = []string{"-c:v", "libx264", "-crf", "23", "-preset", "fast"}
	}
	tmp := destFile + ".tmp.mp4"
	defer os.Remove(tmp)
	args := append(append([]string{"-y"}, inputs...), "-filter_complex", g.String(), "-map", "[out]", "-an", "-threads", "2")
	args = append(args, enc...)
	args = append(args, "-pix_fmt", "yuv420p", "-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", "-color_range", "tv", "-movflags", "+faststart", tmp)
	if err := runWithGPU(args); err != nil {
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
		if buildCmd(GetBinPath("ffmpeg"), gpu...).Run() == nil {
			return nil
		}
	}
	return buildCmd(GetBinPath("ffmpeg"), args...).Run()
}
