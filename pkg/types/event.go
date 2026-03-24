package types

import "time"

// EventType represents the kind of trace event recorded during sandbox operation.
// Events track the full lifecycle of an action from request through policy
// evaluation to execution or denial.
type EventType string

const (
	// EventActionRequested is emitted when an agent submits an action for execution.
	EventActionRequested EventType = "action.requested"
	// EventPolicyEvaluated is emitted after the policy engine evaluates an action.
	EventPolicyEvaluated EventType = "policy.evaluated"
	// EventActionExecuted is emitted when an action completes successfully.
	EventActionExecuted EventType = "action.executed"
	// EventActionDenied is emitted when a policy denies an action.
	EventActionDenied EventType = "action.denied"
	// EventActionFailed is emitted when an allowed action fails during execution.
	EventActionFailed EventType = "action.failed"
	// EventSandboxCreated is emitted when a new sandbox is initialized.
	EventSandboxCreated EventType = "sandbox.created"
	// EventSandboxStopped is emitted when a sandbox is stopped.
	EventSandboxStopped EventType = "sandbox.stopped"
)

// TraceEvent represents a single event in the sandbox execution trace.
// Events form a tree via ParentID, allowing reconstruction of the full
// action lifecycle: request -> policy evaluation -> execution/denial.
type TraceEvent struct {
	// ID is the unique identifier for this event.
	ID string `json:"id"`
	// TraceID groups related events into a single trace (OpenTelemetry compatible).
	TraceID string `json:"trace_id"`
	// SpanID identifies this specific span within the trace.
	SpanID string `json:"span_id"`
	// SandboxID identifies which sandbox generated this event.
	SandboxID string `json:"sandbox_id"`
	// ParentID links this event to a parent event, forming a trace tree.
	ParentID string `json:"parent_id,omitempty"`
	// Type categorizes the event.
	Type EventType `json:"type"`
	// Action contains the action that triggered this event, if applicable.
	Action *Action `json:"action,omitempty"`
	// Result contains the outcome of action execution, if applicable.
	Result *ActionResult `json:"result,omitempty"`
	// PolicyDecision contains the policy evaluation result, if applicable.
	PolicyDecision *PolicyDecision `json:"policy_decision,omitempty"`
	// Timestamp records when the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// Duration records how long the event took to process.
	Duration time.Duration `json:"duration,omitempty"`
	// Attributes holds additional OpenTelemetry-compatible key-value pairs.
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
