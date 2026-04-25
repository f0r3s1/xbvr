package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

// all: prefix is required so files prefixed with '_' (e.g. Vite's
// _plugin-vue_export-helper-*.js) are not silently excluded.
//go:embed all:dist
var Assets embed.FS

func GetFileSystem(useOS bool) http.FileSystem {
	if useOS {
		return http.Dir("ui/dist")
	}

	fs, err := fs.Sub(Assets, "dist")
	if err != nil {
		log.Panic(err)
	}
	return http.FS(fs)
}
