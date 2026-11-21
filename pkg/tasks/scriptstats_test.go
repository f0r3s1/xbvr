package tasks

import "testing"

func TestComputeScriptStats(t *testing.T) {
	// full strokes, one turning point every 500 ms for 60 s
	var acts []Action
	for i := 0; i <= 120; i++ {
		pos := 0
		if i%2 == 1 {
			pos = 100
		}
		acts = append(acts, Action{At: int64(i * 500), Pos: pos})
	}
	st := computeScriptStats(acts)
	if st.Actions != 121 {
		t.Fatalf("actions = %d", st.Actions)
	}
	if st.AvgSpeed != 200 || st.MaxSpeed != 200 {
		t.Errorf("speed avg %.1f max %.1f, want 200", st.AvgSpeed, st.MaxSpeed)
	}
	// 119 turning points over 60 s = 59.5 strokes per minute
	if st.StrokesPerMin < 59 || st.StrokesPerMin > 60 {
		t.Errorf("strokes/min = %.1f, want ~60", st.StrokesPerMin)
	}
	if st.Depth != 100 {
		t.Errorf("depth = %.1f, want 100", st.Depth)
	}

	// a 20 s pause: average speed is the plain mean of interval speeds (the pause
	// interval counts, slowly), strokes per minute ignore the pause
	paused := append([]Action{}, acts[:61]...)
	for _, a := range acts[61:] {
		paused = append(paused, Action{At: a.At + 20000, Pos: a.Pos})
	}
	got := computeScriptStats(paused)
	// 119 intervals at 200, one of 100 units over 20.5 s (~4.88)
	if got.AvgSpeed != 198.4 {
		t.Errorf("speed with pause = %.1f, want 198.4", got.AvgSpeed)
	}
	if got.StrokesPerMin < 59 || got.StrokesPerMin > 61 {
		t.Errorf("strokes/min with pause = %.1f, want ~60", got.StrokesPerMin)
	}

	// equal timestamps count as 0 speed in the mean
	dup := []Action{{At: 0, Pos: 0}, {At: 0, Pos: 50}, {At: 500, Pos: 100}}
	if got := computeScriptStats(dup); got.AvgSpeed != 50 {
		t.Errorf("speed with duplicate timestamp = %.1f, want 50", got.AvgSpeed)
	}

	if got := computeScriptStats(acts[:1]); got.AvgSpeed != 0 || got.Actions != 1 {
		t.Errorf("single action: %+v", got)
	}
}
