package tasks

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
)

func TestWarmCoversSkipsRefusingHost(t *testing.T) {
	var good, bad int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "refusing.example") {
			atomic.AddInt32(&bad, 1)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		atomic.AddInt32(&good, 1)
		w.Write([]byte("img"))
	}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	saved := config.Config.Server.Port
	defer func() { config.Config.Server.Port = saved }()
	config.Config.Server.Port, _ = strconv.Atoi(port)

	var urls []string
	for i := 0; i < 12; i++ {
		urls = append(urls, "https://refusing.example/"+strconv.Itoa(i)+".jpg")
	}
	urls = append(urls, "https://ok.example/a.jpg", "https://ok.example/b.jpg", "not a url", "ftp://x/y.jpg")
	WarmCovers(urls)

	if bad != warmHostFailures {
		t.Errorf("refusing host asked %d times, want %d", bad, warmHostFailures)
	}
	if good != 2 {
		t.Errorf("good covers fetched %d, want 2", good)
	}
}
