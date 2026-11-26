package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/xbapps/xbvr/pkg/api"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
)

var cronInstance *cron.Cron

// scheduledTask is one user configurable task schedule (Options > Task Schedules).
type scheduledTask struct {
	key      string // json key of the schedule in config.Cron, also used by the UI
	name     string
	schedule func() *config.CronSchedule
	run      func()
	// idleOnly skips the run while someone is playing a video (session active)
	idleOnly bool

	mu      sync.Mutex // held while the task runs, so runs never overlap
	entryID cron.EntryID
}

var scheduledTasks = []*scheduledTask{
	{key: "rescrapeSchedule", name: "Rescrape", idleOnly: true,
		schedule: func() *config.CronSchedule { return &config.Config.Cron.RescrapeSchedule },
		run:      func() { tasks.Scrape("_enabled", "", "", false) }},
	{key: "rescanSchedule", name: "Rescan", idleOnly: true,
		schedule: func() *config.CronSchedule { return &config.Config.Cron.RescanSchedule },
		run:      func() { tasks.RescanVolumes(-1) }},
	{key: "previewSchedule", name: "Preview Generation", idleOnly: true,
		schedule: func() *config.CronSchedule { return &config.Config.Cron.PreviewSchedule },
		run:      generatePreviews},
	{key: "actorRescrapeSchedule", name: "Actor Rescrape", idleOnly: true,
		schedule: func() *config.CronSchedule { return &config.Config.Cron.ActorRescrapeSchedule },
		run:      tasks.ScrapeActors},
	{key: "stashdbRescrapeSchedule", name: "Stashdb Rescrape", idleOnly: true,
		schedule: func() *config.CronSchedule { return &config.Config.Cron.StashdbRescrapeSchedule },
		run:      api.StashdbRunAll},
	{key: "linkScenesSchedule", name: "Link Scenes", idleOnly: true,
		schedule: func() *config.CronSchedule { return &config.Config.Cron.LinkScenesSchedule },
		run:      tasks.MatchAlternateSources},
	// Library Health already backs off on its own (skips while other jobs hold a lock, duty cycle)
	{key: "libraryHealthSchedule", name: "Library Health",
		schedule: func() *config.CronSchedule { return &config.Config.Cron.LibraryHealthSchedule },
		run:      tasks.BackgroundHealthCheck},
}

func SetupCron() {
	cronInstance = cron.New()
	cronInstance.AddFunc("@every 2s", session.CheckForDeadSession)
	cronInstance.AddFunc("@every 6h", tasks.CalculateCacheSizes)
	cronInstance.Start()
	go tasks.CalculateCacheSizes()

	api.ReloadTaskSchedules = ReloadSchedules
	api.TaskScheduleNextRuns = NextScheduledRuns
	ReloadSchedules()

	// run-at-startup delays apply once per process start, not on every settings save
	for _, t := range scheduledTasks {
		s := t.schedule()
		if s.Enabled && s.RunAtStartDelay > 0 {
			time.AfterFunc(time.Duration(s.RunAtStartDelay)*time.Minute, t.execute)
		}
	}
}

// ReloadSchedules (re)registers all task schedules from the current config, so saved settings apply without a restart.
var schedulesMu sync.Mutex

func ReloadSchedules() {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	for _, t := range scheduledTasks {
		if t.entryID != 0 {
			cronInstance.Remove(t.entryID)
			t.entryID = 0
		}
		s := t.schedule()
		if !s.Enabled {
			continue
		}
		spec := formatCronSchedule(*s)
		id, err := cronInstance.AddFunc(spec, t.execute)
		if err != nil {
			log.Errorf("Invalid schedule %q for %s task: %v", spec, t.name, err)
			continue
		}
		t.entryID = id
		log.Infof("Scheduled %s task (%s), next run at %v", t.name, spec, cronInstance.Entry(id).Next)
	}
}

// NextScheduledRuns returns the next run time of each enabled schedule, by config key.
func NextScheduledRuns() map[string]time.Time {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	out := map[string]time.Time{}
	for _, t := range scheduledTasks {
		if t.entryID != 0 {
			out[t.key] = cronInstance.Entry(t.entryID).Next
		}
	}
	return out
}

func (t *scheduledTask) execute() {
	if t.idleOnly && session.HasActiveSession() {
		log.Infof("Skipping %s task, a video is playing", t.name)
		return
	}
	if !t.mu.TryLock() {
		log.Infof("Skipping %s task, previous run still in progress", t.name)
		return
	}
	defer t.mu.Unlock()
	t.run()
}

func generatePreviews() {
	s := config.Config.Cron.PreviewSchedule
	if !s.UseRange {
		tasks.GeneratePreviews(nil)
		return
	}
	endTime := calcEndTime(s.HourStart, s.HourEnd, s.MinuteStart)
	log.Infof("Preview Generation will stop at %v", endTime)
	tasks.GeneratePreviews(&endTime)
}

// formatCronSchedule turns a schedule into a robfig/cron spec;
func formatCronSchedule(schedule config.CronSchedule) string {
	interval := schedule.HourInterval
	if interval < 1 {
		interval = 1
	}
	if !schedule.UseRange {
		return fmt.Sprintf("@every %vh", interval)
	}

	start := clamp(schedule.HourStart, 0, 23)
	end := clamp(schedule.HourEnd, 0, 47) % 24 // the UI slider spans two days
	minute := clamp(schedule.MinuteStart, 0, 59)
	step := ""
	if interval > 1 {
		step = fmt.Sprintf("/%v", interval)
	}

	hours := ""
	if start > end {
		afterMidnightStart := (interval - ((24 - start) % interval)) % interval
		if afterMidnightStart <= end {
			hours = fmt.Sprintf("%v-%v%v,%v-23%v", afterMidnightStart, end, step, start, step)
		} else {
			// the interval is too big to fit a run after midnight before the window ends
			hours = fmt.Sprintf("%v-23%v", start, step)
		}
	} else {
		hours = fmt.Sprintf("%v-%v%v", start, end, step)
	}
	return fmt.Sprintf("%v %v * * *", minute, hours)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// calcEndTime is when a ranged task should stop starting new work (end of today's window).
func calcEndTime(startHour int, endHour int, minuteStart int) time.Time {
	endHour %= 24
	dt := time.Now()
	if startHour > endHour && dt.Hour() > endHour {
		next := dt.AddDate(0, 0, 1)
		return time.Date(next.Year(), next.Month(), next.Day(), endHour, minuteStart, 0, 0, dt.Location())
	}
	return time.Date(dt.Year(), dt.Month(), dt.Day(), endHour, minuteStart, 0, 0, dt.Location())
}
