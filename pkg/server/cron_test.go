package server

import (
	"testing"

	"github.com/robfig/cron/v3"
	"github.com/xbapps/xbvr/pkg/config"
)

func TestFormatCronSchedule(t *testing.T) {
	cases := []struct {
		name string
		in   config.CronSchedule
		want string
	}{
		{"plain interval", config.CronSchedule{HourInterval: 12}, "@every 12h"},
		{"zero interval is clamped", config.CronSchedule{HourInterval: 0}, "@every 1h"},
		{"multi day interval", config.CronSchedule{HourInterval: 72}, "@every 72h"},
		{"same day range", config.CronSchedule{UseRange: true, HourInterval: 2, HourStart: 8, HourEnd: 20, MinuteStart: 15}, "15 8-20/2 * * *"},
		{"hourly range has no step", config.CronSchedule{UseRange: true, HourInterval: 1, HourStart: 8, HourEnd: 20}, "0 8-20 * * *"},
		{"zero interval range does not panic", config.CronSchedule{UseRange: true, HourInterval: 0, HourStart: 22, HourEnd: 3}, "0 0-3,22-23 * * *"},
		{"over midnight aligned", config.CronSchedule{UseRange: true, HourInterval: 3, HourStart: 22, HourEnd: 6}, "0 1-6/3,22-23/3 * * *"},
		{"over midnight interval too big", config.CronSchedule{UseRange: true, HourInterval: 5, HourStart: 22, HourEnd: 2}, "0 22-23/5 * * *"},
		{"end from second day of slider", config.CronSchedule{UseRange: true, HourInterval: 1, HourStart: 20, HourEnd: 26}, "0 0-2,20-23 * * *"},
		{"minute 60 clamped", config.CronSchedule{UseRange: true, HourInterval: 1, HourStart: 1, HourEnd: 2, MinuteStart: 60}, "59 1-2 * * *"},
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	for _, c := range cases {
		got := formatCronSchedule(c.in)
		if got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
		if _, err := parser.Parse(got); err != nil {
			t.Errorf("%s: %q does not parse: %v", c.name, got, err)
		}
	}
}

func TestReloadSchedules(t *testing.T) {
	cronInstance = cron.New()
	config.Config.Cron.RescanSchedule = config.CronSchedule{Enabled: true, HourInterval: 2, HourEnd: 23}
	config.Config.Cron.LibraryHealthSchedule = config.CronSchedule{Enabled: true, HourInterval: 6, HourEnd: 23}
	ReloadSchedules()
	before := len(cronInstance.Entries())
	if _, ok := NextScheduledRuns()["rescanSchedule"]; !ok {
		t.Fatal("rescan not scheduled")
	}

	config.Config.Cron.LibraryHealthSchedule.Enabled = false
	ReloadSchedules()
	if _, ok := NextScheduledRuns()["libraryHealthSchedule"]; ok {
		t.Error("disabled health schedule still registered")
	}
	if got := len(cronInstance.Entries()); got != before-1 {
		t.Errorf("entries after reload = %d, want %d (old entries must be removed)", got, before-1)
	}
}
