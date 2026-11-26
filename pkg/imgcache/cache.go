// Package imgcache is the cache behind the image proxy:
package imgcache

import (
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/peterbourgon/diskv"
)

// VariantSuffix marks variant files on disk.
const VariantSuffix = ".v"

// touchAfter limits how often a read refreshes a file's mtime (the LRU clock).
const touchAfter = time.Hour

type Cache struct {
	disk *diskv.Diskv
	mem  *memLRU
}

func New(disk *diskv.Diskv, memBytes int64) *Cache {
	return &Cache{disk: disk, mem: newMemLRU(memBytes)}
}

// isVariant reports whether key is a transformed image (imageproxy uses URL#options).
func isVariant(key string) bool { return strings.Contains(key, "#") }

func hashName(key string) string {
	sum := md5.Sum([]byte(key))
	return hex.EncodeToString(sum[:])
}

// path mirrors the two level directory layout of the diskv transform.
func (c *Cache) path(name string) string {
	return filepath.Join(c.disk.BasePath, name[0:2], name[2:4], name)
}

func (c *Cache) Get(key string) ([]byte, bool) {
	name := hashName(key)
	if !isVariant(key) {
		if b, ok := c.mem.get(key); ok {
			return b, true
		}
		b, err := c.disk.Read(name)
		return b, err == nil
	}
	if b, err := c.disk.Read(name + VariantSuffix); err == nil {
		c.touch(c.path(name + VariantSuffix))
		return b, true
	}
	b, err := c.disk.Read(name)
	if err != nil {
		// variants cached before the default q80 option live under the key without it
		if alt := strings.Replace(key, ",q80", "", 1); alt != key {
			if b, ok := c.Get(alt); ok {
				c.Set(key, b)
				return b, true
			}
		}
		return nil, false
	}
	// promote a legacy variant so it ranks as a variant for eviction
	_ = os.Rename(c.path(name), c.path(name+VariantSuffix))
	return b, true
}

func (c *Cache) Set(key string, data []byte) {
	if !isVariant(key) {
		c.mem.set(key, data)
		return
	}
	_ = c.disk.Write(hashName(key)+VariantSuffix, data)
}

func (c *Cache) Delete(key string) {
	name := hashName(key)
	c.mem.delete(key)
	_ = c.disk.Erase(name)
	_ = c.disk.Erase(name + VariantSuffix)
}

// Has reports whether a variant is on disk without reading it.
func (c *Cache) Has(key string) bool {
	name := hashName(key)
	for _, n := range []string{name + VariantSuffix, name} {
		if _, err := os.Stat(c.path(n)); err == nil {
			return true
		}
	}
	return false
}

func (c *Cache) touch(p string) {
	if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) > touchAfter {
		now := time.Now()
		_ = os.Chtimes(p, now, now)
	}
}

type memEntry struct {
	key  string
	data []byte
}

// memLRU is an LRU bounded by total bytes.
type memLRU struct {
	mu    sync.Mutex
	max   int64
	size  int64
	order *list.List
	items map[string]*list.Element
}

func newMemLRU(max int64) *memLRU {
	return &memLRU{max: max, order: list.New(), items: map[string]*list.Element{}}
}

func (m *memLRU) get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	el, ok := m.items[key]
	if !ok {
		return nil, false
	}
	m.order.MoveToFront(el)
	return el.Value.(*memEntry).data, true
}

func (m *memLRU) set(key string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.remove(key)
	if int64(len(data)) > m.max {
		return
	}
	m.items[key] = m.order.PushFront(&memEntry{key, data})
	m.size += int64(len(data))
	for m.size > m.max {
		m.remove(m.order.Back().Value.(*memEntry).key)
	}
}

func (m *memLRU) delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.remove(key)
}

func (m *memLRU) remove(key string) {
	if el, ok := m.items[key]; ok {
		m.size -= int64(len(el.Value.(*memEntry).data))
		m.order.Remove(el)
		delete(m.items, key)
	}
}
