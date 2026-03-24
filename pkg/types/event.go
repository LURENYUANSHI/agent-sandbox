package types

import "time"

// EventType represents the type of trace event.
type EventType string

const (
	EventTypeAction         EventType = "action"
	EventTypePolicyDecision EventType = "policy_decision"
	EventTypeSpanStart      EventType = "span_start"
	EventTypeSpanEnd        EventType = "span_end"
	EventTypeError          EventType = "error"
	EventTypeInfo           EventType = "info"
)

// TraceEvent represents a single recorded event in a trace.
type TraceEvent struct {
	ID             string            `json:"id"`
	SandboxID      string            `json:"sandbox_id"`
	ParentID       string            `json:"parent_id,omitempty"`
	EventType      EventType         `json:"event_type"`
	Action         *Action           `json:"action,omitempty"`
	Result         *ActionResult     `json:"result,omitempty"`
	PolicyDecision *PolicyDecision   `json:"policy_decision,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
	DurationNs     int64             `json:"duration_ns,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
}

// PolicyDecision records a policy engine's decision about an action.
type PolicyDecision struct {
	Allowed bool   `json:"allowed"`
	Rule    string `json:"rule,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// Trace represents a complete execution trace for a sandbox.
type Trace struct {
	SandboxID string        `json:"sandbox_id"`
	Events    []*TraceEvent `json:"events"`
	RootSpans []*SpanNode   `json:"root_spans,omitempty"`
}

// SpanNode represents a node in a span tree (parent-child hierarchy).
type SpanNode struct {
	Event    *TraceEvent `json:"event"`
	Children []*SpanNode `json:"children,omitempty"`
}

// TimelineEntry is a flat chronological entry for display.
type TimelineEntry struct {
	Event *TraceEvent `json:"event"`
	Depth int         `json:"depth"`
}

// TraceStore defines the interface for trace persistence.
type TraceStore interface {
	SaveEvent(event *TraceEvent) error
	GetEvent(id string) (*TraceEvent, error)
	ListEvents(sandboxID string) ([]*TraceEvent, error)
	QueryEvents(query EventQuery) ([]*TraceEvent, error)
	DeleteEvents(sandboxID string) error
	Close() error
}

// EventQuery defines filters for querying trace events.
type EventQuery struct {
	SandboxID string    `json:"sandbox_id,omitempty"`
	EventType EventType `json:"event_type,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	ParentID  string    `json:"parent_id,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}
