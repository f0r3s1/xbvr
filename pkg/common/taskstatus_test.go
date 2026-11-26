package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func entry(msg string, data logrus.Fields) *logrus.Entry {
	return &logrus.Entry{Message: msg, Data: data, Level: logrus.InfoLevel}
}

func TestTaskStatusTracksScrapes(t *testing.T) {
	taskStatus.last = map[string]map[string]interface{}{}
	taskStatus.scrapers = map[string]bool{}

	recordTaskStatus(entry("Scraping vrspy", logrus.Fields{"task": "scrape"}))
	recordTaskStatus(entry("", logrus.Fields{"task": "scraperProgress", "scraperID": "vrspy", "started": true}))
	recordTaskStatus(entry("", logrus.Fields{"task": "scraperProgress", "scraperID": "slr", "started": true}))
	recordTaskStatus(entry("", logrus.Fields{"task": "scraperProgress", "scraperID": "slr", "completed": true}))
	recordTaskStatus(entry("ignored", logrus.Fields{"task": "other"}))

	status := TaskStatus()
	if msg := status["lastScrapeMessage"].(map[string]interface{})["message"]; msg != "Scraping vrspy" {
		t.Fatalf("last scrape message = %v", msg)
	}
	if running := status["runningScrapers"].([]string); len(running) != 1 || running[0] != "vrspy" {
		t.Fatalf("running scrapers = %v, want [vrspy]", running)
	}
	if status["lastRescanMessage"] != nil && len(status["lastRescanMessage"].(map[string]interface{})) != 0 {
		t.Fatalf("rescan message should be empty, got %v", status["lastRescanMessage"])
	}

	recordTaskStatus(entry("DONE", logrus.Fields{"task": "scraperProgress"}))
	if running := TaskStatus()["runningScrapers"].([]string); len(running) != 0 {
		t.Fatalf("running scrapers after DONE = %v", running)
	}
}
