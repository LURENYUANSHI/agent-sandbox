package trace

import (
	"fmt"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Replayer loads and replays recorded traces.
type Replayer struct {
	store types.TraceStore
}

// NewReplayer creates a new Replayer backed by the given store.
func NewReplayer(store types.TraceStore) *Replayer {
	return &Replayer{store: store}
}

// ReplayState tracks the current position during trace replay.
type ReplayState struct {
	Trace   *types.Trace
	current int
}

// LoadTrace loads a full trace for a sandbox and builds the span tree.
func (r *Replayer) LoadTrace(sandboxID string) (*types.Trace, error) {
	events, err := r.store.ListEvents(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("loading events: %w", err)
	}

	trace := &types.Trace{
		SandboxID: sandboxID,
		Events:    events,
		RootSpans: buildSpanTree(events),
	}

	return trace, nil
}

// NewReplayState creates a replay state for stepping through a trace.
func NewReplayState(trace *types.Trace) *ReplayState {
	return &ReplayState{Trace: trace, current: 0}
}

// Step advances the replay by one event. Returns the event and true if there
// are more events, or nil and false when the trace is exhausted.
func Step(state *ReplayState) (*types.TraceEvent, bool) {
	if state.current >= len(state.Trace.Events) {
		return nil, false
	}
	event := state.Trace.Events[state.current]
	state.current++
	return event, state.current < len(state.Trace.Events)
}

// Rewind resets the replay position to the beginning.
func Rewind(state *ReplayState) {
	state.current = 0
}

// GetTimeline returns a flat chronological view of events with depth information
// based on parent-child nesting.
func (r *Replayer) GetTimeline(sandboxID string) ([]types.TimelineEntry, error) {
	events, err := r.store.ListEvents(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("loading events for timeline: %w", err)
	}

	// Build depth map from parent relationships.
	depthMap := map[string]int{}
	for _, e := range events {
		if e.ParentID == "" {
			depthMap[e.ID] = 0
		}
	}

	// Multi-pass to resolve depths (handles arbitrary nesting).
	changed := true
	for changed {
		changed = false
		for _, e := range events {
			if _, ok := depthMap[e.ID]; ok {
				continue
			}
			if parentDepth, ok := depthMap[e.ParentID]; ok {
				depthMap[e.ID] = parentDepth + 1
				changed = true
			}
		}
	}

	entries := make([]types.TimelineEntry, 0, len(events))
	for _, e := range events {
		entries = append(entries, types.TimelineEntry{
			Event: e,
			Depth: depthMap[e.ID],
		})
	}

	return entries, nil
}

// buildSpanTree constructs a tree of SpanNodes from a flat list of events.
func buildSpanTree(events []*types.TraceEvent) []*types.SpanNode {
	nodeMap := map[string]*types.SpanNode{}

	// Create nodes for all events.
	for _, e := range events {
		nodeMap[e.ID] = &types.SpanNode{Event: e}
	}

	var roots []*types.SpanNode
	for _, e := range events {
		node := nodeMap[e.ID]
		if e.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[e.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Orphan — treat as root.
			roots = append(roots, node)
		}
	}

	return roots
}
