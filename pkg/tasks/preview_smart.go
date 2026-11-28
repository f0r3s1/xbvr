package tasks

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"sort"
	"time"
)

const (
	wideW, wideH       = 320, 240
	wideHFov, wideVFov = 150.0, 140.0
)

// renderSmartPreview is the flat trailer with each moment framed on the people in it.
func renderSmartPreview(inputFile, destFile, projection string, o PreviewOptions) error {
	starts, err := trailerMoments(inputFile, o)
	if err != nil {
		return err
	}
	input, stereo := lens(projection)
	cams := make([]camera, len(starts))
	began := time.Now()
	for k, at := range starts {
		cams[k] = defaultCamera
		if input == "" {
			continue
		}
		if img, err := grabWide(inputFile, input, stereo, at+flatMomentSecs/2); err == nil {
			cams[k] = aim(img, wideW, wideH)
		}
	}
	log.WithField("task", "previews").Infof("%s: aimed in %s", filepath.Base(destFile), time.Since(began).Round(100*time.Millisecond))
	return renderTrailer(inputFile, destFile, projection, o, starts, cams)
}

// grabWide returns one wide, undistorted RGB frame of one eye.
func grabWide(inputFile, input, stereo string, at float64) ([]byte, error) {
	vf := fmt.Sprintf("v360=input=%s:in_stereo=%s:output=flat:out_stereo=2d:h_fov=%.0f:v_fov=%.0f:w=%d:h=%d,format=rgb24", input, stereo, wideHFov, wideVFov, wideW, wideH)
	out, err := buildCmd(GetBinPath("ffmpeg"), "-v", "error", "-skip_frame", "nokey", "-noaccurate_seek", "-ss", fmt.Sprintf("%.2f", at), "-i", inputFile, "-frames:v", "1", "-vf", vf, "-f", "rawvideo", "pipe:1").Output()
	if err != nil {
		return nil, err
	}
	if len(out) < wideW*wideH*3 {
		return nil, fmt.Errorf("short frame")
	}
	return out, nil
}

func isSkin(r, g, b byte) bool {
	y, cb, cr := color.RGBToYCbCr(r, g, b)
	return y > 40 && cb >= 77 && cb <= 127 && cr >= 133 && cr <= 173
}

// aim frames the skin in a wide frame: the whole body when it fits, otherwise its top so the head stays in.
func aim(img []byte, w, h int) camera {
	tanH, tanV := math.Tan(rad(wideHFov/2)), math.Tan(rad(wideVFov/2))
	yawAt := func(x float64) float64 { return deg(math.Atan((x/float64(w)*2 - 1) * tanH)) }
	pitchAt := func(y float64) float64 { return deg(math.Atan((1 - y/float64(h)*2) * tanV)) }
	var xs, ys []int
	for y := 0; y < h; y++ {
		if pitchAt(float64(y)) > 20 {
			continue // ceilings and lamps above the people are often skin coloured
		}
		for x := 0; x < w; x++ {
			i := (y*w + x) * 3
			if isSkin(img[i], img[i+1], img[i+2]) {
				xs = append(xs, x)
				ys = append(ys, y)
			}
		}
	}
	if len(xs) < w*h/100 {
		return defaultCamera
	}
	sort.Ints(xs)
	sort.Ints(ys)
	pick := func(v []int, q float64) float64 { return float64(v[int(q*float64(len(v)-1))]) + 0.5 }
	left, right := yawAt(pick(xs, 0.05)), yawAt(pick(xs, 0.95))
	top, bottom := pitchAt(pick(ys, 0.03)), pitchAt(pick(ys, 0.97))

	hfov := math.Max((right-left)*1.6, deg(2*math.Atan(trailerAspect*math.Tan(rad((top-bottom)*1.5/2)))))
	hfov = clamp(hfov, 90, 120)
	vfov := deg(2 * math.Atan(math.Tan(rad(hfov/2))/trailerAspect))
	// the subject sits in the upper part of the frame, clear of the glass over the card's bottom
	pitch := (top+bottom)/2 - vfov*0.12
	if top-bottom > vfov*0.8 {
		pitch = top - vfov*0.40
	}
	// a VR scene happens in front of the viewer, so the camera only leans toward the people, never turns sideways
	return camera{yaw: clamp((left+right)/2*0.5, -25, 25), pitch: clamp(pitch, -50, -2), hfov: hfov, vfov: vfov}
}

func rad(d float64) float64 { return d * math.Pi / 180 }
func deg(r float64) float64 { return r * 180 / math.Pi }
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
