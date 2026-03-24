package trace

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Recorder records trace events to a store. It is safe for concurrent use.
type Recorder struct {
	store types.TraceStore
	mu    sync.Mutex
}

// NewRecorder creates a new Recorder that persists events to the given store.
func NewRecorder(store types.TraceStore) *Recorder {
	return &Recorder{store: store}
}

// RecordEvent saves an event to the store, auto-generating an ID and timestamp
// if not already set.
func (r *Recorder) RecordEvent(event *types.TraceEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.ID == "" {
		event.ID = generateID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return r.store.SaveEvent(event)
}

// SpanContext holds the state of an in-progress span.
type SpanContext struct {
	Event     *types.TraceEvent
	StartTime time.Time
}

// StartSpan creates a new span event for an action execution.
// Returns a SpanContext that must be passed to EndSpan when the action completes.
func (r *Recorder) StartSpan(sandboxID string, action *types.Action) (*SpanContext, error) {
	now := time.Now()
	event := &types.TraceEvent{
		ID:        generateID(),
		SandboxID: sandboxID,
		EventType: types.EventTypeSpanStart,
		Action:    action,
		Timestamp: now,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.store.SaveEvent(event); err != nil {
		return nil, fmt.Errorf("saving span start: %w", err)
	}

	return &SpanContext{Event: event, StartTime: now}, nil
}

// StartChildSpan creates a nested span under a parent span.
func (r *Recorder) StartChildSpan(parentCtx *SpanContext, sandboxID string, action *types.Action) (*SpanContext, error) {
	now := time.Now()
	event := &types.TraceEvent{
		ID:        generateID(),
		SandboxID: sandboxID,
		ParentID:  parentCtx.Event.ID,
		EventType: types.EventTypeSpanStart,
		Action:    action,
		Timestamp: now,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.store.SaveEvent(event); err != nil {
		return nil, fmt.Errorf("saving child span start: %w", err)
	}

	return &SpanContext{Event: event, StartTime: now}, nil
}

// EndSpan closes a span, calculating duration and recording the result.
func (r *Recorder) EndSpan(ctx *SpanContext, result *types.ActionResult) error {
	now := time.Now()
	duration := now.Sub(ctx.StartTime)

	event := &types.TraceEvent{
		ID:         generateID(),
		SandboxID:  ctx.Event.SandboxID,
		ParentID:   ctx.Event.ID,
		EventType:  types.EventTypeSpanEnd,
		Action:     ctx.Event.Action,
		Result:     result,
		Timestamp:  now,
		DurationNs: duration.Nanoseconds(),
		Attributes: ctx.Event.Attributes,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.store.SaveEvent(event)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
