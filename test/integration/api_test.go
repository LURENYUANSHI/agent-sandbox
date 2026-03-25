package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/api"
)

func setupTestServer() *httptest.Server {
	srv := api.NewServer(api.ServerConfig{Port: 0, DevMode: true})
	return httptest.NewServer(srv.Router())
}

func postJSON(url string, body any) (*http.Response, error) {
	data, _ := json.Marshal(body)
	return http.Post(url, "application/json", bytes.NewBuffer(data))
}

func doRequest(method, url string, body any) (*http.Response, error) {
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(buf))
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
	ts := setupTestServer()
	defer ts.Close()
	base := ts.URL + "/api/v1"

	// 1. POST /sandboxes -> create
	resp, err := postJSON(base+"/sandboxes", map[string]string{
		"name": "api-test-sandbox",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	var created map[string]interface{}
	decodeJSON(resp, &created)
	sandboxID, ok := created["id"].(string)
	if !ok || sandboxID == "" {
		t.Fatal("create response missing id")
	}

	// 2. POST /sandboxes/:id/start -> start
	resp, err = postJSON(base+"/sandboxes/"+sandboxID+"/start", nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status = %d, want 200", resp.StatusCode)
	}

	// 3. POST /sandboxes/:id/exec -> execute action
	// Default policy is deny-all, so this will be denied (403)
	resp, err = postJSON(base+"/sandboxes/"+sandboxID+"/exec", map[string]interface{}{
		"type":   "file.read",
		"params": map[string]string{"path": "/tmp/test.txt"},
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	// Deny-all policy means 403
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("exec status = %d, want 403 (deny-all policy)", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. GET /sandboxes/:id/traces -> verify traces exist
	resp, err = http.Get(base + "/sandboxes/" + sandboxID + "/traces")
	if err != nil {
		t.Fatalf("traces: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("traces status = %d, want 200", resp.StatusCode)
	}
	var traceResp map[string]interface{}
	decodeJSON(resp, &traceResp)
	events, ok := traceResp["events"].([]interface{})
	if !ok {
		t.Fatal("traces response missing events array")
	}
	// Should have at least the sandbox.started event + action events
	if len(events) < 1 {
		t.Errorf("expected at least 1 trace event, got %d", len(events))
	}

	// 5. POST /sandboxes/:id/stop -> stop
	resp, err = postJSON(base+"/sandboxes/"+sandboxID+"/stop", nil)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d, want 200", resp.StatusCode)
	}

	// 6. DELETE /sandboxes/:id -> destroy
	resp, err = doRequest("DELETE", base+"/sandboxes/"+sandboxID, nil)
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("destroy status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Verify sandbox is gone
	resp, err = http.Get(base + "/sandboxes/" + sandboxID)
	if err != nil {
		t.Fatalf("get after destroy: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after destroy, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAPIPolicyValidation(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	base := ts.URL + "/api/v1"

	// Valid policy YAML -> valid
	validYAML := `name: valid-policy
default_effect: deny
rules:
  - name: allow-reads
    action_type: "file.read"
    effect: allow
`
	resp, err := postJSON(base+"/policies/validate", map[string]string{
		"content": validYAML,
	})
	if err != nil {
		t.Fatalf("validate valid: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("valid policy status = %d, want 200", resp.StatusCode)
	}
	var validResp map[string]interface{}
	decodeJSON(resp, &validResp)
	if validResp["valid"] != true {
		t.Errorf("expected valid = true, got %v", validResp["valid"])
	}

	// Invalid YAML -> parse error
	resp, err = postJSON(base+"/policies/validate", map[string]string{
		"content": "{{invalid yaml content}}",
	})
	if err != nil {
		t.Fatalf("validate invalid: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid policy status = %d, want 200", resp.StatusCode)
	}
	var invalidResp map[string]interface{}
	decodeJSON(resp, &invalidResp)
	if invalidResp["valid"] != false {
		t.Errorf("expected valid = false, got %v", invalidResp["valid"])
	}
}

func TestAPIErrorCases(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	base := ts.URL + "/api/v1"

	// GET nonexistent sandbox -> 404
	resp, err := http.Get(base + "/sandboxes/nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("nonexistent sandbox status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// Create a sandbox but don't start it
	resp, _ = postJSON(base+"/sandboxes", map[string]string{"name": "not-started"})
	var created map[string]interface{}
	decodeJSON(resp, &created)
	notStartedID := created["id"].(string)

	// Execute on non-running sandbox -> should fail
	resp, err = postJSON(base+"/sandboxes/"+notStartedID+"/exec", map[string]interface{}{
		"type":   "file.read",
		"params": map[string]string{"path": "/tmp/test.txt"},
	})
	if err != nil {
		t.Fatalf("exec on stopped: %v", err)
	}
	// The sandbox is not running, so exec returns 403 (action denied error)
	if resp.StatusCode == http.StatusOK {
		t.Errorf("exec on non-started sandbox should not return 200")
	}
	resp.Body.Close()

	// Stop on non-running sandbox -> should fail
	resp, err = postJSON(base+"/sandboxes/"+notStartedID+"/stop", nil)
	if err != nil {
		t.Fatalf("stop non-running: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Errorf("stop on non-running sandbox should not return 200")
	}
	resp.Body.Close()
}

func TestAPIListMultipleSandboxes(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()
	base := ts.URL + "/api/v1"

	for i := 0; i < 3; i++ {
		resp, err := postJSON(base+"/sandboxes", map[string]string{
			"name": fmt.Sprintf("list-test-%d", i),
		})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		resp.Body.Close()
	}

	resp, err := http.Get(base + "/sandboxes")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var listResp map[string]interface{}
	decodeJSON(resp, &listResp)
	sandboxes, ok := listResp["sandboxes"].([]interface{})
	if !ok {
		t.Fatal("list response missing sandboxes array")
	}
	if len(sandboxes) != 3 {
		t.Errorf("expected 3 sandboxes, got %d", len(sandboxes))
	}
}

func TestAPIHealth(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health status = %d, want 200", resp.StatusCode)
	}
	var healthResp map[string]interface{}
	decodeJSON(resp, &healthResp)
	if healthResp["status"] != "ok" {
		t.Errorf("health status = %v, want ok", healthResp["status"])
	}
}
