package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
)

func enableTestAuth(t *testing.T) {
	secretOnce.Do(func() { secret = []byte("0123456789abcdef0123456789abcdef") })
	saved := config.Config.Interfaces.DeoVR
	t.Cleanup(func() { config.Config.Interfaces.DeoVR = saved })
	config.Config.Interfaces.DeoVR.AuthEnabled = true
	config.Config.Interfaces.DeoVR.Username = "deo"
	config.Config.Interfaces.DeoVR.Password = "hash"
}

func TestMediaTokenIsBoundToItsMedia(t *testing.T) {
	enableTestAuth(t)
	signed := withMediaToken("http://host:9999/api/dms/file/12?dnt=true")
	if !strings.Contains(signed, "?dnt=true&t=") {
		t.Fatalf("token not appended: %s", signed)
	}
	if !ValidMediaToken(httptest.NewRequest("GET", signed, nil)) {
		t.Fatal("signed url rejected")
	}
	token := signed[strings.Index(signed, "t=")+2:]
	for _, other := range []string{"/api/dms/file/13?t=", "/api/dms/preview/12?t=", "/heresphere/events/12?t="} {
		if ValidMediaToken(httptest.NewRequest("GET", other+token, nil)) {
			t.Fatalf("token for file 12 opened %s", other)
		}
	}
	if ValidMediaToken(httptest.NewRequest("GET", "/api/dms/file/12", nil)) {
		t.Fatal("request without token accepted")
	}
	named := withMediaToken("http://h/api/dms/file/7/7 - Title.funscript")
	if !ValidMediaToken(httptest.NewRequest("GET", strings.ReplaceAll(named, " ", "%20"), nil)) {
		t.Fatal("named script url rejected")
	}
}

func TestMediaTokenOnlyWhenAuthIsOn(t *testing.T) {
	enableTestAuth(t)
	config.Config.Interfaces.DeoVR.AuthEnabled = false
	if got := withMediaToken("http://h/api/dms/file/1"); got != "http://h/api/dms/file/1" {
		t.Fatalf("url signed without auth: %s", got)
	}
}

func TestSessionChangesWithCredentials(t *testing.T) {
	enableTestAuth(t)
	before := sessionValue()
	config.Config.Interfaces.DeoVR.Password = "other-hash"
	if sessionValue() == before {
		t.Fatal("session survives a password change")
	}
}

func TestAPIKey(t *testing.T) {
	apiKeyMu.Lock()
	apiKeyHash, apiKeyLoaded = hashAPIKey("xbvr_test"), true
	apiKeyMu.Unlock()
	t.Cleanup(func() { apiKeyHash, apiKeyLoaded = "", false })

	for header, want := range map[string]bool{"X-API-Key: xbvr_test": true, "Authorization: Bearer xbvr_test": true, "X-API-Key: wrong": false, "": false} {
		r := httptest.NewRequest("GET", "/api/scene/list", nil)
		if name, value, ok := strings.Cut(header, ": "); ok {
			r.Header.Set(name, value)
		}
		if got := ValidAPIKey(r); got != want {
			t.Errorf("%q: got %v want %v", header, got, want)
		}
	}
}

func TestHeresphereToken(t *testing.T) {
	enableTestAuth(t)
	r := httptest.NewRequest("POST", "/heresphere", nil)
	if validHeresphereToken(r) {
		t.Fatal("request without token accepted")
	}
	r.Header.Set("auth-token", heresphereToken())
	if !validHeresphereToken(r) {
		t.Fatal("issued token rejected")
	}
	r.Header.Set("auth-token", "forged")
	if validHeresphereToken(r) {
		t.Fatal("forged token accepted")
	}
}
