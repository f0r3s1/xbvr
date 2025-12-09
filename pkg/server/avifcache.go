package server

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/avif"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/xbapps/xbvr/pkg/common"
)

// ImageCache interface for cache implementations that can be used with image proxy
type ImageCache interface {
	Get(key string) ([]byte, bool)
	Set(key string, data []byte)
	Delete(key string)
}

// conversionJob represents a pending AVIF conversion
type conversionJob struct {
	key  string
	data []byte
}

// AVIFCache wraps a diskcache and converts images to AVIF format in background
// It implements the ImageCache interface for use with fallback and heatmap proxies
type AVIFCache struct {
	cache      *diskcache.Cache
	quality    int
	speed      int
	jobQueue   chan conversionJob
	processing sync.Map // track keys being processed to avoid duplicates
}

// NewAVIFCache creates a new AVIF-converting cache wrapper
func NewAVIFCache(cache *diskcache.Cache) *AVIFCache {
	c := &AVIFCache{
		cache:   cache,
		quality: 65, // Good balance of quality and size
		speed:   6,  // Balanced speed
		// Buffered channel to queue conversion jobs without blocking
		jobQueue: make(chan conversionJob, 10000),
	}
	// Start single background worker - processes jobs sequentially
	go c.conversionWorker()
	return c
}

// conversionWorker processes AVIF conversions in the background with rate limiting
func (c *AVIFCache) conversionWorker() {
	// Recover from any panics that might crash the worker
	defer func() {
		if r := recover(); r != nil {
			common.Log.Errorf("AVIF cache: worker panic, restarting: %v", r)
			// Restart the worker
			go c.conversionWorker()
		}
	}()

	for job := range c.jobQueue {
		c.processConversion(job)
		// Small delay between conversions to prevent overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}
}

// processConversion converts data to AVIF and updates the cache
func (c *AVIFCache) processConversion(job conversionJob) {
	defer c.processing.Delete(job.key)

	// Recover from any panics during conversion
	defer func() {
		if r := recover(); r != nil {
			common.Log.Errorf("AVIF cache: panic during conversion of %s: %v", job.key, r)
		}
	}()

	// Skip very large images to prevent memory issues (> 10MB source)
	if len(job.data) > 10*1024*1024 {
		common.Log.Debugf("AVIF cache: skipping %s - too large (%d bytes)", job.key, len(job.data))
		return
	}

	avifData := c.convertToAVIF(job.data)
	// Only update if conversion succeeded and is smaller
	if avifData != nil && len(avifData) < len(job.data) {
		c.cache.Set(job.key, avifData)
		savings := 100 - (len(avifData) * 100 / len(job.data))
		common.Log.Infof("AVIF: %s saved %d%% (%d -> %d bytes)",
			job.key, savings, len(job.data), len(avifData))
	}
	// If conversion fails or isn't smaller, original is already stored - do nothing
}

// isHTTPResponse checks if data looks like an HTTP response
func isHTTPResponse(data []byte) bool {
	return bytes.HasPrefix(data, []byte("HTTP/"))
}

// GetUnderlying returns the underlying diskcache for compatibility with imageproxy
func (c *AVIFCache) GetUnderlying() *diskcache.Cache {
	return c.cache
}

// Get retrieves data from the cache
func (c *AVIFCache) Get(key string) ([]byte, bool) {
	data, ok := c.cache.Get(key)
	if ok {
		ct := http.DetectContentType(data)
		common.Log.Debugf("AVIF cache GET %s: %s (%d bytes)", key, ct, len(data))
	}
	return data, ok
}

// isAVIF checks if data starts with AVIF magic bytes (ftyp box with avif/avis brand)
func isAVIF(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	// AVIF files start with ftyp box: [4-byte size][ftyp][brand]
	// Check for "ftyp" at offset 4 and "avif" or "avis" at offset 8
	if data[4] == 'f' && data[5] == 't' && data[6] == 'y' && data[7] == 'p' {
		brand := string(data[8:12])
		return brand == "avif" || brand == "avis" || brand == "mif1"
	}
	return false
}

// Set stores data in the cache and queues AVIF conversion in background
func (c *AVIFCache) Set(key string, data []byte) {
	// Quick checks before storing/queueing
	if len(data) < 5000 {
		c.cache.Set(key, data)
		return
	}

	// Skip HTTP responses (from imageproxy's httpcache) - complex format
	if isHTTPResponse(data) {
		c.cache.Set(key, data)
		return
	}

	// Check if already AVIF - no conversion needed, just store
	if isAVIF(data) {
		c.cache.Set(key, data)
		common.Log.Debugf("AVIF cache: %s already AVIF, storing as-is", key)
		return
	}

	// Check content type - only convert JPEG and PNG
	contentType := http.DetectContentType(data)
	if !strings.Contains(contentType, "jpeg") && !strings.Contains(contentType, "png") {
		// Skip SVG, GIF, WebP, and unknown formats - store as-is
		c.cache.Set(key, data)
		return
	}

	// Store original data immediately (non-blocking)
	c.cache.Set(key, data)

	// Skip if already being processed
	if _, exists := c.processing.LoadOrStore(key, true); exists {
		return
	}

	common.Log.Debugf("AVIF cache: queueing %s for conversion (%s, %d bytes)", key, contentType, len(data))

	// Queue for background conversion (non-blocking, drop if queue full)
	select {
	case c.jobQueue <- conversionJob{key: key, data: data}:
		// Job queued successfully
	default:
		// Queue full - just skip conversion, original is already stored
		c.processing.Delete(key)
		common.Log.Warnf("AVIF cache: queue full, skipping %s", key)
	}
}

// Delete removes data from the cache
func (c *AVIFCache) Delete(key string) {
	c.cache.Delete(key)
}

// convertToAVIF attempts to convert JPEG/PNG image data to AVIF format
// Returns nil if conversion fails or isn't possible
func (c *AVIFCache) convertToAVIF(data []byte) (result []byte) {
	// Recover from any panics during encoding
	defer func() {
		if r := recover(); r != nil {
			common.Log.Errorf("AVIF encode panic: %v", r)
			result = nil
		}
	}()

	if len(data) == 0 {
		return nil
	}

	contentType := http.DetectContentType(data)

	// Only convert JPEG and PNG - these benefit from AVIF compression
	var img image.Image
	var err error
	reader := bytes.NewReader(data)

	switch {
	case strings.Contains(contentType, "jpeg"):
		img, err = jpeg.Decode(reader)
	case strings.Contains(contentType, "png"):
		img, err = png.Decode(reader)
	default:
		// SVG, GIF, WebP, AVIF, or unknown - don't convert
		return nil
	}

	if err != nil {
		// Can't decode - might be corrupted or unsupported variant
		return nil
	}

	// Check image dimensions - skip very large images to prevent memory issues
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width > 8000 || height > 8000 || (width*height) > 20000000 {
		common.Log.Debugf("AVIF: skipping image %dx%d - too large", width, height)
		return nil
	}

	var buf bytes.Buffer
	options := avif.Options{
		Quality: c.quality,
		Speed:   c.speed,
	}

	if err := avif.Encode(&buf, img, options); err != nil {
		common.Log.Debugf("AVIF encode failed: %v", err)
		return nil
	}

	return buf.Bytes()
}
