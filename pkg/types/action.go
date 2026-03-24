package types

// ActionType represents the category of action being performed.
type ActionType string

const (
	ActionTypeFile    ActionType = "file"
	ActionTypeNetwork ActionType = "network"
	ActionTypeProcess ActionType = "process"
	ActionTypeShell   ActionType = "shell"
)

// Action represents an operation that an agent wants to perform.
type Action struct {
	Type    ActionType `json:"type"`
	Details any        `json:"details"`
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
	Method    string `json:"method,omitempty"` // GET, POST, etc.
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

// ActionResult holds the outcome of an executed action.
type ActionResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}
