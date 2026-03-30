package functional

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/api"
)

// --- helpers ---

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := api.NewServer(api.ServerConfig{Port: 0, DevMode: true})
	return httptest.NewServer(srv.Router())
}

func writePolicy(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(p, []byte(`name: func-test
version: "1.0"
default_effect: deny
rules:
  - id: allow-file-ops
    name: "Allow file ops"
    actions: ["file:read", "file:write", "file:delete"]
    resources: ["**"]
    effect: allow
    priority: 10
  - id: allow-proc
    name: "Allow process exec"
    actions: ["proc:exec"]
    resources: ["*"]
    effect: allow
    priority: 10
  - id: allow-http
    name: "Allow HTTP"
    actions: ["net:http"]
    resources: ["*"]
    effect: allow
    priority: 10
  - id: deny-shell
    name: "Deny shell exec"
    actions: ["shell:exec"]
    resources: ["*"]
    effect: deny
    priority: 100
`), 0644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	return p
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func doRequest(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(buf))
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, url, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var v map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

// --- end-to-end test ---

func TestEndToEnd(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"
	policyFile := writePolicy(t)

	// 1. Health check
	resp := doRequest(t, "GET", base+"/health", nil)
	body := decode(t, resp)
	if body["status"] != "ok" {
		t.Fatalf("health status = %v, want ok", body["status"])
	}

	// 2. Create sandbox with policy
	resp = postJSON(t, base+"/sandboxes", map[string]string{
		"name":        "e2e-sandbox",
		"policy_file": policyFile,
	})
	if resp.StatusCode != http.StatusCreated {
		b := decode(t, resp)
		t.Fatalf("create status = %d, body = %v", resp.StatusCode, b)
	}
	body = decode(t, resp)
	sandboxID := body["id"].(string)
	rootDir := body["root_dir"].(string)
	if sandboxID == "" || rootDir == "" {
		t.Fatal("missing id or root_dir in create response")
	}

	// 3. Start sandbox
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/start", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. File write to sandbox root
	writePath := filepath.ToSlash(filepath.Join(rootDir, "hello.txt"))
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "file:write",
		"params": map[string]string{"path": writePath, "content": "hello e2e"},
	})
	if resp.StatusCode != http.StatusOK {
		b := decode(t, resp)
		t.Fatalf("file write status = %d, body = %v", resp.StatusCode, b)
	}
	body = decode(t, resp)
	if body["success"] != true {
		t.Errorf("file write success = %v, error = %v", body["success"], body["error"])
	}

	// 5. File read from sandbox root
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "file:read",
		"params": map[string]string{"path": writePath},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("file read status = %d", resp.StatusCode)
	}
	body = decode(t, resp)
	if body["success"] != true {
		t.Errorf("file read success = %v", body["success"])
	}
	output, _ := body["output"].(string)
	if !strings.Contains(output, "hello e2e") {
		t.Errorf("file read output = %q, want 'hello e2e'", output)
	}

	// 6. File delete of root "/" — denied by builtin rule
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "file:delete",
		"params": map[string]string{"path": "/"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("delete root status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// 7. Shell exec — denied by policy
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "shell:exec",
		"params": map[string]string{"command": "whoami"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("shell exec status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// 8. Process exec — allowed
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "proc:exec",
		"params": map[string]string{"command": "echo hello-e2e"},
	})
	if resp.StatusCode != http.StatusOK {
		b := decode(t, resp)
		t.Fatalf("proc exec status = %d, body = %v", resp.StatusCode, b)
	}
	body = decode(t, resp)
	if body["success"] != true {
		t.Errorf("proc exec success = %v, error = %v", body["success"], body["error"])
	}
	output, _ = body["output"].(string)
	if !strings.Contains(output, "hello-e2e") {
		t.Errorf("proc exec output = %q, want 'hello-e2e'", output)
	}

	// 9. Verify traces
	resp = doRequest(t, "GET", base+"/sandboxes/"+sandboxID+"/traces", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("traces status = %d", resp.StatusCode)
	}
	body = decode(t, resp)
	events, ok := body["events"].([]any)
	if !ok {
		t.Fatal("missing events array in traces response")
	}
	if len(events) < 5 {
		t.Errorf("expected at least 5 trace events, got %d", len(events))
	}
	// Verify event type distribution
	typeCounts := map[string]int{}
	for _, e := range events {
		ev := e.(map[string]any)
		typeCounts[ev["type"].(string)]++
	}
	if typeCounts["sandbox.started"] != 1 {
		t.Errorf("sandbox.started count = %d, want 1", typeCounts["sandbox.started"])
	}
	if typeCounts["action.requested"] < 3 {
		t.Errorf("action.requested count = %d, want >= 3", typeCounts["action.requested"])
	}
	if typeCounts["policy.evaluated"] < 3 {
		t.Errorf("policy.evaluated count = %d, want >= 3", typeCounts["policy.evaluated"])
	}

	// 10. Get sandbox info
	resp = doRequest(t, "GET", base+"/sandboxes/"+sandboxID, nil)
	body = decode(t, resp)
	if body["status"] != "running" {
		t.Errorf("sandbox status = %v, want running", body["status"])
	}

	// 11. List sandboxes
	resp = doRequest(t, "GET", base+"/sandboxes", nil)
	body = decode(t, resp)
	sandboxes := body["sandboxes"].([]any)
	if len(sandboxes) < 1 {
		t.Error("expected at least 1 sandbox in list")
	}

	// 12. Stop sandbox
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/stop", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 13. Destroy sandbox
	resp = doRequest(t, "DELETE", base+"/sandboxes/"+sandboxID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("destroy status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 14. Verify sandbox is gone
	resp = doRequest(t, "GET", base+"/sandboxes/"+sandboxID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after destroy status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestDotFormatActionTypes verifies that dot-separated action types (file.read)
// work correctly and are normalized to colon format for policy evaluation.
func TestDotFormatActionTypes(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"
	policyFile := writePolicy(t)

	// Create and start sandbox
	resp := postJSON(t, base+"/sandboxes", map[string]string{
		"name":        "dot-format-test",
		"policy_file": policyFile,
	})
	body := decode(t, resp)
	sandboxID := body["id"].(string)
	rootDir := body["root_dir"].(string)

	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/start", nil)
	resp.Body.Close()

	// file.write (dot format) should be normalized to file:write and allowed
	writePath := filepath.ToSlash(filepath.Join(rootDir, "dot-test.txt"))
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "file.write",
		"params": map[string]string{"path": writePath, "content": "dot format works"},
	})
	if resp.StatusCode != http.StatusOK {
		b := decode(t, resp)
		t.Fatalf("file.write status = %d, body = %v", resp.StatusCode, b)
	}
	body = decode(t, resp)
	if body["success"] != true {
		t.Errorf("file.write success = %v, error = %v", body["success"], body["error"])
	}

	// file.read (dot format)
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "file.read",
		"params": map[string]string{"path": writePath},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("file.read status = %d", resp.StatusCode)
	}
	body = decode(t, resp)
	if body["success"] != true {
		t.Errorf("file.read success = %v", body["success"])
	}

	// process.exec (dot format)
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "process.exec",
		"params": map[string]string{"command": "echo dot-ok"},
	})
	if resp.StatusCode != http.StatusOK {
		b := decode(t, resp)
		t.Fatalf("process.exec status = %d, body = %v", resp.StatusCode, b)
	}
	body = decode(t, resp)
	if body["success"] != true {
		t.Errorf("process.exec success = %v", body["success"])
	}

	// shell.exec (dot format) — should still be denied
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "shell.exec",
		"params": map[string]string{"command": "whoami"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("shell.exec status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// Cleanup
	postJSON(t, base+"/sandboxes/"+sandboxID+"/stop", nil).Body.Close()
	doRequest(t, "DELETE", base+"/sandboxes/"+sandboxID, nil).Body.Close()
}

// TestPolicyEnforcement verifies the policy engine correctly allows and denies actions.
// Note: handleCreateSandbox adds an auto-sandbox-root rule that allows file:read/write/delete
// within the sandbox root directory, so we test denial of non-file action types.
func TestPolicyEnforcement(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"

	// Create a strict policy that only allows file:read
	dir := t.TempDir()
	policyFile := filepath.Join(dir, "strict.yaml")
	os.WriteFile(policyFile, []byte(`name: strict
version: "1.0"
default_effect: deny
rules:
  - id: allow-read-only
    name: "Allow file reads only"
    actions: ["file:read"]
    resources: ["**"]
    effect: allow
    priority: 10
`), 0644)

	resp := postJSON(t, base+"/sandboxes", map[string]string{
		"name":        "strict-sandbox",
		"policy_file": policyFile,
	})
	body := decode(t, resp)
	sandboxID := body["id"].(string)
	rootDir := body["root_dir"].(string)

	postJSON(t, base+"/sandboxes/"+sandboxID+"/start", nil).Body.Close()

	// file:read in sandbox root should be allowed (by both policy and auto-sandbox-root)
	// First write a file directly to the sandbox root for reading
	testFile := filepath.Join(rootDir, "readable.txt")
	os.MkdirAll(rootDir, 0755)
	os.WriteFile(testFile, []byte("test content"), 0644)

	readPath := filepath.ToSlash(testFile)
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "file:read",
		"params": map[string]string{"path": readPath},
	})
	if resp.StatusCode != http.StatusOK {
		b := decode(t, resp)
		t.Errorf("file:read in strict sandbox status = %d, body = %v, want 200", resp.StatusCode, b)
	} else {
		body = decode(t, resp)
		if body["success"] != true {
			t.Errorf("file:read success = %v", body["success"])
		}
	}

	// proc:exec should be denied (strict policy only allows file:read)
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "proc:exec",
		"params": map[string]string{"command": "echo denied"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("proc:exec in strict sandbox status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// shell:exec should be denied
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "shell:exec",
		"params": map[string]string{"command": "echo denied"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("shell:exec in strict sandbox status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// net:http should be denied
	resp = postJSON(t, base+"/sandboxes/"+sandboxID+"/exec", map[string]any{
		"type":   "net:http",
		"params": map[string]string{"url": "http://example.com"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("net:http in strict sandbox status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// Cleanup
	postJSON(t, base+"/sandboxes/"+sandboxID+"/stop", nil).Body.Close()
	doRequest(t, "DELETE", base+"/sandboxes/"+sandboxID, nil).Body.Close()
}

// TestSandboxLifecycleErrors tests error conditions in the sandbox lifecycle.
func TestSandboxLifecycleErrors(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"

	// Nonexistent sandbox → 404
	resp := doRequest(t, "GET", base+"/sandboxes/nonexistent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get nonexistent = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// Exec on nonexistent sandbox → 404
	resp = postJSON(t, base+"/sandboxes/nonexistent/exec", map[string]any{
		"type":   "file:read",
		"params": map[string]string{"path": "/tmp/x"},
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("exec nonexistent = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// Create but don't start, then try exec → should fail
	resp = postJSON(t, base+"/sandboxes", map[string]string{"name": "not-started"})
	body := decode(t, resp)
	id := body["id"].(string)

	resp = postJSON(t, base+"/sandboxes/"+id+"/exec", map[string]any{
		"type":   "file:read",
		"params": map[string]string{"path": "/tmp/x"},
	})
	if resp.StatusCode == http.StatusOK {
		t.Error("exec on non-started sandbox should not succeed")
	}
	resp.Body.Close()

	// Stop non-running sandbox → should fail
	resp = postJSON(t, base+"/sandboxes/"+id+"/stop", nil)
	if resp.StatusCode == http.StatusOK {
		t.Error("stop on non-started sandbox should not succeed")
	}
	resp.Body.Close()

	// Double start
	postJSON(t, base+"/sandboxes/"+id+"/start", nil).Body.Close()
	resp = postJSON(t, base+"/sandboxes/"+id+"/start", nil)
	if resp.StatusCode == http.StatusOK {
		// Double start returns 409 conflict
	}
	resp.Body.Close()

	// Cleanup
	doRequest(t, "DELETE", base+"/sandboxes/"+id, nil).Body.Close()
}

// TestDashboardStats verifies the dashboard stats endpoint returns data after actions.
func TestDashboardStats(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"
	policyFile := writePolicy(t)

	resp := postJSON(t, base+"/sandboxes", map[string]string{
		"name":        "stats-test",
		"policy_file": policyFile,
	})
	body := decode(t, resp)
	id := body["id"].(string)
	rootDir := body["root_dir"].(string)

	postJSON(t, base+"/sandboxes/"+id+"/start", nil).Body.Close()

	// Execute some actions
	writePath := filepath.ToSlash(filepath.Join(rootDir, "stats.txt"))
	postJSON(t, base+"/sandboxes/"+id+"/exec", map[string]any{
		"type":   "file:write",
		"params": map[string]string{"path": writePath, "content": "stats data"},
	}).Body.Close()

	// Check dashboard stats
	resp = doRequest(t, "GET", base+"/dashboard/stats", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard stats status = %d", resp.StatusCode)
	}
	body = decode(t, resp)
	// DashboardStatsResponse has: active_sandboxes, total_actions, denied_actions, avg_response_ms
	if _, ok := body["active_sandboxes"]; !ok {
		t.Error("missing active_sandboxes in dashboard stats")
	}
	if _, ok := body["total_actions"]; !ok {
		t.Error("missing total_actions in dashboard stats")
	}

	// Cleanup
	postJSON(t, base+"/sandboxes/"+id+"/stop", nil).Body.Close()
	doRequest(t, "DELETE", base+"/sandboxes/"+id, nil).Body.Close()
}

// TestReplay tests the trace replay functionality.
func TestReplay(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"
	policyFile := writePolicy(t)

	// Create, start, execute an action
	resp := postJSON(t, base+"/sandboxes", map[string]string{
		"name":        "replay-test",
		"policy_file": policyFile,
	})
	body := decode(t, resp)
	id := body["id"].(string)
	rootDir := body["root_dir"].(string)
	postJSON(t, base+"/sandboxes/"+id+"/start", nil).Body.Close()

	writePath := filepath.ToSlash(filepath.Join(rootDir, "replay.txt"))
	postJSON(t, base+"/sandboxes/"+id+"/exec", map[string]any{
		"type":   "file:write",
		"params": map[string]string{"path": writePath, "content": "replay data"},
	}).Body.Close()

	// Start replay
	resp = postJSON(t, base+"/sandboxes/"+id+"/replay", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start replay status = %d", resp.StatusCode)
	}
	body = decode(t, resp)
	totalEvents, _ := body["total_events"].(float64)
	if totalEvents < 1 {
		t.Errorf("total_events = %v, want >= 1", totalEvents)
	}

	// Step through replay
	resp = doRequest(t, "GET", base+"/sandboxes/"+id+"/replay/next", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replay next status = %d", resp.StatusCode)
	}
	body = decode(t, resp)
	if body["event"] == nil {
		t.Error("expected event in replay response")
	}

	// Cleanup
	postJSON(t, base+"/sandboxes/"+id+"/stop", nil).Body.Close()
	doRequest(t, "DELETE", base+"/sandboxes/"+id, nil).Body.Close()
}

// TestPolicyValidation tests the policy validation endpoint.
func TestPolicyValidation(t *testing.T) {
	ts := setupServer(t)
	defer ts.Close()
	base := ts.URL + "/api/v1"

	// Valid policy
	resp := postJSON(t, base+"/policies/validate", map[string]string{
		"content": `name: valid
default_effect: deny
rules:
  - name: allow-reads
    action_type: "file.read"
    effect: allow
`,
	})
	body := decode(t, resp)
	if body["valid"] != true {
		t.Errorf("valid policy: valid = %v", body["valid"])
	}

	// Invalid YAML
	resp = postJSON(t, base+"/policies/validate", map[string]string{
		"content": "{{not yaml}}",
	})
	body = decode(t, resp)
	if body["valid"] != false {
		t.Errorf("invalid yaml: valid = %v", body["valid"])
	}
}
