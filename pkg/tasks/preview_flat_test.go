package tasks

import (
	"strings"
	"testing"
)

func TestFlatViewProjections(t *testing.T) {
	cases := map[string]string{
		"180_sbs":    "input=hequirect:in_stereo=sbs",
		"360_tb":     "input=equirect:in_stereo=tb",
		"180_mono":   "input=hequirect:in_stereo=2d",
		"fisheye190": "input=fisheye:ih_fov=190:iv_fov=190:in_stereo=sbs",
		"mkx200":     "input=fisheye:ih_fov=200:iv_fov=200:in_stereo=sbs",
		"mkx220":     "input=fisheye:ih_fov=220:iv_fov=220:in_stereo=sbs",
		"flat":       "scale=711:400",
	}
	for projection, want := range cases {
		if got := view(projection, defaultCamera, 711, 400); !strings.Contains(got, want) {
			t.Errorf("%s: %q does not contain %q", projection, got, want)
		}
	}
}

func TestAimFramesSkin(t *testing.T) {
	img := make([]byte, wideW*wideH*3)
	for y := 0; y < wideH; y++ {
		for x := 0; x < wideW; x++ {
			i := (y*wideW + x) * 3
			if x > 200 && x < 260 && y > 60 && y < 160 {
				img[i], img[i+1], img[i+2] = 220, 170, 140
			}
		}
	}
	c := aim(img, wideW, wideH)
	if c.yaw <= 10 || c.pitch <= -8 || c.hfov < 60 || c.hfov > 100 {
		t.Fatalf("camera not aimed at the body on the right: %+v", c)
	}
	if aim(make([]byte, wideW*wideH*3), wideW, wideH) != defaultCamera {
		t.Fatal("empty frame should keep the default camera")
	}
}

func TestUnknownPreviewMethodFallsBack(t *testing.T) {
	if PreviewMethodValid("nope") || !PreviewMethodValid("classic") || !PreviewMethodValid("flat") {
		t.Fatal("unexpected method list")
	}
}
