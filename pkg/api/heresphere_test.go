package api

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func heresphereTestRequest(body string) *restful.Request {
	return restful.NewRequest(httptest.NewRequest("POST", "/heresphere/1", bytes.NewBufferString(body)))
}

func TestHeresphereRequestDataIsPerRequest(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan string, 200)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := heresphereTestRequest(fmt.Sprintf(`{"username":"user%d","rating":%d}`, i, i%5))
			heresphereRequestBody(req)
			got := heresphereRequestData(req)
			if got.Username != fmt.Sprintf("user%d", i) || got.Rating == nil || int(*got.Rating) != i%5 {
				errs <- fmt.Sprintf("request %d read %+v", i, got)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}

func TestHeresphereWantsDeleteOnlyWhenTrue(t *testing.T) {
	saved := config.Config.Interfaces.Heresphere.AllowFileDeletes
	defer func() { config.Config.Interfaces.Heresphere.AllowFileDeletes = saved }()
	config.Config.Interfaces.Heresphere.AllowFileDeletes = true

	yes, no := true, false
	cases := []struct {
		name string
		req  HereSphereAuthRequest
		want bool
	}{
		{"absent", HereSphereAuthRequest{}, false},
		{"false", HereSphereAuthRequest{DeleteFiles: &no}, false},
		{"true", HereSphereAuthRequest{DeleteFiles: &yes}, true},
	}
	for _, c := range cases {
		if got := heresphereWantsDelete(c.req); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}

	config.Config.Interfaces.Heresphere.AllowFileDeletes = false
	if heresphereWantsDelete(HereSphereAuthRequest{DeleteFiles: &yes}) {
		t.Error("deletes must stay off when the option is disabled")
	}
}

func TestFindTheMainTrackWithoutTracks(t *testing.T) {
	one := 1
	tags := []HeresphereTag{
		{Name: "Studio:Example"},
		{Name: "Talent:Ava", StartMilliseconds: 1000, EndMilliseconds: 5000},
		{Name: "kiss", EndMilliseconds: 9000},
		{Name: "Category:beach", EndMilliseconds: 9000, Track: &one},
	}
	if got := findTheMainTrack(HereSphereAuthRequest{Tags: &tags}); got != 0 {
		t.Fatalf("got track %d, want 0", got)
	}

	alternateOnly := []HeresphereTag{{Name: "kiss", EndMilliseconds: 9000, Track: &one}}
	if got := findTheMainTrack(HereSphereAuthRequest{Tags: &alternateOnly}); got != 1 {
		t.Fatalf("alternate track: got %d, want 1", got)
	}
}

func TestHspWriteBackSkipsSceneWithoutVideo(t *testing.T) {
	saved := config.Config.Interfaces.Heresphere.AllowHspData
	defer func() { config.Config.Interfaces.Heresphere.AllowHspData = saved }()
	config.Config.Interfaces.Heresphere.AllowHspData = true

	hsp := "aGVsbG8="
	ProcessHeresphereUpdates(&models.Scene{}, HereSphereAuthRequest{Hsp: &hsp}, models.File{})

	dir := t.TempDir()
	bad := "not base64!"
	ProcessHeresphereUpdates(&models.Scene{}, HereSphereAuthRequest{Hsp: &bad}, models.File{ID: 1, Path: dir, Filename: "video.mp4"})
	if _, err := os.Stat(filepath.Join(dir, "video.hsp")); !os.IsNotExist(err) {
		t.Fatal("an hsp file was written from undecodable data")
	}
}

func TestDeoProjectionWithoutVideoFiles(t *testing.T) {
	if s, st, a := deoProjection(nil); s != "" || st != "" || a {
		t.Fatalf("got %q %q %v for a scene without files", s, st, a)
	}
	s, st, _ := deoProjection([]models.File{{VideoProjection: "360_tb"}})
	if s != "tb" || st != "sphere" {
		t.Fatalf("360_tb mapped to %q %q", s, st)
	}
}

func TestDeoCredentialsCache(t *testing.T) {
	saved := config.Config.Interfaces.DeoVR
	defer func() { config.Config.Interfaces.DeoVR = saved }()

	hash := func(p string) string {
		h, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.MinCost)
		return string(h)
	}
	config.Config.Interfaces.DeoVR.Username = "deo"
	config.Config.Interfaces.DeoVR.Password = hash("first")

	if !deoCredentialsValid("deo", "first") || !deoCredentialsValid("deo", "first") {
		t.Fatal("valid credentials rejected")
	}
	if deoCredentialsValid("deo", "wrong") || deoCredentialsValid("other", "first") {
		t.Fatal("invalid credentials accepted")
	}

	config.Config.Interfaces.DeoVR.Password = hash("second")
	if deoCredentialsValid("deo", "first") {
		t.Fatal("old password still accepted after a password change")
	}
	if !deoCredentialsValid("deo", "second") {
		t.Fatal("new password rejected")
	}
}

func TestHeresphereTagPlayed(t *testing.T) {
	if got := heresphereTagPlayed(3).Name; got != "Played:3" {
		t.Fatalf("got %q", got)
	}
}
