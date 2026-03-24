package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/api"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func setupTestServer() (*httptest.Server, func()) {
	srv := api.NewServer()
	ts := httptest.NewServer(srv.Handler())
	return ts, ts.Close
}

func postJSON(url string, body any) (*http.Response, error) {
	data, _ := json.Marshal(body)
	return http.Post(url, "application/json", bytes.NewBuffer(data))
}

func doRequest(method, url string, body any) (*http.Response, error) {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func decodeJSON(resp *http.Response, v any) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

func TestAPIFullWorkflow(t *testing.T) {
	ts, cleanup := setupTestServer()
	defer cleanup()

	// 1. POST /api/sandboxes -> create
	resp, err := postJSON(ts.URL+"/api/sandboxes", map[string]string{
		"id": "api-test-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}

	// 2. POST /api/sandboxes/:id/start -> start
	resp, err = postJSON(ts.URL+"/api/sandboxes/api-test-1/start", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status = %d, want 200", resp.StatusCode)
	}

	// 3. POST /api/sandboxes/:id/exec -> execute action
	resp, err = postJSON(ts.URL+"/api/sandboxes/api-test-1/exec", map[string]any{
		"action": map[string]any{
			"type":    "file",
			"path":    "/tmp/test.txt",
			"file_op": "read",
		},
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exec status = %d, want 200", resp.StatusCode)
	}
	var event types.TraceEvent
	decodeJSON(resp, &event)
	if event.Decision != types.DecisionAllowed {
		t.Errorf("exec decision = %s, want allowed", event.Decision)
	}

	// 4. GET /api/sandboxes/:id/traces -> verify traces
	resp, err = http.Get(ts.URL + "/api/sandboxes/api-test-1/traces")
	if err != nil {
		t.Fatalf("traces: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("traces status = %d, want 200", resp.StatusCode)
	}
	var traces []types.TraceEvent
	decodeJSON(resp, &traces)
	if len(traces) != 1 {
		t.Errorf("expected 1 trace, got %d", len(traces))
	}

	// 5. POST /api/sandboxes/:id/stop -> stop
	resp, err = postJSON(ts.URL+"/api/sandboxes/api-test-1/stop", nil)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d, want 200", resp.StatusCode)
	}

	// 6. DELETE /api/sandboxes/:id -> destroy
	resp, err = doRequest("DELETE", ts.URL+"/api/sandboxes/api-test-1", nil)
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("destroy status = %d, want 200", resp.StatusCode)
	}

	// Verify sandbox is gone
	resp, err = http.Get(ts.URL + "/api/sandboxes/api-test-1")
	if err != nil {
		t.Fatalf("get after destroy: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after destroy, got %d", resp.StatusCode)
	}
}

func TestAPIPolicyValidation(t *testing.T) {
	ts, cleanup := setupTestServer()
	defer cleanup()

	// Valid policy -> 200
	validPolicy := types.Policy{
		Name:          "valid",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:    "r1",
				Effect:  types.EffectAllow,
				Actions: []types.ActionType{types.ActionTypeFile},
			},
		},
	}
	resp, err := postJSON(ts.URL+"/api/policies/validate", validPolicy)
	if err != nil {
		t.Fatalf("validate valid: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("valid policy status = %d, want 200", resp.StatusCode)
	}

	// Invalid policy (missing name) -> 400
	invalidPolicy := types.Policy{
		DefaultEffect: types.EffectDeny,
	}
	resp, err = postJSON(ts.URL+"/api/policies/validate", invalidPolicy)
	if err != nil {
		t.Fatalf("validate invalid: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid policy status = %d, want 400", resp.StatusCode)
	}

	// Invalid policy (bad effect) -> 400
	badEffect := types.Policy{
		Name:          "bad",
		DefaultEffect: "maybe",
	}
	resp, err = postJSON(ts.URL+"/api/policies/validate", badEffect)
	if err != nil {
		t.Fatalf("validate bad effect: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad effect status = %d, want 400", resp.StatusCode)
	}
}

func TestAPIErrorCases(t *testing.T) {
	ts, cleanup := setupTestServer()
	defer cleanup()

	// GET nonexistent sandbox -> 404
	resp, err := http.Get(ts.URL + "/api/sandboxes/nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("nonexistent sandbox status = %d, want 404", resp.StatusCode)
	}

	// Create a sandbox but don't start it
	postJSON(ts.URL+"/api/sandboxes", map[string]string{"id": "not-started"})

	// Execute on non-running sandbox -> 400
	resp, err = postJSON(ts.URL+"/api/sandboxes/not-started/exec", map[string]any{
		"action": map[string]any{
			"type":    "file",
			"path":    "/tmp/test.txt",
			"file_op": "read",
		},
	})
	if err != nil {
		t.Fatalf("exec on stopped: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("exec on stopped status = %d, want 400", resp.StatusCode)
	}

	// Create and start sandbox, then send invalid action type
	postJSON(ts.URL+"/api/sandboxes", map[string]string{"id": "invalid-action"})
	postJSON(ts.URL+"/api/sandboxes/invalid-action/start", nil)

	resp, err = postJSON(ts.URL+"/api/sandboxes/invalid-action/exec", map[string]any{
		"action": map[string]any{},
	})
	if err != nil {
		t.Fatalf("exec invalid: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid action status = %d, want 400", resp.StatusCode)
	}
}

func TestAPIDuplicateSandbox(t *testing.T) {
	ts, cleanup := setupTestServer()
	defer cleanup()

	postJSON(ts.URL+"/api/sandboxes", map[string]string{"id": "dup-1"})

	resp, err := postJSON(ts.URL+"/api/sandboxes", map[string]string{"id": "dup-1"})
	if err != nil {
		t.Fatalf("create duplicate: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate status = %d, want 409", resp.StatusCode)
	}
}

func TestAPIListMultipleSandboxes(t *testing.T) {
	ts, cleanup := setupTestServer()
	defer cleanup()

	for i := 0; i < 3; i++ {
		postJSON(ts.URL+"/api/sandboxes", map[string]string{
			"id": fmt.Sprintf("list-test-%d", i),
		})
	}

	resp, err := http.Get(ts.URL + "/api/sandboxes")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var sandboxes []json.RawMessage
	decodeJSON(resp, &sandboxes)
	if len(sandboxes) != 3 {
		t.Errorf("expected 3 sandboxes, got %d", len(sandboxes))
	}
}
