package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func newTestServer() *Server {
	return NewServer(ServerConfig{Port: 0, DevMode: true})
}

func doRequest(s *Server, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	return w
}

func parseBody(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

// --- Health ---

func TestHealthEndpoint(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/health", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseBody(w)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
}

// --- Create Sandbox ---

func TestCreateSandbox(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{
		"name": "test-sandbox",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	body := parseBody(w)
	if body["name"] != "test-sandbox" {
		t.Errorf("expected name test-sandbox, got %v", body["name"])
	}
	if body["status"] != "created" {
		t.Errorf("expected status created, got %v", body["status"])
	}
	if body["id"] == nil || body["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestCreateSandboxDefaultName(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	body := parseBody(w)
	name, _ := body["name"].(string)
	if !strings.HasPrefix(name, "sandbox-") {
		t.Errorf("expected default name starting with sandbox-, got %q", name)
	}
}

// --- List Sandboxes ---

func TestListSandboxesEmpty(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/sandboxes", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseBody(w)
	sandboxes, ok := body["sandboxes"].([]interface{})
	if !ok {
		t.Fatal("expected sandboxes array")
	}
	if len(sandboxes) != 0 {
		t.Errorf("expected 0 sandboxes, got %d", len(sandboxes))
	}
}

func TestListSandboxesAfterCreate(t *testing.T) {
	s := newTestServer()
	doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "a"})
	doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "b"})

	w := doRequest(s, "GET", "/api/v1/sandboxes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseBody(w)
	sandboxes := body["sandboxes"].([]interface{})
	if len(sandboxes) != 2 {
		t.Errorf("expected 2 sandboxes, got %d", len(sandboxes))
	}
}

// --- Get Sandbox ---

func TestGetSandbox(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "test"})
	createBody := parseBody(createResp)
	id := createBody["id"].(string)

	w := doRequest(s, "GET", "/api/v1/sandboxes/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseBody(w)
	if body["id"] != id {
		t.Errorf("expected id %s, got %v", id, body["id"])
	}
}

func TestGetSandboxNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/sandboxes/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Start Sandbox ---

func TestStartSandbox(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "test"})
	createBody := parseBody(createResp)
	id := createBody["id"].(string)

	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["status"] != "running" {
		t.Errorf("expected status running, got %v", body["status"])
	}
}

func TestStartSandboxNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes/nonexistent/start", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Stop Sandbox ---

func TestStopSandbox(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "test"})
	id := parseBody(createResp)["id"].(string)

	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/stop", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["status"] != "stopped" {
		t.Errorf("expected status stopped, got %v", body["status"])
	}
}

// --- Destroy Sandbox ---

func TestDestroySandbox(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "test"})
	id := parseBody(createResp)["id"].(string)

	w := doRequest(s, "DELETE", "/api/v1/sandboxes/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseBody(w)
	if body["destroyed"] != true {
		t.Errorf("expected destroyed true, got %v", body["destroyed"])
	}

	// Should be gone now
	w2 := doRequest(s, "GET", "/api/v1/sandboxes/"+id, nil)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after destroy, got %d", w2.Code)
	}
}

func TestDestroyNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "DELETE", "/api/v1/sandboxes/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Exec ---

func TestExecAction(t *testing.T) {
	s := newTestServer()

	// Create and start sandbox with allow-all policy
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "exec-test"})
	id := parseBody(createResp)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Load an allow-all policy so exec succeeds
	s.mu.RLock()
	entry := s.sandboxes[id]
	s.mu.RUnlock()
	entry.Engine.LoadPolicy(allowAllPolicy())

	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/exec", map[string]interface{}{
		"type": "file.write",
		"params": map[string]string{
			"path":    "test.txt",
			"content": "hello",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["success"] != true {
		t.Errorf("expected success true, got %v", body["success"])
	}
}

func TestExecBadRequest(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "test"})
	id := parseBody(createResp)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Send invalid body
	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+id+"/exec", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Traces ---

