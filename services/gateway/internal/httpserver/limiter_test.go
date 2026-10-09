package httpserver

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimiterBoundsAndRefill(t *testing.T) {
	l := newLimiter(2)
	now := time.Now()
	if !l.allow("a", 1, 2, now) || !l.allow("a", 1, 2, now) || l.allow("a", 1, 2, now) {
		t.Fatal("burst not enforced")
	}
	if l.allow("a", 1, 2, now.Add(-time.Second)) {
		t.Fatal("clock reversal refilled bucket")
	}
	if !l.allow("a", 1, 2, now.Add(time.Second)) {
		t.Fatal("refill failed")
	}
	if !l.allow("b", 1, 2, now) || l.allow("c", 1, 2, now) {
		t.Fatal("capacity not enforced")
	}
	if !l.allow("c", 1, 2, now.Add(6*time.Minute)) || len(l.buckets) > 2 {
		t.Fatal("idle cleanup failed")
	}
}
func TestLimiterConcurrentBurst(t *testing.T) {
	l := newLimiter(2)
	now := time.Now()
	var accepted atomic.Int64
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			if l.allow("shared", 1, 10, now) {
				accepted.Add(1)
			}
		})
	}
	wg.Wait()
	if accepted.Load() != 10 {
		t.Fatalf("accepted %d", accepted.Load())
	}
}
