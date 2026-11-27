package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/common"
)

const maxLogLines = 5000

var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// log returns the end of xbvr.log as plain text.
func (i TaskResource) log(req *restful.Request, resp *restful.Response) {
	n, err := strconv.Atoi(req.QueryParameter("lines"))
	if err != nil || n <= 0 {
		n = 300
	}
	n = min(n, maxLogLines)
	filter := strings.ToLower(req.QueryParameter("grep"))

	f, err := os.Open(filepath.Join(common.AppDir, "xbvr.log"))
	if err != nil {
		resp.WriteErrorString(http.StatusNotFound, "log file not available")
		return
	}
	defer f.Close()

	// a filtered tail reads further back so it can still find n matching lines
	window := int64(n) * 400
	if filter != "" {
		window *= 20
	}
	if st, err := f.Stat(); err == nil && st.Size() > window {
		_, _ = f.Seek(st.Size()-window, io.SeekStart)
	}
	data, _ := io.ReadAll(f)

	var out []string
	for _, line := range strings.Split(ansiCodes.ReplaceAllString(string(data), ""), "\n") {
		if line != "" && (filter == "" || strings.Contains(strings.ToLower(line), filter)) {
			out = append(out, line)
		}
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	resp.AddHeader("Content-Type", "text/plain; charset=utf-8")
	_, _ = resp.Write([]byte(strings.Join(out, "\n") + "\n"))
}