func TestGetTraces(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "trace-test"})
	id := parseBody(createResp)["id"].(string)

	// Start to generate some trace events
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	w := doRequest(s, "GET", "/api/v1/sandboxes/"+id+"/traces", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseBody(w)
	events, ok := body["events"].([]interface{})
	if !ok {
		t.Fatal("expected events array")
	}
	// Should have at least the sandbox.started event
	if len(events) == 0 {
		t.Error("expected at least one trace event after start")
	}
}

// --- Replay ---

func TestReplay(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "replay-test"})
	id := parseBody(createResp)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Start replay
	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/replay", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["status"] != "started" {
		t.Errorf("expected status started, got %v", body["status"])
	}

	// Get next event
	w2 := doRequest(s, "GET", "/api/v1/sandboxes/"+id+"/replay/next", nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

func TestReplayNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/sandboxes/nonexistent/replay/next", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Policy Validate ---

func TestValidatePolicy(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/policies/validate", map[string]string{
		"content": "name: test\ndescription: test policy\nrules: []\ndefault_effect: allow\n",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["valid"] != true {
		t.Errorf("expected valid true, got %v", body["valid"])
	}
}

func TestValidatePolicyInvalid(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/policies/validate", map[string]string{
		"content": ": : not yaml at all [[[",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	body := parseBody(w)
	if body["valid"] != false {
		t.Errorf("expected valid false, got %v", body["valid"])
	}
}

// --- Middleware ---

func TestRequestIDMiddleware(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/health", nil)

	reqID := w.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("expected X-Request-ID header")
	}
}

func TestCORSHeaders(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:5173" {
		t.Errorf("expected CORS origin http://localhost:5173, got %q", origin)
	}
}

// --- WebSocket ---

func TestWebSocketConnection(t *testing.T) {
	s := newTestServer()

	// Create a sandbox first
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "ws-test"})
	id := parseBody(createResp)["id"].(string)

	// Start httptest server
	ts := httptest.NewServer(s.Router())
	defer ts.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/sandboxes/" + id + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("expected 101, got %d", resp.StatusCode)
	}
}

func TestWebSocketNotFoundSandbox(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/sandboxes/nonexistent/ws", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Filter ---

func TestListSandboxesFilterByStatus(t *testing.T) {
	s := newTestServer()

	// Create two sandboxes, start only one
	r1 := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "a"})
	id1 := parseBody(r1)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id1+"/start", nil)

	doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "b"})

	// Filter by running
	w := doRequest(s, "GET", "/api/v1/sandboxes?status=running", nil)
	body := parseBody(w)
	sandboxes := body["sandboxes"].([]interface{})
	if len(sandboxes) != 1 {
		t.Errorf("expected 1 running sandbox, got %d", len(sandboxes))
	}
}

// --- Dashboard Stats ---

func TestDashboardStatsMultipleSandboxes(t *testing.T) {
	s := newTestServer()

	// Create and start two sandboxes
	r1 := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "ds1"})
	id1 := parseBody(r1)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id1+"/start", nil)

	r2 := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "ds2"})
	id2 := parseBody(r2)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id2+"/start", nil)

	// Create a third sandbox but don't start it
	doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "ds3"})

	w := doRequest(s, "GET", "/api/v1/dashboard/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	active, _ := body["active_sandboxes"].(float64)
	if int(active) != 2 {
		t.Errorf("expected 2 active sandboxes, got %v", active)
	}
}

// --- Dashboard Activity ---

func TestDashboardActivityWithEvents(t *testing.T) {
	s := newTestServer()

	r1 := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "act1"})
	id1 := parseBody(r1)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id1+"/start", nil)

	w := doRequest(s, "GET", "/api/v1/dashboard/activity", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) == 0 {
		t.Error("expected at least one activity event after sandbox start")
	}
}

// --- Audit endpoint ---

