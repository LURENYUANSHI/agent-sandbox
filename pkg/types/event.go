package types

import "time"

// EventType represents the type of trace event.
type EventType string

const (
	EventActionRequested EventType = "action.requested"
	EventPolicyEvaluated EventType = "policy.evaluated"
	EventActionExecuted  EventType = "action.executed"
	EventActionFailed    EventType = "action.failed"
	EventActionDenied    EventType = "action.denied"
)

// TraceEvent represents a single event in a trace.
type TraceEvent struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"trace_id"`
	SpanID    string                 `json:"span_id"`
	ParentID  string                 `json:"parent_id,omitempty"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Action    *Action                `json:"action,omitempty"`
	Decision  *PolicyDecision        `json:"decision,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
