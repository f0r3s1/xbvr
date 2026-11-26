package config

import (
	"encoding/json"
	"testing"

	"github.com/creasty/defaults"
)

func TestCronScheduleDefaults(t *testing.T) {
	var c ObjectConfig
	if err := defaults.Set(&c); err != nil {
		t.Fatal(err)
	}
	cr := c.Cron
	check := func(name string, s CronSchedule, enabled bool, interval, delay int) {
		t.Helper()
		if s.Enabled != enabled || s.HourInterval != interval || s.RunAtStartDelay != delay || s.HourEnd != 23 {
			t.Errorf("%s defaults = %+v", name, s)
		}
	}
	check("rescrape", cr.RescrapeSchedule, true, 12, 0)
	check("rescan", cr.RescanSchedule, true, 2, 0)
	check("preview", cr.PreviewSchedule, false, 2, 0)
	check("actor", cr.ActorRescrapeSchedule, false, 12, 0)
	check("stashdb", cr.StashdbRescrapeSchedule, false, 12, 0)
	check("link", cr.LinkScenesSchedule, false, 12, 0)
	check("health", cr.LibraryHealthSchedule, true, 168, 0)

	// a saved config from before the health schedule existed keeps its values and gets the new default
	saved := `{"cron":{"rescanSchedule":{"enabled":false,"hourInterval":5,"useRange":true,"minuteStart":7,"hourStart":22,"hourEnd":3,"runAtStartDelay":4}}}`
	if err := json.Unmarshal([]byte(saved), &c); err != nil {
		t.Fatal(err)
	}
	want := CronSchedule{Enabled: false, HourInterval: 5, UseRange: true, MinuteStart: 7, HourStart: 22, HourEnd: 3, RunAtStartDelay: 4}
	if c.Cron.RescanSchedule != want {
		t.Errorf("loaded rescan = %+v", c.Cron.RescanSchedule)
	}
	check("health after load", c.Cron.LibraryHealthSchedule, true, 168, 0)
}
