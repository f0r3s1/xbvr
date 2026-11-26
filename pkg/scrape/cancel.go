package scrape

import "sync/atomic"

var scrapeCancelled atomic.Bool

// ScrapeCancelled reports whether the user asked to stop the running scrape.
func ScrapeCancelled() bool { return scrapeCancelled.Load() }

// SetScrapeCancelled raises or clears the stop request.
func SetScrapeCancelled(v bool) { scrapeCancelled.Store(v) }
