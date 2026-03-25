package types

import "time"

// ActionType represents the category of action an AI agent is attempting to perform.
type ActionType string

// Fine-grained action types used by the policy engine.
const (
	ActionFileRead   ActionType = "file:read"
	ActionFileWrite  ActionType = "file:write"
	ActionFileDelete ActionType = "file:delete"
	ActionNetConnect ActionType = "net:connect"
	ActionNetListen  ActionType = "net:listen"
	ActionNetHTTP    ActionType = "net:http"
	ActionProcExec   ActionType = "proc:exec"
	ActionProcKill   ActionType = "proc:kill"
	ActionShellExec  ActionType = "shell:exec"
)

// Category-level action types used by the trace system.
const (
	ActionTypeFile    ActionType = "file"
	ActionTypeNetwork ActionType = "network"
	ActionTypeProcess ActionType = "process"
	ActionTypeShell   ActionType = "shell"
)

// Action represents a single operation that an AI agent requests to perform
// within the sandbox.
type Action struct {
	ID        string                 `json:"id"`
	Type      ActionType             `json:"type"`
	Resource  string                 `json:"resource,omitempty"`
	Params    map[string]string      `json:"params,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   any                    `json:"details,omitempty"`
}

// FileAction represents a filesystem operation.
type FileAction struct {
	Operation string `json:"operation"` // read, write, delete, list, mkdir
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
}

// NetworkAction represents a network operation.
type NetworkAction struct {
	Operation string `json:"operation"` // http, tcp, dns
	Host      string `json:"host"`
	Port      int    `json:"port,omitempty"`
	Method    string `json:"method,omitempty"`
	URL       string `json:"url,omitempty"`
}

// ProcessAction represents a process operation.
type ProcessAction struct {
	Operation string   `json:"operation"` // exec, kill, signal
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	PID       int      `json:"pid,omitempty"`
}

// ShellAction represents a shell command execution.
type ShellAction struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir,omitempty"`
}

// ActionResult captures the outcome of executing an action within the sandbox.
type ActionResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}
