package server

import (
	"sync"
	"time"
)

// ponytail: process-local sliding window keyed by IP. Fine at ~32 pairs;
// move to Redis if we run more than one API replica.
type ipLimiter struct {
	mu     sync.Mutex
	n      int
	window time.Duration
	hits   map[string][]time.Time
}

func newIPLimiter(n int, window time.Duration) *ipLimiter {
	if n <= 0 {
		return nil
	}
	if window <= 0 {
		window = time.Minute
	}
	return &ipLimiter{n: n, window: window, hits: make(map[string][]time.Time)}
}

func (l *ipLimiter) allow(ip string, now time.Time) bool {
	if l == nil {
		return true
	}
	if ip == "" {
		ip = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	q := l.hits[ip]
	i := 0
	for i < len(q) && !q[i].After(cutoff) {
		i++
	}
	q = q[i:]
	if len(q) >= l.n {
		l.hits[ip] = q
		return false
	}
	l.hits[ip] = append(q, now)
	return true
}
