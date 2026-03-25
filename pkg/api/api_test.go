package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
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

// --- Helpers ---

func allowAllPolicy() types.Policy {
	return types.Policy{
		Name:          "allow-all",
		DefaultEffect: types.EffectAllow,
	}
}
