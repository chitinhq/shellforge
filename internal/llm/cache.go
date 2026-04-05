package llm

import (
	"fmt"
	"hash/fnv"
)

// CacheStats tracks prompt caching effectiveness over a session.
type CacheStats struct {
	Hits            int    // number of requests with cache hits
	Misses          int    // number of requests with no cache activity
	Breaks          int    // cache breaks: expected hit but got creation instead
	TotalCreated    int    // total tokens written to cache
	TotalHit        int    // total tokens read from cache
	Requests        int    // total Chat() calls tracked
	SavedTokens     int    // estimated tokens saved (TotalHit)
	prevFingerprint uint64 // fingerprint of previous request for break detection
}

// CacheTracker wraps a Provider and tracks cache metrics.
// It implements Provider so it can be used as a drop-in replacement.
type CacheTracker struct {
	inner Provider
	stats CacheStats
}

// NewCacheTracker wraps a provider with cache tracking.
func NewCacheTracker(p Provider) *CacheTracker {
	return &CacheTracker{inner: p}
}

// Chat delegates to the inner provider and tracks cache metrics.
func (ct *CacheTracker) Chat(messages []Message, tools []ToolDef) (*Response, error) {
	resp, err := ct.inner.Chat(messages, tools)
	if err != nil {
		return nil, err
	}

	ct.stats.Requests++

	fp := fingerprint(messages, tools)

	switch {
	case resp.CacheHit > 0:
		ct.stats.Hits++
		ct.stats.TotalHit += resp.CacheHit
		ct.stats.SavedTokens = ct.stats.TotalHit
	case resp.CacheCreated > 0:
		ct.stats.TotalCreated += resp.CacheCreated
		if ct.stats.prevFingerprint != 0 && fp == ct.stats.prevFingerprint {
			// Same request content but cache missed — this is a break.
			ct.stats.Breaks++
		}
	default:
		// No cache activity at all.
		ct.stats.Misses++
	}

	ct.stats.prevFingerprint = fp

	return resp, nil
}

// Name delegates to the inner provider.
func (ct *CacheTracker) Name() string {
	return ct.inner.Name()
}

// Stats returns a copy of the current cache statistics.
func (ct *CacheTracker) Stats() CacheStats {
	return ct.stats
}

// fingerprint computes an FNV-1a hash of the request (messages + tools)
// for cache break detection.
func fingerprint(messages []Message, tools []ToolDef) uint64 {
	h := fnv.New64a()
	for _, m := range messages {
		fmt.Fprintf(h, "%s:%s;", m.Role, m.Content)
	}
	for _, t := range tools {
		fmt.Fprintf(h, "t:%s;", t.Name)
	}
	return h.Sum64()
}
