package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

const (
	mediaTokenParam   = "t"
	sessionCookieName = "xbvr_session"
	authSecretKey     = "auth_secret"
)

// AuthRequired is true when either the web UI login (XBVR_UI_USERNAME/PASSWORD) or the player login is set up.
func AuthRequired() bool {
	return common.IsUIAuthEnabled() || isDeoAuthEnabled()
}

// CredentialsValid accepts the web UI login and the player login, so one password works everywhere.
func CredentialsValid(user, password string) bool {
	if common.IsUIAuthEnabled() && common.CheckUICredentials(user, password) {
		return true
	}
	return isDeoAuthEnabled() && deoCredentialsValid(user, password)
}

var (
	secretOnce sync.Once
	secret     []byte
)

func authSecret() []byte {
	secretOnce.Do(func() {
		var kv models.KV
		db, _ := models.GetCommonDB()
		if db.Where(&models.KV{Key: authSecretKey}).First(&kv).Error == nil {
			if b, err := hex.DecodeString(kv.Value); err == nil && len(b) == 32 {
				secret = b
				return
			}
		}
		secret = make([]byte, 32)
		_, _ = rand.Read(secret)
		kv = models.KV{Key: authSecretKey, Value: hex.EncodeToString(secret)}
		kv.Save()
	})
	return secret
}

func sign(subject string) string {
	m := hmac.New(sha256.New, authSecret())
	m.Write([]byte(subject))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil)[:16])
}

// mediaSubjectRe finds what a player URL grants access to: one file, preview, heatmap or event stream.
var mediaSubjectRe = regexp.MustCompile(`/(api/dms/(?:file|preview|heatmap)|heresphere/events)/([^/?#]+)`)

func mediaSubject(path string) string {
	m := mediaSubjectRe.FindStringSubmatch(path)
	if m == nil {
		return ""
	}
	return m[1] + ":" + m[2]
}

// withMediaToken signs a media URL handed to a player, which cannot send credentials when it fetches it.
func withMediaToken(rawURL string) string {
	if !AuthRequired() {
		return rawURL
	}
	subject := mediaSubject(rawURL)
	if subject == "" {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + mediaTokenParam + "=" + sign(subject)
}

// ValidMediaToken reports whether the request carries the token for exactly the media it asks for.
func ValidMediaToken(r *http.Request) bool {
	subject := mediaSubject(r.URL.Path)
	token := r.URL.Query().Get(mediaTokenParam)
	return subject != "" && token != "" && hmac.Equal([]byte(token), []byte(sign(subject)))
}

// sessionValue changes with the credentials, so changing a password signs every browser out.
func sessionValue() string {
	cfg := config.Config.Interfaces.DeoVR
	return sign("session\x00" + common.EnvConfig.UIUsername + "\x00" + common.EnvConfig.UIPassword + "\x00" + cfg.Username + "\x00" + cfg.Password)
}

func ValidSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(sessionValue())) == 1
}

func ClearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

func SetSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionValue(),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 3600,
	})
}

const apiKeyHashKey = "api_key_hash"

var (
	apiKeyMu     sync.Mutex
	apiKeyLoaded bool
	apiKeyHash   string
)

func storedAPIKeyHash() string {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	if !apiKeyLoaded {
		var kv models.KV
		db, _ := models.GetCommonDB()
		if db.Where(&models.KV{Key: apiKeyHashKey}).First(&kv).Error == nil {
			apiKeyHash = kv.Value
		}
		apiKeyLoaded = true
	}
	return apiKeyHash
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// ValidAPIKey accepts "X-API-Key: <key>" or "Authorization: Bearer <key>"; only a hash of the key is stored.
func ValidAPIKey(r *http.Request) bool {
	key := r.Header.Get("X-API-Key")
	if key == "" {
		key, _ = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	stored := storedAPIKeyHash()
	return key != "" && stored != "" && subtle.ConstantTimeCompare([]byte(hashAPIKey(key)), []byte(stored)) == 1
}

// newAPIKey replaces any previous key; the plain key is returned once and never stored.
func newAPIKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	key := "xbvr_" + base64.RawURLEncoding.EncodeToString(b)
	setAPIKeyHash(hashAPIKey(key))
	return key
}

func setAPIKeyHash(hash string) {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	db, _ := models.GetCommonDB()
	if hash == "" {
		db.Where(&models.KV{Key: apiKeyHashKey}).Delete(&models.KV{})
	} else {
		kv := models.KV{Key: apiKeyHashKey, Value: hash}
		kv.Save()
	}
	apiKeyHash, apiKeyLoaded = hash, true
}
