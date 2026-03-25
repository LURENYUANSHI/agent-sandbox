package types

import (
	"context"
	"time"
)

// SandboxStatus represents the current lifecycle state of a sandbox.
type SandboxStatus string

const (
	// StatusCreated indicates the sandbox has been initialized but not started.
	StatusCreated SandboxStatus = "created"
	// StatusRunning indicates the sandbox is active and accepting actions.
	StatusRunning SandboxStatus = "running"
	// StatusStopped indicates the sandbox has been stopped.
	StatusStopped SandboxStatus = "stopped"
	// StatusError indicates the sandbox encountered a fatal error.
	StatusError SandboxStatus = "error"
)

// Sandbox defines the interface for managing a sandbox lifecycle.
// A sandbox provides an isolated execution environment for AI agent actions.
type Sandbox interface {
	// ID returns the unique identifier of this sandbox.
	ID() string
	// Start initializes the sandbox environment and begins accepting actions.
	Start(ctx context.Context) error
	// Execute runs an action within the sandbox boundaries, returning its result.
	Execute(ctx context.Context, action Action) (*ActionResult, error)
	// Stop shuts down the sandbox, releasing all resources.
	Stop(ctx context.Context) error
	// Status returns the current lifecycle state of the sandbox.
	Status() SandboxStatus
}

// PolicyEngine defines the interface for evaluating actions against security policies.
type PolicyEngine interface {
	// Evaluate checks an action against loaded policies and returns a decision.
	Evaluate(ctx context.Context, action Action) (*PolicyDecision, error)
	// LoadPolicy adds or replaces a policy in the engine.
	LoadPolicy(policy Policy) error
	// ListPolicies returns all currently loaded policies.
	ListPolicies() []Policy
}

// TraceRecorder defines the interface for recording execution traces.
// It provides span-based tracing compatible with OpenTelemetry concepts.
type TraceRecorder interface {
	// RecordEvent persists a single trace event.
	RecordEvent(event TraceEvent) error
	// StartSpan begins a new trace span for an action, returning a context
	// that must be passed to EndSpan when the action completes.
	StartSpan(sandboxID string, action Action) (SpanContext, error)
	// EndSpan completes a trace span with the action's result.
	EndSpan(ctx SpanContext, result *ActionResult) error
}

// TraceStore defines the interface for persistent trace event storage and retrieval.
type TraceStore interface {
	// Save persists a single trace event.
	Save(event TraceEvent) error
	// GetBySandbox retrieves all events for a given sandbox.
	GetBySandbox(sandboxID string) ([]TraceEvent, error)
	// GetByID retrieves a single event by its unique identifier.
	GetByID(eventID string) (*TraceEvent, error)
	// Query retrieves events matching the given filter criteria.
	Query(filter TraceFilter) ([]TraceEvent, error)
}

// SpanContext holds the state for an in-progress trace span.
type SpanContext struct {
	// TraceID is the identifier for the overall trace.
	TraceID string
	// SpanID is the identifier for this specific span.
	SpanID string
	// StartTime records when the span began.
	StartTime time.Time
}

// TraceFilter specifies criteria for querying trace events.
type TraceFilter struct {
	// SandboxID filters events by sandbox.
	SandboxID string
	// EventType filters events by type.
	EventType EventType
	// StartTime filters events occurring at or after this time.
	StartTime *time.Time
	// EndTime filters events occurring at or before this time.
	EndTime *time.Time
	// Limit caps the number of returned events.
	Limit int
}
