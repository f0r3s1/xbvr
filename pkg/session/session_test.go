package session

import (
	"testing"
	"time"
)

func TestRemainingPlayback(t *testing.T) {
	sceneLength = 60 * time.Minute
	cases := []struct {
		position time.Duration
		speed    float64
		want     time.Duration
	}{
		{0, 1, 60 * time.Minute},
		{50 * time.Minute, 1, 10 * time.Minute},
		{50 * time.Minute, 2, 5 * time.Minute},
		{70 * time.Minute, 1, 0},
	}
	for _, c := range cases {
		playPosition, playSpeed = c.position, c.speed
		if got := remainingPlayback(); got != c.want {
			t.Errorf("position %v speed %v: got %v want %v", c.position, c.speed, got, c.want)
		}
	}
}
