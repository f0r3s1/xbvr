package imgcache

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/peterbourgon/diskv"
)

func newTestCache(t *testing.T, memBytes int64) (*Cache, *diskv.Diskv) {
	d := diskv.New(diskv.Options{
		BasePath:  t.TempDir(),
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return New(d, memBytes), d
}

func countFiles(t *testing.T, dir string) (n int) {
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			n++
		}
		return nil
	})
	return
}

func TestOriginalsStayInMemory(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	c.Set("http://x/a.jpg", []byte("orig"))
	if b, ok := c.Get("http://x/a.jpg"); !ok || string(b) != "orig" {
		t.Fatal("original not served from memory")
	}
	if n := countFiles(t, d.BasePath); n != 0 {
		t.Fatalf("original written to disk (%d files)", n)
	}
}

func TestVariantsGoToDisk(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	c.Set("http://x/a.jpg#200x", []byte("var"))
	if n := countFiles(t, d.BasePath); n != 1 {
		t.Fatalf("want 1 disk file, got %d", n)
	}
	c2 := New(d, 1<<20) // fresh memory, same disk
	if b, ok := c2.Get("http://x/a.jpg#200x"); !ok || string(b) != "var" {
		t.Fatal("variant not read back from disk")
	}
	c.Delete("http://x/a.jpg#200x")
	if _, ok := c.Get("http://x/a.jpg#200x"); ok {
		t.Fatal("variant still present after delete")
	}
}

func TestLegacyEntriesStillRead(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	// legacy layout: plain md5 name, for both originals and variants
	d.Write(hashName("http://x/a.jpg"), []byte("old original"))
	d.Write(hashName("http://x/a.jpg#700x"), []byte("old variant"))

	if b, ok := c.Get("http://x/a.jpg"); !ok || string(b) != "old original" {
		t.Fatal("legacy original not read")
	}
	if b, ok := c.Get("http://x/a.jpg#700x"); !ok || string(b) != "old variant" {
		t.Fatal("legacy variant not read")
	}
	if _, err := os.Stat(c.path(hashName("http://x/a.jpg#700x") + VariantSuffix)); err != nil {
		t.Fatal("legacy variant was not promoted")
	}
	if _, ok := c.Get("http://x/a.jpg#700x"); !ok {
		t.Fatal("promoted variant not readable")
	}
}

func TestMemoryBound(t *testing.T) {
	c, _ := newTestCache(t, 10)
	c.Set("a", []byte("123456"))
	c.Set("b", []byte("123456"))
	if _, ok := c.Get("a"); ok {
		t.Fatal("oldest entry not evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("newest entry missing")
	}
	c.Set("big", bytes.Repeat([]byte("x"), 11))
	if _, ok := c.Get("big"); ok {
		t.Fatal("oversized entry stored")
	}
}

func TestEvictOrder(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	write := func(name string, age time.Duration) {
		p := filepath.Join(dir, name)
		os.WriteFile(p, bytes.Repeat([]byte("x"), 10), 0o644)
		os.Chtimes(p, now.Add(-age), now.Add(-age))
	}
	write("origNew", time.Hour)
	write("origOld", 3*time.Hour)
	write("varOld"+VariantSuffix, 5*time.Hour)
	write("varNew"+VariantSuffix, time.Minute)

	if got := Evict(dir, 100, 0, true); got != 0 {
		t.Fatalf("under cap, removed %d", got)
	}
	// 40 bytes, cap 25: originals go first (oldest first), then stop
	Evict(dir, 25, 0, true)
	if _, err := os.Stat(filepath.Join(dir, "origOld")); err == nil {
		t.Fatal("old original kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "origNew")); err == nil {
		t.Fatal("new original kept before variants")
	}
	if _, err := os.Stat(filepath.Join(dir, "varOld"+VariantSuffix)); err != nil {
		t.Fatal("variant removed while under cap")
	}
	// cap 10: the older variant goes before the newer one
	Evict(dir, 10, 0, true)
	if _, err := os.Stat(filepath.Join(dir, "varOld"+VariantSuffix)); err == nil {
		t.Fatal("older variant kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "varNew"+VariantSuffix)); err != nil {
		t.Fatal("newer variant evicted")
	}
}

func TestQ80AliasesLegacyKey(t *testing.T) {
	c, d := newTestCache(t, 1<<20)
	d.Write(hashName("http://x/a.jpg#700x"), []byte("old variant"))

	b, ok := c.Get("http://x/a.jpg#700x,q80")
	if !ok || string(b) != "old variant" {
		t.Fatal("legacy variant not found through q80 key")
	}
	if _, err := os.Stat(c.path(hashName("http://x/a.jpg#700x,q80") + VariantSuffix)); err != nil {
		t.Fatal("alias not copied to the new key")
	}
	if _, ok := c.Get("http://x/b.jpg#700x,q80"); ok {
		t.Fatal("unexpected hit")
	}
}

func TestEvictKeepsLegacyWhenDeferred(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "legacy"), bytes.Repeat([]byte("x"), 10), 0o644)
	os.WriteFile(filepath.Join(dir, "a"+VariantSuffix), bytes.Repeat([]byte("x"), 10), 0o644)

	Evict(dir, 0, 0, false)
	if _, err := os.Stat(filepath.Join(dir, "legacy")); err != nil {
		t.Fatal("legacy file evicted while deferred")
	}
	if _, err := os.Stat(filepath.Join(dir, "a"+VariantSuffix)); err == nil {
		t.Fatal("variant not evicted")
	}
	Evict(dir, 0, 0, true)
	if _, err := os.Stat(filepath.Join(dir, "legacy")); err == nil {
		t.Fatal("legacy file kept after deferral")
	}
}
