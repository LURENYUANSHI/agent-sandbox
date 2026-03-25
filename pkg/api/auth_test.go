package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testSecret = "test-secret-key-for-jwt"

func newAuthServer() *Server {
	return NewServer(ServerConfig{
		Port:        0,
		DevMode:     true,
		AuthEnabled: true,
		AuthSecret:  testSecret,
	})
}

func doAuthRequest(s *Server, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	return w
}

func TestAuthMissingToken(t *testing.T) {
	s := newAuthServer()
	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthInvalidToken(t *testing.T) {
	s := newAuthServer()
	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", "not-a-valid-token")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthExpiredToken(t *testing.T) {
	s := newAuthServer()
	token, err := GenerateToken(testSecret, "user1", "admin", -1*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthValidToken(t *testing.T) {
	s := newAuthServer()
	token, err := GenerateToken(testSecret, "user1", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHealthBypass(t *testing.T) {
	s := newAuthServer()
	w := doAuthRequest(s, "GET", "/api/v1/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthDisabled(t *testing.T) {
	s := NewServer(ServerConfig{
		Port:        0,
		DevMode:     true,
		AuthEnabled: false,
	})
	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with auth disabled, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthWrongSecret(t *testing.T) {
	s := newAuthServer()
	token, err := GenerateToken("wrong-secret", "user1", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
