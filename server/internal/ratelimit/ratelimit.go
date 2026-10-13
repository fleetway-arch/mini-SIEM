// Package ratelimit is a small in-memory fixed-window limiter. It is per
// process, so limits are not shared between several API instances.
package ratelimit

import (
	"sync"
	"time"
)

type entry struct {
	count int
	start time.Time
}

type Limiter struct {
	limit     int
	window    time.Duration
	now       func() time.Time
	mu        sync.Mutex
	entries   map[string]*entry
	lastSweep time.Time
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, now: time.Now, entries: map[string]*entry{}}
}

// Allow records one attempt for key. When the limit is reached it returns
// false and how long the caller must wait before the window resets.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	e, ok := l.entries[key]
	if !ok || now.Sub(e.start) >= l.window {
		l.entries[key] = &entry{count: 1, start: now}
		return true, 0
	}
	if e.count >= l.limit {
		return false, e.start.Add(l.window).Sub(now)
	}
	e.count++
	return true, 0
}

// sweep drops expired windows at most once per window so memory stays bounded.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	for key, e := range l.entries {
		if now.Sub(e.start) >= l.window {
			delete(l.entries, key)
		}
	}
	l.lastSweep = now
}