func TestAuditEndpointDisabled(t *testing.T) {
	s := newTestServer() // no AuditDBPath
	w := doRequest(s, "GET", "/api/v1/audit", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestAuditEndpointWithFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_api_test.db")
	s := NewServer(ServerConfig{Port: 0, DevMode: true, AuditDBPath: dbPath})
	t.Cleanup(func() { s.auditLogger.Close() })

	// Log some entries
	s.auditLogger.LogDecision("sb-1", "file:read", "/a", "allow", "r1", "", "u1")
	s.auditLogger.LogDecision("sb-2", "file:write", "/b", "deny", "r2", "", "u2")
	s.auditLogger.LogDecision("sb-1", "proc:exec", "/c", "deny", "r3", "", "u1")

	// Filter by sandbox_id
	w := doRequest(s, "GET", "/api/v1/audit?sandbox_id=sb-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	entries := body["entries"].([]interface{})
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for sb-1, got %d", len(entries))
	}

	// Filter by effect
	w = doRequest(s, "GET", "/api/v1/audit?effect=deny", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body = parseBody(w)
	entries = body["entries"].([]interface{})
	if len(entries) != 2 {
		t.Errorf("expected 2 deny entries, got %d", len(entries))
	}

	// Filter by time range
	now := time.Now().UTC()
	start := now.Add(-1 * time.Minute).Format(time.RFC3339)
	end := now.Add(1 * time.Minute).Format(time.RFC3339)
	w = doRequest(s, "GET", "/api/v1/audit?start_time="+start+"&end_time="+end, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body = parseBody(w)
	entries = body["entries"].([]interface{})
	if len(entries) != 3 {
		t.Errorf("expected 3 entries in time range, got %d", len(entries))
	}

	// Filter with limit
	w = doRequest(s, "GET", "/api/v1/audit?limit=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body = parseBody(w)
	entries = body["entries"].([]interface{})
	if len(entries) != 1 {
		t.Errorf("expected 1 entry with limit, got %d", len(entries))
	}
}

func TestAuditEndpointInvalidTime(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_badtime_test.db")
	s := NewServer(ServerConfig{Port: 0, DevMode: true, AuditDBPath: dbPath})
	t.Cleanup(func() { s.auditLogger.Close() })

	w := doRequest(s, "GET", "/api/v1/audit?start_time=not-a-date", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Auth token endpoint ---

func TestAuthTokenEndpoint(t *testing.T) {
	secret := "test-secret-key"
	s := NewServer(ServerConfig{
		Port:        0,
		DevMode:     true,
		AuthEnabled: true,
		AuthSecret:  secret,
	})

	// Generate an admin token so we can make authenticated requests
	token, err := GenerateToken(secret, "admin", "admin", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Request a new token
	body, _ := json.Marshal(map[string]string{
		"user_id": "new-user",
		"role":    "viewer",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	result := parseBody(w)
	if result["token"] == nil || result["token"] == "" {
		t.Error("expected non-empty token in response")
	}
}

func TestAuthTokenEndpointDisabled(t *testing.T) {
	s := NewServer(ServerConfig{Port: 0, DevMode: true, AuthEnabled: false})

	w := doRequest(s, "POST", "/api/v1/auth/token", map[string]string{
		"user_id": "u1",
		"role":    "admin",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthTokenInvalidRole(t *testing.T) {
	secret := "test-secret"
	s := NewServer(ServerConfig{
		Port:        0,
		DevMode:     true,
		AuthEnabled: true,
		AuthSecret:  secret,
	})

	token, _ := GenerateToken(secret, "admin", "admin", time.Hour)
	body, _ := json.Marshal(map[string]string{
		"user_id": "u1",
		"role":    "superuser",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Rate Limiter ---

func TestRateLimiterReturns429(t *testing.T) {
	s := NewServer(ServerConfig{
		Port:           0,
		DevMode:        true,
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	})

	// First request should succeed (uses burst)
	w := doRequest(s, "GET", "/api/v1/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", w.Code)
	}

	// Rapid subsequent requests should eventually hit 429
	got429 := false
	for i := 0; i < 10; i++ {
		w = doRequest(s, "GET", "/api/v1/health", nil)
		if w.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Error("expected 429 after exceeding rate limit")
	}
}

// --- Validation Errors ---

func TestValidationErrorStructured(t *testing.T) {
	s := newTestServer()

	// Create sandbox with invalid name (starts with hyphen)
	w := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{
		"name": "-invalid-name",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var result ValidationErrors
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result.Errors) == 0 {
		t.Error("expected validation errors in response")
	}
	if result.Errors[0].Field != "name" {
		t.Errorf("expected field 'name', got %q", result.Errors[0].Field)
	}
}

func TestExecValidationErrorMissingType(t *testing.T) {
	s := newTestServer()
	r := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "val-test"})
	id := parseBody(r)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/exec", map[string]interface{}{
		"type":   "",
		"params": map[string]string{"path": "/tmp/x"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var result ValidationErrors
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result.Errors) == 0 {
		t.Error("expected validation errors")
	}
}

// --- WebSocket with message ---

func TestWebSocketReceivesEvents(t *testing.T) {
	s := newTestServer()

	r := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "ws-event"})
	id := parseBody(r)["id"].(string)

	ts := httptest.NewServer(s.Router())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/sandboxes/" + id + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()

	// Start the sandbox — this triggers a trace event broadcast
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Connection should be alive; write a ping
	err = conn.WriteMessage(websocket.PingMessage, nil)
	if err != nil {
		t.Errorf("expected ws connection to be alive: %v", err)
	}
}

// --- Auth middleware with invalid format ---

func TestAuthInvalidFormat(t *testing.T) {
	s := NewServer(ServerConfig{
		Port:        0,
		DevMode:     true,
		AuthEnabled: true,
		AuthSecret:  "test-secret",
	})
	req := httptest.NewRequest("GET", "/api/v1/sandboxes", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic auth, not Bearer
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-Bearer auth, got %d", w.Code)
	}
}

// --- Metrics endpoint ---

func TestMetricsEndpoint(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /metrics, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") && !strings.Contains(ct, "openmetrics") {
		t.Errorf("expected text/plain content type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "agentsandbox_api_requests_total") {
		t.Error("expected Prometheus metrics in /metrics response")
	}
}

// --- Stop not found ---

func TestStopSandboxNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes/nonexistent/stop", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Exec not found ---

func TestExecSandboxNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes/nonexistent/exec", map[string]interface{}{
		"type":   "file.read",
		"params": map[string]string{"path": "/tmp/x"},
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Traces not found ---

func TestGetTracesNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "GET", "/api/v1/sandboxes/nonexistent/traces", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Replay not found for start ---

func TestReplayStartNotFound(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes/nonexistent/replay", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- Exec with empty body ---

func TestExecEmptyBody(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "body-test"})
	id := parseBody(createResp)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	req := httptest.NewRequest("POST", "/api/v1/sandboxes/"+id+"/exec", nil)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 0
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Create with empty body ---

func TestCreateSandboxEmptyBody(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("POST", "/api/v1/sandboxes", nil)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 0
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Validate policy with direct JSON object ---

func TestValidatePolicyDirect(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/policies/validate", map[string]interface{}{
		"name":           "direct-policy",
		"version":        "1.0",
		"default_effect": "allow",
		"rules":          []interface{}{},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["valid"] != true {
		t.Errorf("expected valid=true, got %v", body["valid"])
	}
}

// --- Replay done ---

func TestReplayDone(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "replay-done"})
	id := parseBody(createResp)["id"].(string)

	// Start replay
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/replay", nil)

	// Advance until done
	for i := 0; i < 100; i++ {
		w := doRequest(s, "GET", "/api/v1/sandboxes/"+id+"/replay/next", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := parseBody(w)
		if done, ok := body["done"]; ok && done == true {
			return // success
		}
		if hasMore, ok := body["has_more"]; ok && hasMore == false {
			// Next call should return done
			w = doRequest(s, "GET", "/api/v1/sandboxes/"+id+"/replay/next", nil)
			body = parseBody(w)
			if done, ok := body["done"]; ok && done == true {
				return
			}
		}
	}
}

// --- Swagger endpoint ---

func TestSwaggerEndpoint(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// Swagger may or may not be configured, but shouldn't panic
	if w.Code == http.StatusInternalServerError {
		t.Fatalf("swagger endpoint returned 500")
	}
}

// --- Start already running ---

func TestStartSandboxAlreadyRunning(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "double-start"})
	id := parseBody(createResp)["id"].(string)

	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Second start should return 409
	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already running sandbox, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Stop already stopped ---

func TestStopSandboxAlreadyStopped(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "double-stop"})
	id := parseBody(createResp)["id"].(string)

	// Don't start it — try to stop directly
	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/stop", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for not-running sandbox, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Validate policy missing name ---

func TestValidatePolicyMissingName(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/policies/validate", map[string]interface{}{
		"name":           "",
		"default_effect": "allow",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Recovery middleware ---

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- effectForEvent and detailForEvent ---

func TestEffectForEvent(t *testing.T) {
	tests := []struct {
		eventType types.EventType
		expected  string
	}{
		{types.EventActionDenied, "deny"},
		{types.EventActionFailed, "deny"},
		{types.EventActionAllowed, "allow"},
		{types.EventActionExecuted, "allow"},
		{types.EventSandboxStarted, "audit"},
		{types.EventSandboxStopped, "audit"},
	}
	for _, tt := range tests {
		ev := types.TraceEvent{Type: tt.eventType}
		got := effectForEvent(ev)
		if got != tt.expected {
			t.Errorf("effectForEvent(%s) = %q, want %q", tt.eventType, got, tt.expected)
		}
	}
}

func TestDetailForEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    types.TraceEvent
		contains string
	}{
		{
			"sandbox started with name",
			types.TraceEvent{Type: types.EventSandboxStarted, Data: map[string]string{"name": "test"}},
			"Sandbox started: test",
		},
		{
			"sandbox started no name",
			types.TraceEvent{Type: types.EventSandboxStarted, Data: map[string]string{}},
			"Sandbox started",
		},
		{
			"sandbox stopped",
			types.TraceEvent{Type: types.EventSandboxStopped},
			"Sandbox stopped",
		},
		{
			"sandbox created",
			types.TraceEvent{Type: types.EventSandboxCreated},
			"Sandbox created",
		},
		{
			"action requested with type",
			types.TraceEvent{Type: types.EventActionRequested, Data: map[string]string{"type": "file.read"}},
			"Action requested: file.read",
		},
		{
			"action requested no type",
			types.TraceEvent{Type: types.EventActionRequested, Data: map[string]string{}},
			"Action requested",
		},
		{
			"policy evaluated allowed",
			types.TraceEvent{Type: types.EventPolicyEvaluated, Data: map[string]string{"allowed": "true"}},
			"Policy evaluated: true",
		},
		{
			"policy evaluated no data",
			types.TraceEvent{Type: types.EventPolicyEvaluated, Data: map[string]string{}},
			"Policy evaluated",
		},
		{
			"action denied with reason",
			types.TraceEvent{Type: types.EventActionDenied, Data: map[string]string{"reason": "forbidden"}},
			"Action denied: forbidden",
		},
		{
			"action denied no reason",
			types.TraceEvent{Type: types.EventActionDenied, Data: map[string]string{}},
			"Action denied",
		},
		{
			"action executed",
			types.TraceEvent{Type: types.EventActionExecuted},
			"Action executed",
		},
		{
			"action failed with error",
			types.TraceEvent{Type: types.EventActionFailed, Data: map[string]string{"error": "timeout"}},
			"Action failed: timeout",
		},
		{
			"action failed no error",
			types.TraceEvent{Type: types.EventActionFailed, Data: map[string]string{}},
			"Action failed",
		},
		{
			"unknown event type",
			types.TraceEvent{Type: "custom.event"},
			"custom.event",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detailForEvent(tt.event)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("detailForEvent() = %q, want containing %q", got, tt.contains)
			}
		})
	}
}

// --- Helpers ---

func allowAllPolicy() types.Policy {
	return types.Policy{
		Name:          "allow-all",
		DefaultEffect: types.EffectAllow,
	}
}
