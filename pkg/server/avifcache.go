package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/avif"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
)

// ImageCache interface for cache implementations that can be used with image proxy
type ImageCache interface {
	Get(key string) ([]byte, bool)
	Set(key string, data []byte)
	Delete(key string)
}

// AVIFCache wraps a diskcache and converts images to AVIF format on schedule
// It implements the ImageCache interface for use with fallback and heatmap proxies
// Images are stored immediately as-is, and converted to AVIF during scheduled processing
type AVIFCache struct {
	cache           *diskcache.Cache
	quality         int
	speed           int
	pendingFile     string       // File to persist pending conversions
	pendingKeys     []string     // In-memory list of keys pending conversion
	pendingMu       sync.RWMutex // Protects pendingKeys
	processedKeys   sync.Map     // Track already processed keys to avoid duplicates
	isProcessing    bool         // Whether scheduled processing is running
	processingMu    sync.Mutex   // Protects isProcessing
	stopProcessing  chan struct{}
	cacheIdentifier string // Unique identifier for this cache instance (for pending file)
}

// Global AVIF caches that can be accessed by cron
var (
	avifCaches   []*AVIFCache
	avifCachesMu sync.Mutex
)

// NewAVIFCache creates a new AVIF-converting cache wrapper
func NewAVIFCache(cache *diskcache.Cache, identifier string) *AVIFCache {
	c := &AVIFCache{
		cache:           cache,
		quality:         65, // Good balance of quality and size
		speed:           6,  // Balanced speed
		pendingKeys:     make([]string, 0),
		stopProcessing:  make(chan struct{}),
		cacheIdentifier: identifier,
	}

	// Set up pending file path
	c.pendingFile = filepath.Join(common.AppDir, "avif_pending_"+identifier+".json")

	// Load any previously saved pending conversions
	c.loadPendingKeys()

	// Register this cache for scheduled processing
	avifCachesMu.Lock()
	avifCaches = append(avifCaches, c)
	avifCachesMu.Unlock()

	return c
}

// loadPendingKeys loads pending conversion keys from disk
func (c *AVIFCache) loadPendingKeys() {
	data, err := os.ReadFile(c.pendingFile)
	if err != nil {
		if !os.IsNotExist(err) {
			common.Log.Warnf("AVIF cache [%s]: failed to load pending keys: %v", c.cacheIdentifier, err)
		}
		return
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	if err := json.Unmarshal(data, &c.pendingKeys); err != nil {
		common.Log.Warnf("AVIF cache [%s]: failed to parse pending keys: %v", c.cacheIdentifier, err)
		c.pendingKeys = make([]string, 0)
	}

	common.Log.Infof("AVIF cache [%s]: loaded %d pending conversions", c.cacheIdentifier, len(c.pendingKeys))
}

// savePendingKeys persists pending conversion keys to disk
func (c *AVIFCache) savePendingKeys() {
	c.pendingMu.RLock()
	data, err := json.Marshal(c.pendingKeys)
	c.pendingMu.RUnlock()

	if err != nil {
		common.Log.Warnf("AVIF cache [%s]: failed to serialize pending keys: %v", c.cacheIdentifier, err)
		return
	}

	if err := os.WriteFile(c.pendingFile, data, 0644); err != nil {
		common.Log.Warnf("AVIF cache [%s]: failed to save pending keys: %v", c.cacheIdentifier, err)
	}
}

// addPendingKey adds a key to the pending list (thread-safe)
func (c *AVIFCache) addPendingKey(key string) {
	// Skip if already processed or already pending
	if _, exists := c.processedKeys.Load(key); exists {
		return
	}

	c.pendingMu.Lock()
	// Check if already in pending list
	for _, k := range c.pendingKeys {
		if k == key {
			c.pendingMu.Unlock()
			return
		}
	}
	c.pendingKeys = append(c.pendingKeys, key)
	c.pendingMu.Unlock()

	// Save periodically (every 100 additions) to avoid too many disk writes
	c.pendingMu.RLock()
	count := len(c.pendingKeys)
	c.pendingMu.RUnlock()
	if count%100 == 0 {
		c.savePendingKeys()
	}
}

// removePendingKey removes a key from the pending list (thread-safe)
func (c *AVIFCache) removePendingKey(key string) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	for i, k := range c.pendingKeys {
		if k == key {
			c.pendingKeys = append(c.pendingKeys[:i], c.pendingKeys[i+1:]...)
			return
		}
	}
}

