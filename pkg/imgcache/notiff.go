package imgcache

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
)

var tiffMagic = [][]byte{[]byte("II*\x00"), []byte("MM\x00*")}

// NoTIFF refuses TIFF responses before the proxy decodes them: the unmaintained imaging decoder crashes on crafted TIFF files.
type NoTIFF struct {
	Next http.RoundTripper
}

func (t *NoTIFF) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Next.RoundTrip(req)
	if err != nil || resp.Body == nil {
		return resp, err
	}
	body := bufio.NewReader(resp.Body)
	head, _ := body.Peek(4)
	if isTIFF(head) || strings.Contains(resp.Header.Get("Content-Type"), "tiff") {
		resp.Body.Close()
		return &http.Response{
			StatusCode: http.StatusUnsupportedMediaType,
			Status:     "415 Unsupported Media Type",
			Header:     http.Header{"Content-Type": {"text/plain"}},
			Body:       io.NopCloser(strings.NewReader("TIFF images are not supported")),
			Request:    req,
		}, nil
	}
	resp.Body = struct {
		io.Reader
		io.Closer
	}{body, resp.Body}
	return resp, nil
}

func isTIFF(head []byte) bool {
	for _, m := range tiffMagic {
		if bytes.HasPrefix(head, m) {
			return true
		}
	}
	return false
}
