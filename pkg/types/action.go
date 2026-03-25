// Package types defines the core types, interfaces, and contracts shared across
// the AgentSandbox system. All packages depend on these types for consistent
// communication about actions, events, policies, and sandbox lifecycle.
package types

import "time"

// ActionType represents the category of action an AI agent is attempting to perform.
// Actions are namespaced as "category:operation" (e.g., "file:read", "net:connect").
type ActionType string

const (
	// ActionFileRead represents a file read operation.
	ActionFileRead ActionType = "file:read"
	// ActionFileWrite represents a file write operation.
	ActionFileWrite ActionType = "file:write"
	// ActionFileDelete represents a file delete operation.
	ActionFileDelete ActionType = "file:delete"
	// ActionNetConnect represents an outbound network connection.
	ActionNetConnect ActionType = "net:connect"
	// ActionNetListen represents binding to a network port.
	ActionNetListen ActionType = "net:listen"
	// ActionNetHTTP represents an HTTP request.
	ActionNetHTTP ActionType = "net:http"
	// ActionProcExec represents spawning a new process.
	ActionProcExec ActionType = "proc:exec"
	// ActionProcKill represents killing a running process.
	ActionProcKill ActionType = "proc:kill"
	// ActionShellExec represents executing a shell command.
	ActionShellExec ActionType = "shell:exec"
)

// Action represents a single operation that an AI agent requests to perform
// within the sandbox. Every agent action is captured as an Action before
// being evaluated by the policy engine and executed.
type Action struct {
	// ID is the unique identifier for this action.
	ID string `json:"id"`
	// Type categorizes the action (e.g., "file:read", "net:connect").
	Type ActionType `json:"type"`
	// Resource is the primary target of the action (e.g., file path, host, command).
	Resource string `json:"resource"`
	// Params contains operation-specific parameters.
	// For file operations: {"path": "/etc/passwd", "mode": "r"}
	// For network operations: {"host": "example.com", "port": "443"}
	// For process operations: {"command": "ls", "args": "-la"}
	Params map[string]string `json:"params,omitempty"`
	// Metadata holds agent-provided context about why the action is being performed.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Timestamp records when the action was requested.
	Timestamp time.Time `json:"timestamp"`
}

// ActionResult captures the outcome of executing an action within the sandbox.
type ActionResult struct {
	// Success indicates whether the action completed without error.
	Success bool `json:"success"`
	// Output contains any stdout or return data from the action.
	Output string `json:"output,omitempty"`
	// Error contains the error message if the action failed.
	Error string `json:"error,omitempty"`
}
