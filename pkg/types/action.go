package types

// ActionType represents the type of action being performed.
type ActionType string

const (
	ActionFileRead   ActionType = "file:read"
	ActionFileWrite  ActionType = "file:write"
	ActionFileDelete ActionType = "file:delete"
	ActionNetHTTP    ActionType = "net:http"
	ActionNetListen  ActionType = "net:listen"
	ActionNetConnect ActionType = "net:connect"
	ActionProcExec   ActionType = "proc:exec"
	ActionProcKill   ActionType = "proc:kill"
	ActionShellExec  ActionType = "shell:exec"
)

// Action represents an action to be evaluated by the policy engine.
type Action struct {
	Type     ActionType             `json:"type"`
	Resource string                 `json:"resource"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
