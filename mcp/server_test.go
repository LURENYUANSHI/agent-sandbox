package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *MCPServer {
	t.Helper()
	return NewMCPServer("")
}

// --- JSON-RPC protocol tests ---

func TestHandleInitialize(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}
	resp := s.handleRequest(req)
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(mcpInitializeResult)
	if !ok {
		t.Fatalf("expected mcpInitializeResult, got %T", resp.Result)
	}
	if result.ServerInfo.Name != "agent-sandbox" {
		t.Errorf("server name = %s, want agent-sandbox", result.ServerInfo.Name)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocol version = %s, want 2024-11-05", result.ProtocolVersion)
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability")
	}
}

func TestHandleToolsList(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}
	resp := s.handleRequest(req)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(mcpToolsListResult)
	if !ok {
		t.Fatalf("expected mcpToolsListResult, got %T", resp.Result)
	}

	expected := map[string]bool{
		"sandbox_create":       false,
		"sandbox_exec":         false,
		"sandbox_stop":         false,
		"sandbox_traces":       false,
		"sandbox_policy_check": false,
	}
	for _, tool := range result.Tools {
		if _, ok := expected[tool.Name]; ok {
			expected[tool.Name] = true
		}
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("tool %s has nil input schema", tool.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestHandlePing(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "ping",
	}
	resp := s.handleRequest(req)
	if resp.Error != nil {
		t.Fatalf("ping error: %s", resp.Error.Message)
	}
}

func TestHandleInitialized(t *testing.T) {
	s := newTestServer(t)
	// With ID (should return empty result)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "initialized",
	}
	resp := s.handleRequest(req)
	if resp.Error != nil {
		t.Fatalf("initialized error: %s", resp.Error.Message)
	}

	// Without ID (notification)
	req = jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}
	resp = s.handleRequest(req)
	if resp.Error != nil {
		t.Fatalf("initialized notification error: %s", resp.Error.Message)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`5`),
		Method:  "nonexistent/method",
	}
	resp := s.handleRequest(req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func TestHandleToolsCallInvalidParams(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`6`),
		Method:  "tools/call",
		Params:  json.RawMessage(`not valid json`),
	}
	resp := s.handleRequest(req)
	if resp.Error == nil {
		t.Fatal("expected error for invalid params")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleToolsCallUnknownTool(t *testing.T) {
	s := newTestServer(t)
	params, _ := json.Marshal(mcpToolCallParams{Name: "nonexistent_tool", Arguments: json.RawMessage(`{}`)})
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`7`),
		Method:  "tools/call",
		Params:  params,
	}
	resp := s.handleRequest(req)
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(mcpToolCallResult)
	if !ok {
		t.Fatalf("expected mcpToolCallResult, got %T", resp.Result)
	}
	if !result.IsError {
		t.Error("expected tool error for unknown tool")
	}
}

// --- Tool handler tests ---

func TestToolSandboxCreateAndExecAndStop(t *testing.T) {
	s := newTestServer(t)

	// Create sandbox
	args, _ := json.Marshal(createArgs{Name: "test-mcp"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create error: %s", result.Content[0].Text)
	}

	var createResp map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	sandboxID, ok := createResp["sandbox_id"].(string)
	if !ok || sandboxID == "" {
		t.Fatal("missing sandbox_id")
	}
	if createResp["status"] != "running" {
		t.Errorf("status = %v, want running", createResp["status"])
	}

	// Execute a process (will be denied by default deny-all policy)
	execArg, _ := json.Marshal(execArgs{
		SandboxID:  sandboxID,
		ActionType: "process.exec",
		Params:     map[string]string{"command": "echo hello"},
	})
	result = s.toolSandboxExec(execArg)
	// Without a loaded policy, default is deny
	var execResp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &execResp)
	// Either denied by policy or executed — both valid responses
	if result.IsError {
		t.Fatalf("exec returned unexpected error: %s", result.Content[0].Text)
	}

	// Get traces
	tracesArg, _ := json.Marshal(tracesArgs{SandboxID: sandboxID, Limit: 50})
	result = s.toolSandboxTraces(tracesArg)
	if result.IsError {
		t.Fatalf("traces error: %s", result.Content[0].Text)
	}
	var tracesResp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &tracesResp)
	eventCount, _ := tracesResp["event_count"].(float64)
	if eventCount < 1 {
		t.Errorf("expected at least 1 trace event, got %v", eventCount)
	}

	// Stop sandbox
	stopArg, _ := json.Marshal(stopArgs{SandboxID: sandboxID})
	result = s.toolSandboxStop(stopArg)
	if result.IsError {
		t.Fatalf("stop error: %s", result.Content[0].Text)
	}
	var stopResp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &stopResp)
	if stopResp["status"] != "stopped" {
		t.Errorf("status = %v, want stopped", stopResp["status"])
	}
}