// GetPendingCount returns the number of pending conversions
func (c *AVIFCache) GetPendingCount() int {
	c.pendingMu.RLock()
	defer c.pendingMu.RUnlock()
	return len(c.pendingKeys)
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

// Set stores data in the cache and queues for AVIF conversion if scheduled processing is enabled
func (c *AVIFCache) Set(key string, data []byte) {
	// Always store original data immediately (no delay)
	c.cache.Set(key, data)

	// Quick checks - skip small files
	if len(data) < 5000 {
		return
	}

	// Skip HTTP responses (from imageproxy's httpcache) - complex format
	if isHTTPResponse(data) {
		return
	}

	// Check if already AVIF - no conversion needed
	if isAVIF(data) {
		c.processedKeys.Store(key, true)
		common.Log.Debugf("AVIF cache [%s]: %s already AVIF, storing as-is", c.cacheIdentifier, key)
		return
	}

	// Check content type - only convert JPEG and PNG
	contentType := http.DetectContentType(data)
	if !strings.Contains(contentType, "jpeg") && !strings.Contains(contentType, "png") {
		// Skip SVG, GIF, WebP, and unknown formats
		return
	}

	// Add to pending list for scheduled conversion
	c.addPendingKey(key)
	common.Log.Debugf("AVIF cache [%s]: queued %s for scheduled conversion (%s, %d bytes)", c.cacheIdentifier, key, contentType, len(data))
}

// Delete removes data from the cache
func (c *AVIFCache) Delete(key string) {
	c.cache.Delete(key)
	c.removePendingKey(key)
	c.processedKeys.Delete(key)
}

// ProcessPendingConversions processes pending AVIF conversions
// Called by the scheduled task. Returns number processed.
// If endTime is provided, stops processing when current time exceeds endTime.
func (c *AVIFCache) ProcessPendingConversions(endTime *time.Time) int {
	c.processingMu.Lock()
	if c.isProcessing {
		c.processingMu.Unlock()
		common.Log.Infof("AVIF cache [%s]: processing already in progress, skipping", c.cacheIdentifier)
		return 0
	}
	c.isProcessing = true
	c.stopProcessing = make(chan struct{})
	c.processingMu.Unlock()

	defer func() {
		c.processingMu.Lock()
		c.isProcessing = false
		c.processingMu.Unlock()
		// Save remaining pending keys
		c.savePendingKeys()
	}()

	processed := 0
	startTime := time.Now()

	c.pendingMu.RLock()
	totalPending := len(c.pendingKeys)
	c.pendingMu.RUnlock()

	common.Log.Infof("AVIF cache [%s]: starting scheduled conversion of %d images", c.cacheIdentifier, totalPending)

	for {
		// Check if we should stop
		select {
		case <-c.stopProcessing:
			common.Log.Infof("AVIF cache [%s]: processing stopped early after %d conversions", c.cacheIdentifier, processed)
			return processed
		default:
		}

		// Check time window
		if endTime != nil && time.Now().After(*endTime) {
			common.Log.Infof("AVIF cache [%s]: time window ended, processed %d of %d images", c.cacheIdentifier, processed, totalPending)
			return processed
		}

		// Get next pending key
		c.pendingMu.Lock()
		if len(c.pendingKeys) == 0 {
			c.pendingMu.Unlock()
			break
		}
		key := c.pendingKeys[0]
		c.pendingKeys = c.pendingKeys[1:]
		c.pendingMu.Unlock()

		// Process this key
		if c.processConversion(key) {
			processed++
		}

		// Small delay between conversions to prevent overwhelming the system
		time.Sleep(50 * time.Millisecond)

		// Log progress periodically
		if processed%100 == 0 && processed > 0 {
			common.Log.Infof("AVIF cache [%s]: processed %d images so far...", c.cacheIdentifier, processed)
		}
	}

	elapsed := time.Since(startTime)
	common.Log.Infof("AVIF cache [%s]: completed %d conversions in %v", c.cacheIdentifier, processed, elapsed)
	return processed
}

// StopProcessing signals the processing loop to stop
func (c *AVIFCache) StopProcessing() {
	c.processingMu.Lock()
	defer c.processingMu.Unlock()
	if c.isProcessing {
		close(c.stopProcessing)
	}
}

// processConversion converts a single key from the cache to AVIF
func (c *AVIFCache) processConversion(key string) bool {
	// Mark as processed to avoid re-adding
	c.processedKeys.Store(key, true)

	// Recover from any panics during conversion
	defer func() {
		if r := recover(); r != nil {
			common.Log.Errorf("AVIF cache [%s]: panic during conversion of %s: %v", c.cacheIdentifier, key, r)
		}
	}()

	// Get original data from cache
	data, ok := c.cache.Get(key)
	if !ok {
		common.Log.Debugf("AVIF cache [%s]: %s no longer in cache, skipping", c.cacheIdentifier, key)
		return false
	}

	// Skip if already AVIF (might have been converted by another process)
	if isAVIF(data) {
		common.Log.Debugf("AVIF cache [%s]: %s already AVIF, skipping", c.cacheIdentifier, key)
		return false
	}

	// Skip very large images to prevent memory issues (> 10MB source)
	if len(data) > 10*1024*1024 {
		common.Log.Debugf("AVIF cache [%s]: skipping %s - too large (%d bytes)", c.cacheIdentifier, key, len(data))
		return false
	}

	avifData := c.convertToAVIF(data)
	// Only update if conversion succeeded and is smaller
	if avifData != nil && len(avifData) < len(data) {
		c.cache.Set(key, avifData)
		savings := 100 - (len(avifData) * 100 / len(data))
		common.Log.Infof("AVIF [%s]: %s saved %d%% (%d -> %d bytes)",
			c.cacheIdentifier, key, savings, len(data), len(avifData))
		return true
	}

	// Conversion failed or wasn't beneficial - original remains in cache
	return false
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

// GetAllAVIFCaches returns all registered AVIF caches for processing
func GetAllAVIFCaches() []*AVIFCache {
	avifCachesMu.Lock()
	defer avifCachesMu.Unlock()
	return avifCaches
}

// GetTotalPendingConversions returns total pending across all caches
func GetTotalPendingConversions() int {
	total := 0
	for _, cache := range GetAllAVIFCaches() {
		total += cache.GetPendingCount()
	}
	return total
}

// ProcessAllAVIFConversions processes all pending conversions across all caches
func ProcessAllAVIFConversions(endTime *time.Time) int {
	total := 0
	for _, cache := range GetAllAVIFCaches() {
		// Check time before processing each cache
		if endTime != nil && time.Now().After(*endTime) {
			common.Log.Infof("AVIF conversion: time window ended")
			break
		}
		total += cache.ProcessPendingConversions(endTime)
	}
	return total
}

// IsWithinScheduleWindow checks if current time is within the AVIF conversion schedule window
func IsWithinScheduleWindow() bool {
	if !config.Config.Cron.AVIFConversionSchedule.Enabled {
		return false
	}

	if !config.Config.Cron.AVIFConversionSchedule.UseRange {
		// If not using range, always allow (cron handles timing)
		return true
	}

	now := time.Now()
	hour := now.Hour()
	startHour := config.Config.Cron.AVIFConversionSchedule.HourStart
	endHour := config.Config.Cron.AVIFConversionSchedule.HourEnd

	if startHour <= endHour {
		// Normal range (e.g., 10-14)
		return hour >= startHour && hour < endHour
	}
	// Overnight range (e.g., 22-6)
	return hour >= startHour || hour < endHour
}
