package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/gammazero/nexus/v3/router"
	"github.com/gammazero/nexus/v3/wamp"
	"github.com/go-openapi/spec"
	"github.com/gorilla/mux"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/koding/websocketproxy"
	"github.com/peterbourgon/diskv"
	"github.com/rs/cors"
	"github.com/xbapps/xbvr/pkg/imgcache"
	"willnorris.com/go/imageproxy"

	"github.com/xbapps/xbvr/pkg/api"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/migrations"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
	"github.com/xbapps/xbvr/ui"
)

var (
	wsAddr = common.WsAddr
	log    = &common.Log
)

// requireAuth guards everything once a login is configured. Browsers sign in on /login and carry a session cookie
// (also sent on the websocket), tools use basic auth or the API key, players log in through their own API and get signed media URLs.
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			handleLogin(w, r)
			return
		case "/logout":
			handleLogout(w, r)
			return
		case "/healthz":
			handleHealthz(w, r)
			return
		}
		if isHereSphereBrowser(r) && (r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/ui")) {
			http.Redirect(w, r, "/heresphere", http.StatusTemporaryRedirect)
			return
		}
		if !api.AuthRequired() || r.Method == http.MethodOptions || api.ValidSession(r) || api.ValidAPIKey(r) || isPlayerAPI(r.URL.Path) || api.ValidMediaToken(r) {
			next.ServeHTTP(w, r)
			return
		}
		if user, pass, ok := r.BasicAuth(); ok && api.CredentialsValid(user, pass) {
			api.SetSession(w, r)
			next.ServeHTTP(w, r)
			return
		}
		if wantsPage(r) {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
			return
		}
		// browsers get a plain 401 (the app goes to the login page); tools get the basic auth challenge
		if r.Header.Get("Sec-Fetch-Mode") == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="XBVR"`)
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}

// isHereSphereBrowser: HereSphere's built-in browser ends its User-Agent with "HereSphere/<version>",
// so typing just the server address in HereSphere opens the library instead of the web UI.
func isHereSphereBrowser(r *http.Request) bool {
	return strings.Contains(r.UserAgent(), " HereSphere/")
}

// isPlayerAPI: DeoVR and HereSphere check the login sent in each request body themselves;
// thumbnails go through the image proxy, which players load without credentials.
func isPlayerAPI(path string) bool {
	for _, prefix := range []string{"/deovr", "/heresphere", "/img/", "/imghm/"} {
		if strings.HasPrefix(path, prefix) && !strings.HasPrefix(path, "/heresphere/events/") {
			return true
		}
	}
	return false
}

// logRequests logs each request with its status and duration (XBVR_DEBUG_REQUESTS).
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Infof("%v %s %s %v %q", rec.status, r.Method, r.URL.RequestURI(), time.Since(start).Round(time.Millisecond), r.UserAgent())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func StartServer(version, commit, branch, date string) {
	common.CurrentVersion = version

	// Run GC more aggressively to keep memory usage lower.
	// GOGC=50 means GC triggers when heap grows 50% above live data (default 100%).
	debug.SetGCPercent(50)

	config.LoadConfig()
	common.CopyXbvrData()

	// Remove old locks
	models.RemoveAllLocks()

	migrations.Migrate("0024-drop-actions-old")

	// Run migrations in background
	go func() {
		config.State.Migration.IsRunning = true
		migrations.ProcessCustomSceneRemappingFiles()
		migrations.Migrate("")
		config.CompleteMigration()
	}()

	go tasks.CheckDependencies()
	models.CheckVolumes()

	models.InitSites()

	restful.DefaultContainer.EnableContentEncoding(true)

	// API endpoints
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(func(req *restful.Request, resp *restful.Response) {
		resp.AddHeader("Location", "/ui/")
		resp.WriteHeader(http.StatusFound)
	}))

	restful.Add(ws)
	restful.Add(api.SceneResource{}.WebService())
	restful.Add(api.ActorResource{}.WebService())
	restful.Add(api.TaskResource{}.WebService())
	restful.Add(api.DMSResource{}.WebService())
	restful.Add(api.ConfigResource{}.WebService())
	restful.Add(api.FilesResource{}.WebService())
	restful.Add(api.DeoVRResource{}.WebService())
	restful.Add(api.HeresphereResource{}.WebService())
	restful.Add(api.PlaylistResource{}.WebService())
	restful.Add(api.AkaResource{}.WebService())
	restful.Add(api.TagGroupResource{}.WebService())
	restful.Add(api.ExternalReference{}.WebService())
	restful.Add(api.HealthResource{}.WebService())
	restful.Add(api.InconsistenciesResource{}.WebService())

	restConfig := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(),
		APIPath:     "/api.json",
		PostBuildSwaggerObjectHandler: func(swo *spec.Swagger) {
			e := spec.VendorExtensible{}
			e.AddExtension("x-logo", map[string]interface{}{
				"url": "/ui/icons/xbvr-512.png",
			})

			swo.Info = &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "XBVR API",
					Version: common.CurrentVersion,
				},
				VendorExtensible: e,
			}
			swo.Tags = []spec.Tag{
				{
					TagProps: spec.TagProps{
						Name:        "Config",
						Description: "Endpoints used by options screen",
					},
				},
				{
					TagProps: spec.TagProps{
						Name:        "DeoVR",
						Description: "Endpoints for interfacing with DeoVR player",
					},
				},
				{
					TagProps: spec.TagProps{
						Name:        "HereSphere",
						Description: "Endpoints for interfacing with HereSphere player",
					},
				},
			}
		},
	}
	restful.Add(restfulspec.NewOpenAPIService(restConfig))

	// Static files
	http.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(ui.GetFileSystem(common.EnvConfig.Debug))))

	// Imageproxy
	r := mux.NewRouter()
	imgDisk := newDiskv(common.ImgDir)
	imgStore := imgcache.New(imgDisk, 64<<20)
	p := imageproxy.NewProxy(&imgcache.NoTIFF{Next: NewForceCacheTransport()}, imgStore)
	p.Client.Transport = &imgcache.Passthrough{Next: p.Client.Transport, Cache: imgStore}
	p.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	// imageproxy >= 0.13 defaults to NOT following redirects when embedded (only its CLI
	// defaults the flag to true). Studio CDNs commonly redirect image URLs, and without
	// this the 30x body is handed to the resizer, which fails with "image: unknown format".
	p.FollowRedirects = true
	// If the client request has a cache-control header (such as 'no-cache'), pass them
	// onto the imageproxy so that this can be respected.
	p.PassRequestHeaders = append(p.PassRequestHeaders, "Cache-Control")
	u, _ := url.Parse("http://127.0.0.1:" + strconv.Itoa(config.Config.Server.Port))
	p.DefaultBaseURL = u
	imgFallback := NewImageProxyFallbackHandler(p, diskcache.NewWithDiskv(imgDisk))
	r.PathPrefix("/img/").Handler(ForceShortCacheHandler(http.StripPrefix("/img", imgFallback)))
	hmp := NewHeatmapThumbnailProxy(p, diskCache(common.HeatmapThumbnailDir))
	r.PathPrefix("/imghm/").Handler(http.StripPrefix("/imghm", hmp))
	downloadhandler := DownloadHandler{}
	r.PathPrefix("/download/").Handler(http.StripPrefix("/download/", downloadhandler))
	myfileshandler := MyFilesHandler{}
	r.PathPrefix("/myfiles/").Handler(http.StripPrefix("/myfiles/", myfileshandler))
	r.SkipClean(true)

	r.PathPrefix("/").Handler(http.DefaultServeMux)

	// CORS
	handler := requireAuth(cors.Default().Handler(r))

	// WAMP router
	routerConfig := &router.Config{
		Debug: false,
		RealmConfigs: []*router.RealmConfig{
			{
				URI:           wamp.URI("default"),
				AnonymousAuth: true,
				AllowDisclose: false,
			},
		},
	}

	wampRouter, err := router.NewRouter(routerConfig, log)
	if err != nil {
		log.Fatal(err)
	}
	defer wampRouter.Close()

	// Run websocket server.
	// the raw WAMP port has no auth of its own; with a login set up only the guarded /ws/ proxy may reach it
	if api.AuthRequired() && wsAddr == "0.0.0.0:9998" {
		wsAddr = "127.0.0.1:9998"
	}
	wss := router.NewWebsocketServer(wampRouter)
	wss.AllowOrigins([]string{"*"})
	wsCloser, err := wss.ListenAndServe(wsAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer wsCloser.Close()

	// Proxy websocket
	wsURL, err := url.Parse("ws://" + wsAddr)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/ws/", func(w http.ResponseWriter, req *http.Request) {
		req.Header["Origin"] = nil
		handler := websocketproxy.ProxyHandler(wsURL)
		handler.ServeHTTP(w, req)
	})

	// Attach logrus hook
	wampHook := common.NewWampHook()
	log.AddHook(wampHook)

	log.Infof("XBVR %v (build date %v) starting...", version, date)

	// DMS
	if config.Config.Interfaces.DLNA.Enabled {
		go tasks.StartDMS()
	}

	// DeoVR remote
	go session.DeoRemote()

	// Cron
	SetupCron()

	// List binding addresses
	addrs, _ := net.InterfaceAddrs()
	ips := []string{}
	for _, addr := range addrs {
		ip, _ := addr.(*net.IPNet)
		if ip.IP.To4() != nil {
			ips = append(ips, fmt.Sprintf("http://%v:%v/", ip.IP, config.Config.Server.Port))
		}
	}

	// Prepare state
	tasks.UpdateState()
	config.State.Server.BoundIP = ips
	config.SaveState()

	log.Infof("Web UI available at %s", strings.Join(ips, ", "))
	log.Infof("Web UI Authentication enabled: %v", common.IsUIAuthEnabled())
	log.Infof("Using database: %s", common.DATABASE_URL)

	httpAddr := fmt.Sprintf("%v:%v", config.Config.Server.BindAddress, config.Config.Server.Port)
	if common.EnvConfig.DebugRequests {
		log.Fatal(http.ListenAndServe(httpAddr, logRequests(handler)))
	} else {
		log.Fatal(http.ListenAndServe(httpAddr, handler))
	}
}

func newDiskv(path string) *diskv.Diskv {
	return diskv.New(diskv.Options{
		BasePath:  path,
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
}

func diskCache(path string) *diskcache.Cache {
	return diskcache.NewWithDiskv(newDiskv(path))
}
