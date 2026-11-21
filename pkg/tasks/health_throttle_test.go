package tasks

import (
	"sync/atomic"
	"testing"
	"time"
)

// The background health check must stay light: busy work interleaved with
// backgroundYield() may only use a small share of one core.
func TestBackgroundYieldCapsCPU(t *testing.T) {
	atomic.StoreInt32(&healthBackground, 1)
	bgSliceStart = time.Time{}
	defer atomic.StoreInt32(&healthBackground, 0)

	start := time.Now()
	var busy time.Duration
	for time.Since(start) < 1500*time.Millisecond {
		w := time.Now()
		for time.Since(w) < time.Millisecond { // ~1 ms of pure CPU work
		}
		busy += time.Since(w)
		backgroundYield()
	}
	share := float64(busy) / float64(time.Since(start))
	if share > 0.25 {
		t.Fatalf("background work used %.0f%% of the time, want <= 25%%", share*100)
	}
	t.Logf("background work share: %.0f%%", share*100)

	// manual checks are not slowed down
	atomic.StoreInt32(&healthBackground, 0)
	w := time.Now()
	for i := 0; i < 1000; i++ {
		backgroundYield()
	}
	if time.Since(w) > 50*time.Millisecond {
		t.Fatal("backgroundYield slowed down a manual check")
	}
}
