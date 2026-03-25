package types

import "time"

// ActionType represents the category of action being performed.
type ActionType string

const (
	ActionTypeFileRead   ActionType = "file.read"
	ActionTypeFileWrite  ActionType = "file.write"
	ActionTypeFileDelete ActionType = "file.delete"
	ActionTypeNetHTTP    ActionType = "net.http"
	ActionTypeNetConnect ActionType = "net.connect"
	ActionTypeProcess    ActionType = "process.exec"
	ActionTypeShell      ActionType = "shell.exec"
)

// Action represents a request to perform an operation within the sandbox.
type Action struct {
	ID        string            `json:"id"`
	Type      ActionType        `json:"type"`
	Params    map[string]string `json:"params"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ActionResult holds the outcome of an executed action.
type ActionResult struct {
	ActionID   string        `json:"action_id"`
	Success    bool          `json:"success"`
	Output     string        `json:"output,omitempty"`
	Error      string        `json:"error,omitempty"`
	ExitCode   int           `json:"exit_code,omitempty"`
	Duration   time.Duration `json:"duration"`
	BytesRead  int64         `json:"bytes_read,omitempty"`
	BytesWrite int64         `json:"bytes_written,omitempty"`
}
