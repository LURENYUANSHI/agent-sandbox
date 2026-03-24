package types

import "time"

// ActionType represents the category of an action.
type ActionType string

const (
	ActionTypeFile    ActionType = "file"
	ActionTypeNetwork ActionType = "network"
	ActionTypeProcess ActionType = "process"
	ActionTypeShell   ActionType = "shell"
)

// FileOp represents a filesystem operation.
type FileOp string

const (
	FileOpRead   FileOp = "read"
	FileOpWrite  FileOp = "write"
	FileOpDelete FileOp = "delete"
	FileOpList   FileOp = "list"
)

// Action represents an operation that an agent wants to perform.
type Action struct {
	ID        string     `json:"id"`
	Type      ActionType `json:"type"`
	Path      string     `json:"path,omitempty"`
	FileOp    FileOp     `json:"file_op,omitempty"`
	Content   string     `json:"content,omitempty"`
	Host      string     `json:"host,omitempty"`
	Port      int        `json:"port,omitempty"`
	Command   string     `json:"command,omitempty"`
	Args      []string   `json:"args,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}
