package httpserver

import (
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	seen   time.Time
}
type limiter struct {
	mu       sync.Mutex
	buckets  map[string]bucket
	capacity int
}

func newLimiter(capacity int) *limiter {
	return &limiter{buckets: make(map[string]bucket), capacity: capacity}
}
func (l *limiter) allow(key string, rate float64, burst float64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.capacity {
			for k, v := range l.buckets {
				if now.Sub(v.seen) > 5*time.Minute {
					delete(l.buckets, k)
				}
			}
		}
		if len(l.buckets) >= l.capacity {
			return false
		}
		b = bucket{tokens: burst, seen: now}
	}
	b.tokens = min(burst, b.tokens+max(0, now.Sub(b.seen).Seconds())*rate)
	b.seen = now
	allowed := b.tokens >= 1
	if allowed {
		b.tokens--
	}
	l.buckets[key] = b
	return allowed
}
