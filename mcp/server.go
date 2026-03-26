package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
	"github.com/google/uuid"
)

// JSON-RPC 2.0 message types.

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP protocol types.

type mcpInitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    mcpCapabilities   `json:"capabilities"`
	ServerInfo      mcpServerInfo     `json:"serverInfo"`
}

type mcpCapabilities struct {
	Tools *mcpToolCapability `json:"tools,omitempty"`
}

type mcpToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpToolDef `json:"tools"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type mcpToolCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Sandbox instance tracked by the server.

type managedSandbox struct {
	instance *sandbox.Instance
	exec     *executor.Executor
	recorder *trace.Recorder
	config   sandbox.Config
}

// MCPServer implements the MCP protocol over stdio.
type MCPServer struct {
	mu         sync.Mutex
	sandboxes  map[string]*managedSandbox
	appConfig  *config.AppConfig
	policyFile string
	logger     *log.Logger
}

// NewMCPServer creates a new MCP server.
func NewMCPServer(policyFile string) *MCPServer {
	return &MCPServer{
		sandboxes:  make(map[string]*managedSandbox),
		appConfig:  config.LoadFromEnv(),
		policyFile: policyFile,
		logger:     log.New(os.Stderr, "[mcp] ", log.LstdFlags),
	}
}

// Run reads JSON-RPC requests from stdin and writes responses to stdout.
func (s *MCPServer) Run() error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	s.logger.Println("AgentSandbox MCP server started")

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				s.logger.Println("stdin closed, shutting down")
				s.shutdown()
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				Error: &jsonRPCError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			}
			writeResponse(writer, resp)
			continue
		}

		resp := s.handleRequest(req)
		writeResponse(writer, resp)
	}
}

func writeResponse(w io.Writer, resp jsonRPCResponse) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	w.Write(data)
}

func (s *MCPServer) handleRequest(req jsonRPCRequest) jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed but we still send one if ID is present.
		if req.ID == nil {
			return jsonRPCResponse{}
		}
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jsonRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *MCPServer) handleInitialize(req jsonRPCRequest) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: mcpInitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: mcpCapabilities{
				Tools: &mcpToolCapability{},
			},
			ServerInfo: mcpServerInfo{
				Name:    "agent-sandbox",
				Version: "1.0.0",
			},
		},
	}
}

func (s *MCPServer) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
	tools := []mcpToolDef{
		{
			Name:        "sandbox_create",
			Description: "Create and start a new isolated sandbox for executing agent operations with policy enforcement.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Human-readable name for the sandbox.",
					},
					"policy": map[string]any{
						"type":        "string",
						"description": "Path to a YAML policy file. Uses the server default if omitted.",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "sandbox_exec",
			Description: "Execute an action inside a sandbox with policy enforcement. Supports file read/write/delete, network HTTP/TCP, process exec, and shell commands.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sandbox_id": map[string]any{
						"type":        "string",
						"description": "ID of the sandbox to execute in.",
					},
					"action_type": map[string]any{
						"type":        "string",
						"description": "Action type: file.read, file.write, file.delete, net.http, net.connect, process.exec, shell.exec",
						"enum":        []string{"file.read", "file.write", "file.delete", "net.http", "net.connect", "process.exec", "shell.exec"},
					},
					"params": map[string]any{
						"type":        "object",
						"description": "Action parameters. For file ops: {path, content}. For net.http: {url, method}. For process/shell: {command, args}.",
						"additionalProperties": map[string]any{"type": "string"},
					},
				},
				"required": []string{"sandbox_id", "action_type", "params"},
			},
		},
		{
			Name:        "sandbox_stop",
			Description: "Stop a running sandbox and release its resources.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sandbox_id": map[string]any{
						"type":        "string",
						"description": "ID of the sandbox to stop.",
					},
				},
				"required": []string{"sandbox_id"},
			},
		},
		{
			Name:        "sandbox_traces",
			Description: "Retrieve execution trace events for a sandbox, showing the full audit trail of actions, policy decisions, and results.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sandbox_id": map[string]any{
						"type":        "string",
						"description": "ID of the sandbox to get traces for.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of events to return. Defaults to 50.",
					},
				},
				"required": []string{"sandbox_id"},
			},
		},
		{
			Name:        "sandbox_policy_check",
			Description: "Pre-check whether an action would be allowed or denied by the current policy, without executing it.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action_type": map[string]any{
						"type":        "string",
						"description": "Action type to check: file.read, file.write, file.delete, net.http, net.connect, process.exec, shell.exec",
					},
					"resource": map[string]any{
						"type":        "string",
						"description": "Resource path or target to check against policy rules.",
					},
				},
				"required": []string{"action_type", "resource"},
			},
		},
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  mcpToolsListResult{Tools: tools},
	}
}

func (s *MCPServer) handleToolsCall(req jsonRPCRequest) jsonRPCResponse {
	var params mcpToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jsonRPCError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	var result mcpToolCallResult

	switch params.Name {
	case "sandbox_create":
		result = s.toolSandboxCreate(params.Arguments)
	case "sandbox_exec":
		result = s.toolSandboxExec(params.Arguments)
	case "sandbox_stop":
		result = s.toolSandboxStop(params.Arguments)
	case "sandbox_traces":
		result = s.toolSandboxTraces(params.Arguments)
	case "sandbox_policy_check":
		result = s.toolSandboxPolicyCheck(params.Arguments)
	default:
		result = errorResult(fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// Tool implementations.

type createArgs struct {
	Name   string `json:"name"`
	Policy string `json:"policy"`
}

func (s *MCPServer) toolSandboxCreate(args json.RawMessage) mcpToolCallResult {
	var a createArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %s", err))
	}
	if a.Name == "" {
		return errorResult("name is required")
	}

	// Set up policy engine.
	engine := policy.NewEngineWithConfig(s.appConfig.Policy)
	policyFile := s.policyFile
	if a.Policy != "" {
		policyFile = a.Policy
	}
	if policyFile != "" {
		p, err := policy.LoadFromFile(policyFile)
		if err != nil {
			return errorResult(fmt.Sprintf("Failed to load policy: %s", err))
		}
		engine.LoadPolicy(*p)
	}

	// Create trace recorder.
	id := uuid.New().String()
	tracePath := filepath.Join(os.TempDir(), "agent-sandbox-mcp", id+".db")
	os.MkdirAll(filepath.Dir(tracePath), 0o755)

	recorder, err := trace.NewRecorder(tracePath)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to create trace recorder: %s", err))
	}

	// Create sandbox config.
	rootDir := filepath.Join(os.TempDir(), "agent-sandbox-mcp", "roots", id)
	sbxCfg := sandbox.DefaultConfig()
	sbxCfg.ID = id
	sbxCfg.Name = a.Name
	sbxCfg.RootDir = rootDir
	sbxCfg.TracePath = tracePath

	if s.appConfig.Sandbox.DefaultTimeoutSec > 0 {
		sbxCfg.TimeoutSeconds = s.appConfig.Sandbox.DefaultTimeoutSec
	}

	// Create and start sandbox.
	instance, err := sandbox.NewSandbox(sbxCfg, engine, recorder)
	if err != nil {
		recorder.Close()
		return errorResult(fmt.Sprintf("Failed to create sandbox: %s", err))
	}

	if err := instance.Start(context.Background()); err != nil {
		recorder.Close()
		return errorResult(fmt.Sprintf("Failed to start sandbox: %s", err))
	}

	exec := executor.NewExecutor(sbxCfg, s.appConfig.Executor)

	s.mu.Lock()
	s.sandboxes[id] = &managedSandbox{
		instance: instance,
		exec:     exec,
		recorder: recorder,
		config:   sbxCfg,
	}
	s.mu.Unlock()

	s.logger.Printf("Created sandbox %s (%s)", id, a.Name)

	return jsonResult(map[string]any{
		"sandbox_id": id,
		"name":       a.Name,
		"status":     "running",
		"root_dir":   rootDir,
	})
}

type execArgs struct {
	SandboxID  string            `json:"sandbox_id"`
	ActionType string            `json:"action_type"`
	Params     map[string]string `json:"params"`
}

func (s *MCPServer) toolSandboxExec(args json.RawMessage) mcpToolCallResult {
	var a execArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %s", err))
	}

	s.mu.Lock()
	ms, ok := s.sandboxes[a.SandboxID]
	s.mu.Unlock()
	if !ok {
		return errorResult(fmt.Sprintf("Sandbox not found: %s", a.SandboxID))
	}

	action := types.Action{
		ID:        uuid.New().String(),
		Type:      types.ActionType(a.ActionType),
		Params:    a.Params,
		Timestamp: time.Now(),
	}

	// Set resource from params for policy matching.
	if path, ok := a.Params["path"]; ok {
		action.Resource = path
	} else if url, ok := a.Params["url"]; ok {
		action.Resource = url
	} else if host, ok := a.Params["host"]; ok {
		action.Resource = host
	} else if cmd, ok := a.Params["command"]; ok {
		action.Resource = cmd
	}

	result, err := ms.instance.Execute(context.Background(), action, ms.exec.Execute)
	if err != nil {
		return jsonResult(map[string]any{
			"success":         false,
			"error":           err.Error(),
			"policy_decision": "denied",
		})
	}

	return jsonResult(map[string]any{
		"success":   result.Success,
		"output":    result.Output,
		"error":     result.Error,
		"exit_code": result.ExitCode,
	})
}

type stopArgs struct {
	SandboxID string `json:"sandbox_id"`
}

func (s *MCPServer) toolSandboxStop(args json.RawMessage) mcpToolCallResult {
	var a stopArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %s", err))
	}

	s.mu.Lock()
	ms, ok := s.sandboxes[a.SandboxID]
	s.mu.Unlock()
	if !ok {
		return errorResult(fmt.Sprintf("Sandbox not found: %s", a.SandboxID))
	}

	if err := ms.instance.Stop(context.Background()); err != nil {
		return errorResult(fmt.Sprintf("Failed to stop sandbox: %s", err))
	}

	ms.recorder.Close()

	s.mu.Lock()
	delete(s.sandboxes, a.SandboxID)
	s.mu.Unlock()

	s.logger.Printf("Stopped sandbox %s", a.SandboxID)

	return jsonResult(map[string]any{
		"sandbox_id": a.SandboxID,
		"status":     "stopped",
	})
}

type tracesArgs struct {
	SandboxID string `json:"sandbox_id"`
	Limit     int    `json:"limit"`
}

func (s *MCPServer) toolSandboxTraces(args json.RawMessage) mcpToolCallResult {
	var a tracesArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %s", err))
	}

	s.mu.Lock()
	ms, ok := s.sandboxes[a.SandboxID]
	s.mu.Unlock()
	if !ok {
		return errorResult(fmt.Sprintf("Sandbox not found: %s", a.SandboxID))
	}

	events, err := ms.instance.GetTraces()
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to get traces: %s", err))
	}

	limit := a.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > len(events) {
		limit = len(events)
	}

	// Return the most recent events.
	start := 0
	if len(events) > limit {
		start = len(events) - limit
	}
	events = events[start:]

	type eventSummary struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		ActionID  string `json:"action_id,omitempty"`
		Timestamp string `json:"timestamp"`
		Data      any    `json:"data,omitempty"`
	}

	summaries := make([]eventSummary, len(events))
	for i, e := range events {
		summaries[i] = eventSummary{
			ID:        e.ID,
			Type:      string(e.Type),
			ActionID:  e.ActionID,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Data:      e.Data,
		}
	}

	return jsonResult(map[string]any{
		"sandbox_id":  a.SandboxID,
		"event_count": len(summaries),
		"events":      summaries,
	})
}

type policyCheckArgs struct {
	ActionType string `json:"action_type"`
	Resource   string `json:"resource"`
}

func (s *MCPServer) toolSandboxPolicyCheck(args json.RawMessage) mcpToolCallResult {
	var a policyCheckArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %s", err))
	}

	// Build a policy engine with the server's default policy.
	engine := policy.NewEngineWithConfig(s.appConfig.Policy)
	if s.policyFile != "" {
		p, err := policy.LoadFromFile(s.policyFile)
		if err != nil {
			return errorResult(fmt.Sprintf("Failed to load policy: %s", err))
		}
		engine.LoadPolicy(*p)
	}

	action := types.Action{
		ID:       uuid.New().String(),
		Type:     types.ActionType(a.ActionType),
		Resource: a.Resource,
		Params:   map[string]string{"path": a.Resource},
	}

	decision := engine.Evaluate(action)

	return jsonResult(map[string]any{
		"effect":  string(decision.Effect),
		"allowed": decision.Allowed,
		"rule":    decision.Rule,
		"reason":  decision.Reason,
	})
}

// Helpers.

func jsonResult(v any) mcpToolCallResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	return mcpToolCallResult{
		Content: []mcpContent{{Type: "text", Text: string(data)}},
	}
}

func errorResult(msg string) mcpToolCallResult {
	data, _ := json.MarshalIndent(map[string]string{"error": msg}, "", "  ")
	return mcpToolCallResult{
		Content: []mcpContent{{Type: "text", Text: string(data)}},
		IsError: true,
	}
}

func (s *MCPServer) shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, ms := range s.sandboxes {
		ms.instance.Stop(context.Background())
		ms.recorder.Close()
		s.logger.Printf("Stopped sandbox %s during shutdown", id)
	}
	s.sandboxes = make(map[string]*managedSandbox)
}
