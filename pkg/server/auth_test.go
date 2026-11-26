package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
)

func TestRequireAuthOpenWithoutLogin(t *testing.T) {
	saved := config.Config.Interfaces.DeoVR
	defer func() { config.Config.Interfaces.DeoVR = saved }()
	config.Config.Interfaces.DeoVR.AuthEnabled = false

	h := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/scene/list", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d without any login configured", rec.Code)
	}
}

func TestIsPlayerAPI(t *testing.T) {
	for path, want := range map[string]bool{
		"/deovr":               true,
		"/heresphere/12":       true,
		"/heresphere/events/1": false,
		"/api/dms/file/1":      false,
		"/ws/":                 false,
		"/img/700x/http://x":   true,
	} {
		if got := isPlayerAPI(path); got != want {
			t.Errorf("%s: got %v want %v", path, got, want)
		}
	}
}

func TestHereSphereBrowserRedirect(t *testing.T) {
	h := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ua := "Mozilla/5.0 (X11; Linux x86_64; Quest 3) AppleWebKit/537.36 Chrome/134.0 VR Safari/537.36 HereSphere/0.11.6 Quest2"
	for path, want := range map[string]int{"/": http.StatusTemporaryRedirect, "/ui/": http.StatusTemporaryRedirect, "/api/scene/list": http.StatusOK} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("User-Agent", ua)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("%s: got %d want %d", path, rec.Code, want)
		}
	}
}
