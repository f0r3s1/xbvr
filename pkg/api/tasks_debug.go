package api

import (
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/tasks"
)

var (
	started      = time.Now()
	debugMu      sync.Mutex
	debugSources = map[string]func() any{}
)

// RegisterDebugSource adds a named section to /api/task/debug, for packages this one cannot import.
func RegisterDebugSource(name string, f func() any) {
	debugMu.Lock()
	defer debugMu.Unlock()
	debugSources[name] = f
}

// requireAPIKey locks debug endpoints to the API key, even when no login is configured.
func requireAPIKey(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if !ValidAPIKey(req.Request) {
		resp.WriteErrorString(http.StatusUnauthorized, "API key required (Settings > API & access)")
		return
	}
	chain.ProcessFilter(req, resp)
}

// debug is a snapshot of the server's state for troubleshooting.
func (i TaskResource) debug(req *restful.Request, resp *restful.Response) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	mb := func(b uint64) float64 { return float64(b*10/(1<<20)) / 10 }
	p := config.Config.Library.Preview
	ffmpeg, _ := exec.LookPath("ffmpeg")
	_, gpuErr := os.Stat("/dev/dri/renderD128")
	proxyHost := ""
	if u, err := url.Parse(config.Config.Advanced.ProxyAddress); err == nil {
		proxyHost = u.Host
	}
	out := map[string]any{
		"version": common.CurrentVersion,
		"uptime":  time.Since(started).Round(time.Second).String(),
		"runtime": map[string]any{
			"goroutines": runtime.NumGoroutine(), "cpus": runtime.NumCPU(),
			"heap_mb": mb(m.HeapAlloc), "sys_mb": mb(m.Sys), "gc_runs": m.NumGC,
		},
		"previews": map[string]any{
			"method": p.Method, "codec": p.Codec, "resolution": p.Resolution,
			"ffmpeg": ffmpeg, "gpu": gpuErr == nil,
		},
		"health_check_running": tasks.IsHealthRunning(),
		"image_proxy_host":     proxyHost,
	}
	debugMu.Lock()
	for name, f := range debugSources {
		out[name] = f()
	}
	debugMu.Unlock()
	resp.WriteHeaderAndEntity(http.StatusOK, out)
}
