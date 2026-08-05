package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type IPRateLimiter struct {
	mu         sync.Mutex
	limiters   map[string]*ipLimiterEntry
	maxTokens  float64
	refillRate float64
	idleTTL    time.Duration
}

type ipLimiterEntry struct {
	limiter  *RateLimiter
	lastSeen time.Time
}

func NewIPRateLimiter(maxTokens, refillRate float64) *IPRateLimiter {
	rl := &IPRateLimiter{
		limiters:   make(map[string]*ipLimiterEntry),
		maxTokens:  maxTokens,
		refillRate: refillRate,
		idleTTL:    10 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *IPRateLimiter) getLimiter(key string) *RateLimiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.limiters[key]
	if !ok {
		entry = &ipLimiterEntry{limiter: NewRateLimiter(rl.maxTokens, rl.refillRate)}
		rl.limiters[key] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-rl.idleTTL)
		rl.mu.Lock()
		for key, entry := range rl.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := rl.getLimiter(clientIP(r))
		if !limiter.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
