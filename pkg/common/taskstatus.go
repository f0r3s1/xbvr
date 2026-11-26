package common

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type taskStatusSnapshot struct {
	mu       sync.Mutex
	last     map[string]map[string]interface{}
	scrapers map[string]bool
}

var taskStatus = taskStatusSnapshot{last: map[string]map[string]interface{}{}, scrapers: map[string]bool{}}

func recordTaskStatus(entry *logrus.Entry) {
	task, _ := entry.Data["task"].(string)
	if task != "scrape" && task != "rescan" && task != "scraperProgress" {
		return
	}
	taskStatus.mu.Lock()
	defer taskStatus.mu.Unlock()
	if task == "scraperProgress" {
		id, _ := entry.Data["scraperID"].(string)
		switch {
		case entry.Message == "DONE":
			taskStatus.scrapers = map[string]bool{}
		case entry.Data["started"] == true:
			taskStatus.scrapers[id] = true
		case entry.Data["completed"] == true:
			delete(taskStatus.scrapers, id)
		}
		return
	}
	taskStatus.last[task] = map[string]interface{}{
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"data":      entry.Data,
		"timestamp": entry.Time.String(),
	}
}

// TaskStatus returns the last scrape and rescan messages and the scrapers still running.
func TaskStatus() map[string]interface{} {
	taskStatus.mu.Lock()
	defer taskStatus.mu.Unlock()
	running := make([]string, 0, len(taskStatus.scrapers))
	for id := range taskStatus.scrapers {
		running = append(running, id)
	}
	return map[string]interface{}{
		"lastScrapeMessage": taskStatus.last["scrape"],
		"lastRescanMessage": taskStatus.last["rescan"],
		"runningScrapers":   running,
	}
}
