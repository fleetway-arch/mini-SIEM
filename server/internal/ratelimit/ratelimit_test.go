package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newTestLimiter(limit int, window time.Duration) (*Limiter, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	limiter := New(limit, window)
	limiter.now = clock.Now
	return limiter, clock
}

func TestLimiterBlocksAfterLimitWithRetryAfter(t *testing.T) {
	limiter, clock := newTestLimiter(3, time.Minute)
	for i := 1; i <= 3; i++ {
		if ok, _ := limiter.Allow("ip:1"); !ok {
			t.Fatalf("attempt %d was blocked, limit is 3", i)
		}
	}
	clock.Advance(20 * time.Second)
	ok, retryAfter := limiter.Allow("ip:1")
	if ok {
		t.Fatal("4th attempt inside the window was allowed")
	}
	if retryAfter != 40*time.Second {
		t.Fatalf("retryAfter = %v, want 40s (time left in the window)", retryAfter)
	}
}

func TestLimiterResetsAfterWindow(t *testing.T) {
	limiter, clock := newTestLimiter(1, time.Minute)
	limiter.Allow("k")
	if ok, _ := limiter.Allow("k"); ok {
		t.Fatal("second attempt was allowed")
	}
	clock.Advance(time.Minute)
	if ok, _ := limiter.Allow("k"); !ok {
		t.Fatal("attempt after the window was blocked")
	}
}

func TestLimiterKeysAreIndependent(t *testing.T) {
	limiter, _ := newTestLimiter(1, time.Minute)
	limiter.Allow("login:alice@example.com")
	if ok, _ := limiter.Allow("login:bob@example.com"); !ok {
		t.Fatal("a different key was blocked")
	}
}

func TestLimiterSweepsExpiredEntries(t *testing.T) {
	limiter, clock := newTestLimiter(5, time.Minute)
	for i := 0; i < 100; i++ {
		limiter.Allow(fmt.Sprintf("ip:%d", i))
	}
	clock.Advance(2 * time.Minute)
	limiter.Allow("ip:new")
	if got := len(limiter.entries); got != 1 {
		t.Fatalf("entries after sweep = %d, want 1", got)
	}
}
