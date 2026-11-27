package tasks

import (
	"fmt"
	"image/color"
	"math"
	"sort"
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
	for k, at := range starts {
		cams[k] = defaultCamera
		if input == "" {
			continue
		}
		if img, err := grabWide(inputFile, input, stereo, at+flatMomentSecs/2); err == nil {
			cams[k] = aim(img, wideW, wideH)
		}
	}
	return renderTrailer(inputFile, destFile, projection, o, starts, cams)
}

// grabWide returns one wide, undistorted RGB frame of one eye.
func grabWide(inputFile, input, stereo string, at float64) ([]byte, error) {
	vf := fmt.Sprintf("v360=input=%s:in_stereo=%s:output=flat:out_stereo=2d:h_fov=%.0f:v_fov=%.0f:w=%d:h=%d,format=rgb24", input, stereo, wideHFov, wideVFov, wideW, wideH)
	out, err := buildCmd(GetBinPath("ffmpeg"), "-v", "error", "-ss", fmt.Sprintf("%.2f", at), "-i", inputFile, "-frames:v", "1", "-vf", vf, "-f", "rawvideo", "pipe:1").Output()
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
	var xs, ys []int
	for y := 0; y < h; y++ {
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

	tanH, tanV := math.Tan(rad(wideHFov/2)), math.Tan(rad(wideVFov/2))
	yawAt := func(x float64) float64 { return deg(math.Atan((x/float64(w)*2 - 1) * tanH)) }
	pitchAt := func(y float64) float64 { return deg(math.Atan((1 - y/float64(h)*2) * tanV)) }
	left, right := yawAt(pick(xs, 0.05)), yawAt(pick(xs, 0.95))
	top, bottom := pitchAt(pick(ys, 0.03)), pitchAt(pick(ys, 0.97))

	hfov := math.Max((right-left)*1.2, deg(2*math.Atan(16.0/9*math.Tan(rad((top-bottom)*1.15/2)))))
	hfov = clamp(hfov, 60, 100)
	vfov := deg(2 * math.Atan(math.Tan(rad(hfov/2))*9/16))
	pitch := (top + bottom) / 2
	if top-bottom > vfov*0.9 {
		pitch = top - vfov*0.45
	}
	return camera{yaw: clamp((left+right)/2, -60, 60), pitch: clamp(pitch, -50, 15), hfov: hfov, vfov: vfov}
}

func rad(d float64) float64 { return d * math.Pi / 180 }
func deg(r float64) float64 { return r * 180 / math.Pi }
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
