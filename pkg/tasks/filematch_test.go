package tasks

import (
	"reflect"
	"testing"
)

func TestMatchKeyIgnoresEncodeTagsAndVariants(t *testing.T) {
	base := matchKey("LumenWorks_Velvet_Harbor_Part_3_0002_8K_180x180_3dh.mp4")
	if base == "" {
		t.Fatal("expected a key for a normal scene file name")
	}
	same := []string{
		"LumenWorks_Velvet_Harbor_Part_3_0002_4K_180x180_3dh_ai.mp4",     // other resolution + _ai
		"LumenWorks Velvet Harbor Part 3 0002 8K 180x180 3dh.mkv",        // spaces, other container
		"LumenWorks_Velvet_Velvet_Harbor_Part_3_0002_8K_180x180_3dh.mp4", // repeated word
		"lumenworks-velvet-harbor-part-3-0002-6k-h265-60fps-LR.mp4",      // codec/fps/projection noise
		"LumenWorks_Velvet_Harbor_Part_3_0002_8K_180x180_3dh_ai.funscript",
	}
	for _, n := range same {
		if k := matchKey(n); k != base {
			t.Errorf("matchKey(%q) = %q, want %q", n, k, base)
		}
	}
	if k := matchKey("LumenWorks_Velvet_Harbor_Part_4_0003_8K.mp4"); k == base {
		t.Errorf("a different scene must not share the key %q", base)
	}
}

func TestMatchKeyRejectsTooShortNames(t *testing.T) {
	for _, n := range []string{"GOPR0042.mp4", "clip_8K_180x180.mp4", "a_b.mp4"} {
		if k := matchKey(n); k != "" {
			t.Errorf("matchKey(%q) = %q, want empty (too ambiguous)", n, k)
		}
	}
}

func TestSiblingPrefix(t *testing.T) {
	video := matchTokens("Golden Coast Sunset 8K.mkv")
	for _, s := range []string{
		"Golden Coast Sunset 8K.mkv.funscript",
		"Golden_Coast_Sunset - Script v2.funscript",
		"Golden Coast Sunset_AI.funscript",
		"Golden.Coast.Sunset.8K.hsp",
	} {
		if !hasTokenPrefix(matchTokens(s), video) {
			t.Errorf("%q should be recognised as belonging to the video", s)
		}
	}
	if hasTokenPrefix(matchTokens("Golden Coast Sunrise.funscript"), video) {
		t.Error("a different title must not match")
	}
	if got := matchTokens("Golden_Golden_Coast_8K_180x180_3dh_ai.mp4"); !reflect.DeepEqual(got, []string{"golden", "coast"}) {
		t.Errorf("matchTokens = %v", got)
	}
}