func TestToolSandboxCreateRequiresName(t *testing.T) {
	s := newTestServer(t)
	args, _ := json.Marshal(createArgs{Name: ""})
	result := s.toolSandboxCreate(args)
	if !result.IsError {
		t.Error("expected error when name is empty")
	}
}

func TestToolSandboxCreateInvalidArgs(t *testing.T) {
	s := newTestServer(t)
	result := s.toolSandboxCreate(json.RawMessage(`not json`))
	if !result.IsError {
		t.Error("expected error for invalid arguments")
	}
}

func TestToolSandboxExecNotFound(t *testing.T) {
	s := newTestServer(t)
	args, _ := json.Marshal(execArgs{
		SandboxID:  "nonexistent-sandbox",
		ActionType: "file.read",
		Params:     map[string]string{"path": "/tmp/x"},
	})
	result := s.toolSandboxExec(args)
	if !result.IsError {
		t.Error("expected error for nonexistent sandbox")
	}
	if !strings.Contains(result.Content[0].Text, "not found") {
		t.Errorf("error should mention 'not found', got: %s", result.Content[0].Text)
	}
}

func TestToolSandboxStopNotFound(t *testing.T) {
	s := newTestServer(t)
	args, _ := json.Marshal(stopArgs{SandboxID: "nonexistent"})
	result := s.toolSandboxStop(args)
	if !result.IsError {
		t.Error("expected error for nonexistent sandbox")
	}
}

func TestToolSandboxTracesNotFound(t *testing.T) {
	s := newTestServer(t)
	args, _ := json.Marshal(tracesArgs{SandboxID: "nonexistent"})
	result := s.toolSandboxTraces(args)
	if !result.IsError {
		t.Error("expected error for nonexistent sandbox")
	}
}

func TestToolSandboxPolicyCheck(t *testing.T) {
	s := newTestServer(t)

	args, _ := json.Marshal(policyCheckArgs{
		ActionType: "file.read",
		Resource:   "/tmp/test.txt",
	})
	result := s.toolSandboxPolicyCheck(args)
	if result.IsError {
		t.Fatalf("policy check error: %s", result.Content[0].Text)
	}

	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	if resp["effect"] == nil {
		t.Error("missing effect in policy check response")
	}
	if _, ok := resp["allowed"]; !ok {
		t.Error("missing allowed in policy check response")
	}
}

func TestToolSandboxPolicyCheckInvalidArgs(t *testing.T) {
	s := newTestServer(t)
	result := s.toolSandboxPolicyCheck(json.RawMessage(`not json`))
	if !result.IsError {
		t.Error("expected error for invalid args")
	}
}

func TestToolSandboxCreateWithPolicyFile(t *testing.T) {
	dir := t.TempDir()
	policyFile := filepath.Join(dir, "policy.yaml")
	err := os.WriteFile(policyFile, []byte(`name: test-policy
version: "1.0"
default_effect: deny
rules:
  - id: allow-read
    name: "Allow file reads"
    actions: ["file:read"]
    resources: ["**"]
    effect: allow
    priority: 10
`), 0644)
	if err != nil {
		t.Fatalf("write policy: %v", err)
	}

	s := newTestServer(t)
	args, _ := json.Marshal(createArgs{Name: "policy-test", Policy: policyFile})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create with policy error: %s", result.Content[0].Text)
	}

	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	if resp["status"] != "running" {
		t.Errorf("status = %v, want running", resp["status"])
	}
}

