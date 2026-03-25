package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter holds a rate limiter and its last-seen time for cleanup.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterStore manages per-IP rate limiters.
type RateLimiterStore struct {
	limiters sync.Map
	rps      rate.Limit
	burst    int
	stopCh   chan struct{}
}

// NewRateLimiterStore creates a new store and starts a background cleanup goroutine.
func NewRateLimiterStore(requestsPerSecond float64, burst int) *RateLimiterStore {
	s := &RateLimiterStore{
		rps:    rate.Limit(requestsPerSecond),
		burst:  burst,
		stopCh: make(chan struct{}),
	}
	go s.cleanup(3 * time.Minute)
	return s
}

// getLimiter returns the rate limiter for a given IP, creating one if needed.
func (s *RateLimiterStore) getLimiter(ip string) *rate.Limiter {
	if v, ok := s.limiters.Load(ip); ok {
		entry := v.(*ipLimiter)
		entry.lastSeen = time.Now()
		return entry.limiter
	}
	limiter := rate.NewLimiter(s.rps, s.burst)
	s.limiters.Store(ip, &ipLimiter{limiter: limiter, lastSeen: time.Now()})
	return limiter
}

// cleanup removes stale entries that haven't been seen for the given TTL.
func (s *RateLimiterStore) cleanup(ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-ttl)
			s.limiters.Range(func(key, value any) bool {
				entry := value.(*ipLimiter)
				if entry.lastSeen.Before(cutoff) {
					s.limiters.Delete(key)
				}
				return true
			})
		case <-s.stopCh:
			return
		}
	}
}

// Stop halts the background cleanup goroutine.
func (s *RateLimiterStore) Stop() {
	close(s.stopCh)
}

// NewRateLimiter returns a gin middleware that enforces per-IP rate limiting.
func NewRateLimiter(requestsPerSecond float64, burst int) gin.HandlerFunc {
	store := NewRateLimiterStore(requestsPerSecond, burst)
	return rateLimiterMiddleware(store)
}

func rateLimiterMiddleware(store *RateLimiterStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)
		if !limiter.Allow() {
			retryAfter := 1.0 / float64(store.rps)
			c.Header("Retry-After", strconv.Itoa(int(retryAfter)+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}
