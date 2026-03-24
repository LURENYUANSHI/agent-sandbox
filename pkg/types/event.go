package types

import "time"

// Decision represents whether an action was allowed or denied.
type Decision string

const (
	DecisionAllowed Decision = "allowed"
	DecisionDenied  Decision = "denied"
)

// TraceEvent records a single action execution within a sandbox.
type TraceEvent struct {
	ID         string   `json:"id"`
	TraceID    string   `json:"trace_id"`
	SandboxID  string   `json:"sandbox_id"`
	Action     Action   `json:"action"`
	Decision   Decision `json:"decision"`
	Reason     string   `json:"reason,omitempty"`
	Result     string   `json:"result,omitempty"`
	Error      string   `json:"error,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	DurationMs int64    `json:"duration_ms"`
}
