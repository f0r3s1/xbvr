package server

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/api"
)

var loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>XBVR · Sign in</title>
<style>
  :root { --bg: #f5f5f7; --card: #fff; --text: #1d1d1f; --muted: #6e6e73; --line: #d2d2d7; --accent: #7957d5; --error: #d93f5a; }
  @media (prefers-color-scheme: dark) { :root { --bg: #111114; --card: #1c1c21; --text: #f2f2f5; --muted: #9a9aa3; --line: #34343c; --error: #ff6b81; } }
  * { box-sizing: border-box; }
  body { margin: 0; min-height: 100vh; display: grid; place-items: center; background: var(--bg); color: var(--text);
         font: 15px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 16px; }
  form { width: 100%; max-width: 340px; background: var(--card); border: 1px solid var(--line); border-radius: 14px; padding: 28px; }
  h1 { margin: 0 0 22px; font-size: 26px; letter-spacing: 0.02em; text-align: center; }
  label { display: block; font-size: 13px; color: var(--muted); margin: 14px 0 6px; }
  input { width: 100%; padding: 10px 12px; font: inherit; color: var(--text); background: transparent;
          border: 1px solid var(--line); border-radius: 8px; }
  input:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 25%, transparent); }
  button { width: 100%; margin-top: 22px; padding: 11px; font: inherit; font-weight: 600; color: #fff;
           background: var(--accent); border: 0; border-radius: 8px; cursor: pointer; }
  button:hover { filter: brightness(1.08); }
  .error { margin: 0 0 4px; padding: 9px 12px; border-radius: 8px; color: var(--error);
           background: color-mix(in srgb, var(--error) 12%, transparent); font-size: 14px; }
</style>
</head>
<body>
<form method="post" action="/login">
  <h1>XBVR</h1>
  {{if .Failed}}<p class="error" role="alert">Wrong username or password</p>{{end}}
  <input type="hidden" name="next" value="{{.Next}}">
  <label for="username">Username</label>
  <input id="username" name="username" autocomplete="username" autofocus required>
  <label for="password">Password</label>
  <input id="password" name="password" type="password" autocomplete="current-password" required>
  <button type="submit">Sign in</button>
</form>
</body>
</html>
`))

// safeNext only returns to a path on this server, so the login cannot be used to redirect elsewhere.
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") || strings.HasPrefix(next, "/login") {
		return "/ui/"
	}
	return next
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	next := safeNext(r.FormValue("next"))
	if !api.AuthRequired() || api.ValidSession(r) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	failed := false
	if r.Method == http.MethodPost {
		if api.CredentialsValid(r.FormValue("username"), r.FormValue("password")) {
			api.SetSession(w, r)
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
		time.Sleep(time.Second) // slows down password guessing
		failed = true
		w.WriteHeader(http.StatusUnauthorized)
	}
	_ = loginPage.Execute(w, struct {
		Next   string
		Failed bool
	}{next, failed})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	api.ClearSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// wantsPage tells a browser opening a page (sent to the login form) from API calls and tools (answered with 401).
func wantsPage(r *http.Request) bool {
	return r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html") && !strings.HasPrefix(r.URL.Path, "/api/")
}
