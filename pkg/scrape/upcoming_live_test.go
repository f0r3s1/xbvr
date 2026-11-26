//go:build scrapelive

// Live check of the upcoming-scene sources (see live_test.go for how to run live tests):
//
//	go test -tags='json1 scrapelive' -vet=off ./pkg/scrape/ -run TestLiveUpcoming -v
//
// Sites can legitimately have nothing announced, so an empty list only logs.

package scrape

import (
	"encoding/json"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestLiveUpcomingVRBangers(t *testing.T) {
	content := "https://content.vrbangers.com"
	builder, err := fetchURLWithFlareSolverr(content+"/api/content/v1/home-page-builder", "vrbangers.com")
	if err != nil {
		t.Fatal(err)
	}
	scenes := parseVRBangersUpcoming(builder, "vrbangers", "VRBangers", "https://vrbangers.com/", content)
	if len(scenes) == 0 {
		t.Log("no upcoming scenes announced")
	}
	for _, sc := range scenes {
		if sc.Title == "" || len(sc.Covers) == 0 || !sc.Upcoming {
			t.Errorf("incomplete upcoming scene: %+v", sc)
		}
	}
	t.Logf("%d upcoming scenes", len(scenes))
}

func TestLiveUpcomingVRSpy(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"query": vrspyUpcomingQuery})
	resp, err := resty.New().R().SetHeader("User-Agent", UserAgent).SetHeader("Content-Type", "application/json").
		SetBody(body).Post(baseURL + "/api/graphql")
	if err != nil || resp.StatusCode() != 200 {
		t.Fatalf("graphql: %v %v", err, resp.StatusCode())
	}
	scenes := parseVRSpyUpcoming(resp.String())
	if len(scenes) == 0 {
		t.Log("no upcoming scenes announced: " + resp.String())
	}
	for _, sc := range scenes {
		if sc.SiteID == "" || sc.Title == "" || sc.Released == "" || !sc.Upcoming {
			t.Errorf("incomplete upcoming scene: %+v", sc)
		}
	}
	t.Logf("%d upcoming scenes", len(scenes))
}
