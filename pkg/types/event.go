package types

import "time"

// EventType represents trace event categories.
type EventType string

const (
	EventSandboxStarted  EventType = "sandbox.started"
	EventSandboxStopped  EventType = "sandbox.stopped"
	EventActionRequested EventType = "action.requested"
	EventPolicyEvaluated EventType = "policy.evaluated"
	EventActionAllowed   EventType = "action.allowed"
	EventActionDenied    EventType = "action.denied"
	EventActionExecuted  EventType = "action.executed"
	EventActionFailed    EventType = "action.failed"
)

// TraceEvent records a single event in the sandbox lifecycle.
type TraceEvent struct {
	ID        string            `json:"id"`
	SandboxID string            `json:"sandbox_id"`
	Type      EventType         `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	ActionID  string            `json:"action_id,omitempty"`
	Data      map[string]string `json:"data,omitempty"`
	Duration  time.Duration     `json:"duration,omitempty"`
}
