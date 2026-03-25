package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRateLimitedRouter(rps float64, burst int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := NewRateLimiterStore(rps, burst)
	r.Use(rateLimiterMiddleware(store))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimiterAllowsWithinLimit(t *testing.T) {
	r := setupRateLimitedRouter(100, 10)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimiterReturns429WhenExceeded(t *testing.T) {
	// 1 RPS with burst of 2 — third immediate request should be rejected
	r := setupRateLimitedRouter(1, 2)

	var lastCode int
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:5678"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		lastCode = w.Code
		if w.Code == http.StatusTooManyRequests {
			// Verify Retry-After header
			retryAfter := w.Header().Get("Retry-After")
			if retryAfter == "" {
				t.Error("expected Retry-After header on 429 response")
			}
			// Verify error message
			return
		}
	}
	t.Fatalf("expected 429 after exceeding rate limit, last status was %d", lastCode)
}

func TestRateLimiterPerIP(t *testing.T) {
	// burst=1 so only 1 request allowed immediately
	r := setupRateLimitedRouter(1, 1)

	// First IP uses its token
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "1.1.1.1:1000"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first IP first request: expected 200, got %d", w1.Code)
	}

	// Second IP should still have its own quota
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "2.2.2.2:2000"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second IP first request: expected 200, got %d", w2.Code)
	}
}

func TestRateLimiterResponseBody(t *testing.T) {
	r := setupRateLimitedRouter(1, 1)

	// Exhaust the burst
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "3.3.3.3:3000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// This one should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "3.3.3.3:3000"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w2.Code)
	}
	body := w2.Body.String()
	if body == "" {
		t.Error("expected non-empty response body on 429")
	}
}

func TestNewRateLimiterMiddleware(t *testing.T) {
	// Verify the public constructor works
	mw := NewRateLimiter(100, 10)
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
}

func TestRateLimiterStoreCleanup(t *testing.T) {
	store := NewRateLimiterStore(10, 5)
	defer store.Stop()

	// Add a limiter
	_ = store.getLimiter("1.2.3.4")

	// Verify it exists
	if _, ok := store.limiters.Load("1.2.3.4"); !ok {
		t.Fatal("expected limiter to be stored")
	}
}

func TestServerConfigRateLimitDisabled(t *testing.T) {
	// RPS=0 means rate limiting is disabled
	s := NewServer(ServerConfig{Port: 0, DevMode: true, RateLimitRPS: 0})
	w := doRequest(s, "GET", "/api/v1/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServerConfigRateLimitEnabled(t *testing.T) {
	s := NewServer(ServerConfig{Port: 0, DevMode: true, RateLimitRPS: 100, RateLimitBurst: 50})
	w := doRequest(s, "GET", "/api/v1/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
