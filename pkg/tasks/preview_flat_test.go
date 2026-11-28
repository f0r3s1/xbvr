package tasks

import (
	"strings"
	"testing"

	"github.com/xbapps/xbvr/pkg/models"
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
			if x > 200 && x < 260 && y > 110 && y < 200 {
				img[i], img[i+1], img[i+2] = 220, 170, 140
			}
		}
	}
	c := aim(img, wideW, wideH)
	if c.yaw <= 10 || c.pitch >= -2 || c.pitch <= -50 || c.hfov < 90 || c.hfov > 120 {
		t.Fatalf("camera not aimed at the body on the right: %+v", c)
	}
	ceiling := make([]byte, wideW*wideH*3)
	for i := 0; i < wideW*30*3; i += 3 {
		ceiling[i], ceiling[i+1], ceiling[i+2] = 220, 170, 140
	}
	if c := aim(ceiling, wideW, wideH); c.pitch > -2 {
		t.Fatalf("camera looked at the ceiling: %+v", c)
	}
	if aim(make([]byte, wideW*wideH*3), wideW, wideH) != defaultCamera {
		t.Fatal("empty frame should keep the default camera")
	}
}

func TestCueMoments(t *testing.T) {
	cues := []models.SceneCuepoint{{TimeStart: 5, Name: "intro"}, {TimeStart: 100, Name: "standing"}, {TimeStart: 110, Name: "standing"}, {TimeStart: 300, Name: "cowgirl"}, {TimeStart: 600, Name: "cowgirl"}, {TimeStart: 900, Name: "missionary"}, {TimeStart: 1200, Name: "doggy"}, {TimeStart: 1790, Name: "end"}}
	got := cueMoments(cues, 10, 1710)
	want := []float64{103, 303, 903, 1203}
	if len(got) != 4 {
		t.Fatalf("got %v", got)
	}
	for k := range want {
		if got[k] != want[k] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if len(cueMoments(nil, 10, 100)) != 0 {
		t.Fatal("no cue points should give no moments")
	}
}

func TestUnknownPreviewMethodFallsBack(t *testing.T) {
	if PreviewMethodValid("nope") || !PreviewMethodValid("classic") || !PreviewMethodValid("flat") {
		t.Fatal("unexpected method list")
	}
}
