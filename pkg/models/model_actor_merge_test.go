package models

import (
	"strings"
	"testing"
)

func TestActorNameTokens(t *testing.T) {
	cases := map[string]string{
		"Ava Brooks":      "ava brooks",
		"ava  brooks":     "ava brooks",
		"Avá Brooks":      "ava brooks",
		"AVA BROOKS.":     "ava brooks",
		"Ava-Brooks":      "ava brooks",
		"Zoë Ölander":     "zoe olander",
		"  Ava   Brooks ": "ava brooks",
	}
	for in, want := range cases {
		if got := strings.Join(actorNameTokens(in), " "); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestJaroWinkler(t *testing.T) {
	if s := jaroWinkler("ava brooks", "ava brooks"); s != 1 {
		t.Errorf("identical = %f", s)
	}
	if s := jaroWinkler("ava brooks", "ava brookes"); s < 0.93 {
		t.Errorf("one extra letter = %f, want >= 0.93", s)
	}
	if s := jaroWinkler("ava brooks", "mia stone"); s >= 0.8 {
		t.Errorf("different names = %f, want low", s)
	}
}

func TestUnionJSONList(t *testing.T) {
	got := unionJSONList(`["A. Brooks","Ava B"]`, `["ava b","Avy"]`, "Ava  Brooks", "Ava Brooks")
	if got != `["A. Brooks","Ava B","Avy","Ava  Brooks"]` {
		t.Errorf("union = %s", got)
	}
	// the kept name itself is never added as an alias
	if got := unionJSONList("", "", "Ava Brooks", "ava brooks"); got != "" {
		t.Errorf("self alias = %q", got)
	}
}

func TestPreferredActor(t *testing.T) {
	a := ActorBrief{ID: 1, Count: 10}
	b := ActorBrief{ID: 2, Count: 3, Stash: true}
	if preferredActor(a, b) != 2 {
		t.Error("StashDB-linked actor should be kept")
	}
	b.Stash = false
	if preferredActor(a, b) != 1 {
		t.Error("actor with more scenes should be kept")
	}
	b.Count = 10
	b.filled = 3
	if preferredActor(a, b) != 2 {
		t.Error("more complete profile should be kept")
	}
	b.filled = 0
	if preferredActor(a, b) != 1 {
		t.Error("older record should be kept on a tie")
	}
}