func TestToolSandboxCreateWithBadPolicyFile(t *testing.T) {
	s := newTestServer(t)
	args, _ := json.Marshal(createArgs{Name: "bad-policy", Policy: "/nonexistent/policy.yaml"})
	result := s.toolSandboxCreate(args)
	if !result.IsError {
		t.Error("expected error for nonexistent policy file")
	}
}

func TestToolSandboxTracesWithLimit(t *testing.T) {
	s := newTestServer(t)

	// Create and start a sandbox
	args, _ := json.Marshal(createArgs{Name: "traces-limit-test"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create: %s", result.Content[0].Text)
	}
	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	sandboxID := resp["sandbox_id"].(string)

	// Get traces with limit=1
	tracesArg, _ := json.Marshal(tracesArgs{SandboxID: sandboxID, Limit: 1})
	result = s.toolSandboxTraces(tracesArg)
	if result.IsError {
		t.Fatalf("traces: %s", result.Content[0].Text)
	}
	var tracesResp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &tracesResp)
	events, ok := tracesResp["events"].([]any)
	if !ok {
		t.Fatal("missing events array")
	}
	if len(events) > 1 {
		t.Errorf("expected at most 1 event with limit=1, got %d", len(events))
	}
}

// --- Invalid JSON args tests for coverage ---

func TestToolSandboxExecInvalidArgs(t *testing.T) {
	s := newTestServer(t)
	result := s.toolSandboxExec(json.RawMessage(`not json`))
	if !result.IsError {
		t.Error("expected error for invalid JSON args")
	}
	if !strings.Contains(result.Content[0].Text, "Invalid arguments") {
		t.Errorf("expected 'Invalid arguments' in error, got: %s", result.Content[0].Text)
	}
}

func TestToolSandboxStopInvalidArgs(t *testing.T) {
	s := newTestServer(t)
	result := s.toolSandboxStop(json.RawMessage(`not json`))
	if !result.IsError {
		t.Error("expected error for invalid JSON args")
	}
	if !strings.Contains(result.Content[0].Text, "Invalid arguments") {
		t.Errorf("expected 'Invalid arguments' in error, got: %s", result.Content[0].Text)
	}
}

func TestToolSandboxTracesInvalidArgs(t *testing.T) {
	s := newTestServer(t)
	result := s.toolSandboxTraces(json.RawMessage(`not json`))
	if !result.IsError {
		t.Error("expected error for invalid JSON args")
	}
	if !strings.Contains(result.Content[0].Text, "Invalid arguments") {
		t.Errorf("expected 'Invalid arguments' in error, got: %s", result.Content[0].Text)
	}
}

