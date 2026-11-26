package imgcache

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

const evictBatch = 100

type cacheFile struct {
	path    string
	size    int64
	mod     time.Time
	variant bool
}

// Evict deletes files under dir until the total size is at most maxBytes.
func Evict(dir string, maxBytes int64, pause time.Duration, evictLegacy bool) (removed int64) {
	var files []cacheFile
	var total int64
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		files = append(files, cacheFile{p, fi.Size(), fi.ModTime(), strings.HasSuffix(p, VariantSuffix)})
		total += fi.Size()
		return nil
	})
	if total <= maxBytes {
		return 0
	}

	if !evictLegacy {
		files = slices.DeleteFunc(files, func(f cacheFile) bool { return !f.variant })
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].variant != files[j].variant {
			return !files[i].variant
		}
		return files[i].mod.Before(files[j].mod)
	})

	for n, f := range files {
		if total <= maxBytes {
			break
		}
		if os.Remove(f.path) == nil {
			total -= f.size
			removed += f.size
		}
		if (n+1)%evictBatch == 0 {
			time.Sleep(pause)
		}
	}
	return removed
}