func TestHandleToolsCallNilParams(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`10`),
		Method:  "tools/call",
		// Params is nil
	}
	resp := s.handleRequest(req)
	if resp.Error == nil {
		t.Fatal("expected error for nil params on tools/call")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleToolsCallEmptyParams(t *testing.T) {
	s := newTestServer(t)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`11`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{}`),
	}
	resp := s.handleRequest(req)
	// Empty params means name="" which routes to default "Unknown tool" result
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(mcpToolCallResult)
	if !ok {
		t.Fatalf("expected mcpToolCallResult, got %T", resp.Result)
	}
	if !result.IsError {
		t.Error("expected tool error for empty tool name")
	}
}

func TestToolSandboxPolicyCheckWithServerPolicy(t *testing.T) {
	dir := t.TempDir()
	policyFile := filepath.Join(dir, "policy.yaml")
	os.WriteFile(policyFile, []byte(`name: check-policy
version: "1.0"
default_effect: deny
rules:
  - id: allow-read
    name: "Allow reads"
    actions: ["file:read"]
    resources: ["**"]
    effect: allow
    priority: 10
`), 0644)

	s := NewMCPServer(policyFile)
	args, _ := json.Marshal(policyCheckArgs{
		ActionType: "file.read",
		Resource:   "/tmp/test.txt",
	})
	result := s.toolSandboxPolicyCheck(args)
	if result.IsError {
		t.Fatalf("policy check error: %s", result.Content[0].Text)
	}
	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	// Verify the policy check returns a result with an effect field
	if resp["effect"] == nil {
		t.Error("expected effect in policy check response")
	}
}

func TestToolSandboxPolicyCheckBadPolicyFile(t *testing.T) {
	s := NewMCPServer("/nonexistent/policy.yaml")
	args, _ := json.Marshal(policyCheckArgs{
		ActionType: "file.read",
		Resource:   "/tmp/test.txt",
	})
	result := s.toolSandboxPolicyCheck(args)
	if !result.IsError {
		t.Error("expected error for nonexistent policy file")
	}
}

func TestWriteResponse(t *testing.T) {
	var buf strings.Builder
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Result:  map[string]string{"status": "ok"},
	}
	writeResponse(&buf, resp)
	output := buf.String()
	if !strings.Contains(output, `"status":"ok"`) {
		t.Errorf("unexpected writeResponse output: %s", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Error("writeResponse should end with newline")
	}
}

func TestToolSandboxExecWithResourceParams(t *testing.T) {
	s := newTestServer(t)

	// Create a sandbox first
	args, _ := json.Marshal(createArgs{Name: "resource-test"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create: %s", result.Content[0].Text)
	}
	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	sandboxID := resp["sandbox_id"].(string)

	// Test with url param
	execArg, _ := json.Marshal(execArgs{
		SandboxID:  sandboxID,
		ActionType: "net.http",
		Params:     map[string]string{"url": "http://example.com"},
	})
	result = s.toolSandboxExec(execArg)
	// Expect denial or execution, not an error in parsing
	if result.IsError {
		t.Fatalf("exec returned unexpected error: %s", result.Content[0].Text)
	}

	// Test with host param
	execArg, _ = json.Marshal(execArgs{
		SandboxID:  sandboxID,
		ActionType: "net.connect",
		Params:     map[string]string{"host": "localhost", "port": "80"},
	})
	result = s.toolSandboxExec(execArg)
	if result.IsError {
		t.Fatalf("exec returned unexpected error: %s", result.Content[0].Text)
	}

	// Test with command param
	execArg, _ = json.Marshal(execArgs{
		SandboxID:  sandboxID,
		ActionType: "shell.exec",
		Params:     map[string]string{"command": "echo test"},
	})
	result = s.toolSandboxExec(execArg)
	if result.IsError {
		t.Fatalf("exec returned unexpected error: %s", result.Content[0].Text)
	}
}

// Test tools/call routing through handleRequest for all tool names
func TestHandleToolsCallAllTools(t *testing.T) {
	s := newTestServer(t)

	// Create a sandbox for tools that need one
	args, _ := json.Marshal(createArgs{Name: "route-test"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create: %s", result.Content[0].Text)
	}
	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	sandboxID := resp["sandbox_id"].(string)

	tests := []struct {
		name    string
		tool    string
		args    any
		isError bool
	}{
		{
			"sandbox_create via handleRequest",
			"sandbox_create",
			createArgs{Name: "via-request"},
			false,
		},
		{
			"sandbox_exec via handleRequest",
			"sandbox_exec",
			execArgs{SandboxID: sandboxID, ActionType: "file.read", Params: map[string]string{"path": "x.txt"}},
			false, // denied by policy is not IsError
		},
		{
			"sandbox_stop via handleRequest",
			"sandbox_stop",
			stopArgs{SandboxID: sandboxID},
			false,
		},
		{
			"sandbox_traces via handleRequest",
			"sandbox_traces",
			tracesArgs{SandboxID: "nonexistent"},
			true,
		},
		{
			"sandbox_policy_check via handleRequest",
			"sandbox_policy_check",
			policyCheckArgs{ActionType: "file.read", Resource: "/tmp/x"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsJSON, _ := json.Marshal(tt.args)
			params, _ := json.Marshal(mcpToolCallParams{
				Name:      tt.tool,
				Arguments: argsJSON,
			})
			req := jsonRPCRequest{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`99`),
				Method:  "tools/call",
				Params:  params,
			}
			resp := s.handleRequest(req)
			if resp.Error != nil {
				t.Fatalf("unexpected JSON-RPC error: %s", resp.Error.Message)
			}
			result, ok := resp.Result.(mcpToolCallResult)
			if !ok {
				t.Fatalf("expected mcpToolCallResult, got %T", resp.Result)
			}
			if tt.isError && !result.IsError {
				t.Error("expected tool error")
			}
		})
	}
}

func TestToolSandboxTracesZeroLimit(t *testing.T) {
	s := newTestServer(t)

	args, _ := json.Marshal(createArgs{Name: "traces-zero"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create: %s", result.Content[0].Text)
	}
	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	sandboxID := resp["sandbox_id"].(string)

	// Limit 0 should default to 50
	tracesArg, _ := json.Marshal(tracesArgs{SandboxID: sandboxID, Limit: 0})
	result = s.toolSandboxTraces(tracesArg)
	if result.IsError {
		t.Fatalf("traces: %s", result.Content[0].Text)
	}
}

func TestToolSandboxTracesNegativeLimit(t *testing.T) {
	s := newTestServer(t)

	args, _ := json.Marshal(createArgs{Name: "traces-neg"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create: %s", result.Content[0].Text)
	}
	var resp map[string]any
	json.Unmarshal([]byte(result.Content[0].Text), &resp)
	sandboxID := resp["sandbox_id"].(string)

	// Negative limit should default to 50
	tracesArg, _ := json.Marshal(tracesArgs{SandboxID: sandboxID, Limit: -1})
	result = s.toolSandboxTraces(tracesArg)
	if result.IsError {
		t.Fatalf("traces: %s", result.Content[0].Text)
	}
}

func TestRunWithStdinPipe(t *testing.T) {
	s := newTestServer(t)

	// Save original stdin/stdout
	origStdin := os.Stdin
	origStdout := os.Stdout

	// Create a pipe for stdin
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	// Create a pipe for stdout
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdin = stdinR
	os.Stdout = stdoutW

	// Run server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run()
	}()

	// Send an initialize request
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	stdinW.WriteString(initReq)

	// Send a ping request
	pingReq := `{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n"
	stdinW.WriteString(pingReq)

	// Send invalid JSON to cover parse error path
	stdinW.WriteString("not valid json\n")

	// Close stdin to trigger EOF
	stdinW.Close()

	// Wait for Run() to return
	runErr := <-errCh

	// Restore
	os.Stdin = origStdin
	os.Stdout = origStdout
	stdoutW.Close()

	if runErr != nil {
		t.Errorf("Run() returned error: %v", runErr)
	}

	// Read all output
	outBytes, _ := io.ReadAll(stdoutR)
	output := string(outBytes)

	// Should have multiple JSON responses
	if !strings.Contains(output, "2.0") {
		t.Errorf("expected JSON-RPC responses in output, got: %s", output)
	}
}

func TestShutdown(t *testing.T) {
	s := newTestServer(t)

	// Create a sandbox
	args, _ := json.Marshal(createArgs{Name: "shutdown-test"})
	result := s.toolSandboxCreate(args)
	if result.IsError {
		t.Fatalf("create: %s", result.Content[0].Text)
	}

	s.mu.Lock()
	count := len(s.sandboxes)
	s.mu.Unlock()
	if count == 0 {
		t.Fatal("expected at least 1 sandbox before shutdown")
	}

	s.shutdown()

	s.mu.Lock()
	count = len(s.sandboxes)
	s.mu.Unlock()
	if count != 0 {
		t.Errorf("expected 0 sandboxes after shutdown, got %d", count)
	}
}
