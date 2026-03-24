package trace

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("creating test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func testAction() *types.Action {
	return &types.Action{
		Type:    types.ActionTypeFile,
		Details: map[string]string{"operation": "read", "path": "/tmp/test.txt"},
	}
}

// --- SQLiteStore tests ---

func TestSQLiteStore_SaveAndGetEvent(t *testing.T) {
	store := newTestStore(t)

	event := &types.TraceEvent{
		ID:        "evt-001",
		SandboxID: "sb-001",
		EventType: types.EventTypeAction,
		Action:    testAction(),
		Result:    &types.ActionResult{Success: true, Output: "file contents"},
		Timestamp: time.Now().Truncate(time.Microsecond),
		Attributes: map[string]string{
			"agent": "test-agent",
		},
	}

	if err := store.SaveEvent(event); err != nil {
		t.Fatalf("SaveEvent: %v", err)
	}

	got, err := store.GetEvent("evt-001")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}

	if got.ID != event.ID {
		t.Errorf("ID = %q, want %q", got.ID, event.ID)
	}
	if got.SandboxID != event.SandboxID {
		t.Errorf("SandboxID = %q, want %q", got.SandboxID, event.SandboxID)
	}
	if got.EventType != event.EventType {
		t.Errorf("EventType = %q, want %q", got.EventType, event.EventType)
	}
	if got.Action == nil || got.Action.Type != types.ActionTypeFile {
		t.Error("Action not round-tripped correctly")
	}
	if got.Result == nil || !got.Result.Success || got.Result.Output != "file contents" {
		t.Error("Result not round-tripped correctly")
	}
	if got.Attributes["agent"] != "test-agent" {
		t.Errorf("Attributes[agent] = %q, want %q", got.Attributes["agent"], "test-agent")
	}
}

func TestSQLiteStore_GetEvent_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetEvent("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent event")
	}
}

func TestSQLiteStore_ListEvents(t *testing.T) {
	store := newTestStore(t)

	base := time.Now().Truncate(time.Microsecond)
	for i := range 3 {
		event := &types.TraceEvent{
			ID:        fmt.Sprintf("evt-%03d", i),
			SandboxID: "sb-001",
			EventType: types.EventTypeAction,
			Timestamp: base.Add(time.Duration(i) * time.Second),
		}
		if err := store.SaveEvent(event); err != nil {
			t.Fatalf("SaveEvent %d: %v", i, err)
		}
	}

	// Save event for different sandbox.
	if err := store.SaveEvent(&types.TraceEvent{
		ID:        "evt-other",
		SandboxID: "sb-002",
		EventType: types.EventTypeAction,
		Timestamp: base,
	}); err != nil {
		t.Fatalf("SaveEvent other: %v", err)
	}

	events, err := store.ListEvents("sb-001")
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	// Should be ordered by timestamp.
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Error("events not in chronological order")
		}
	}
}

func TestSQLiteStore_QueryEvents_ByType(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Microsecond)
	store.SaveEvent(&types.TraceEvent{ID: "e1", SandboxID: "sb-1", EventType: types.EventTypeAction, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e2", SandboxID: "sb-1", EventType: types.EventTypeError, Timestamp: now.Add(time.Second)})
	store.SaveEvent(&types.TraceEvent{ID: "e3", SandboxID: "sb-1", EventType: types.EventTypeAction, Timestamp: now.Add(2 * time.Second)})

	events, err := store.QueryEvents(types.EventQuery{
		SandboxID: "sb-1",
		EventType: types.EventTypeAction,
	})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
}

func TestSQLiteStore_QueryEvents_ByTimeRange(t *testing.T) {
	store := newTestStore(t)

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 5 {
		store.SaveEvent(&types.TraceEvent{
			ID:        fmt.Sprintf("e%d", i),
			SandboxID: "sb-1",
			EventType: types.EventTypeAction,
			Timestamp: base.Add(time.Duration(i) * time.Hour),
		})
	}

	events, err := store.QueryEvents(types.EventQuery{
		SandboxID: "sb-1",
		StartTime: base.Add(1 * time.Hour),
		EndTime:   base.Add(3 * time.Hour),
	})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (hours 1,2,3)", len(events))
	}
}

func TestSQLiteStore_QueryEvents_WithLimit(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Microsecond)
	for i := range 10 {
		store.SaveEvent(&types.TraceEvent{
			ID:        fmt.Sprintf("e%d", i),
			SandboxID: "sb-1",
			EventType: types.EventTypeAction,
			Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}

	events, err := store.QueryEvents(types.EventQuery{SandboxID: "sb-1", Limit: 3})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
}

func TestSQLiteStore_DeleteEvents(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Microsecond)
	store.SaveEvent(&types.TraceEvent{ID: "e1", SandboxID: "sb-1", EventType: types.EventTypeAction, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e2", SandboxID: "sb-1", EventType: types.EventTypeAction, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e3", SandboxID: "sb-2", EventType: types.EventTypeAction, Timestamp: now})

	if err := store.DeleteEvents("sb-1"); err != nil {
		t.Fatalf("DeleteEvents: %v", err)
	}

	events, _ := store.ListEvents("sb-1")
	if len(events) != 0 {
		t.Fatalf("got %d events after delete, want 0", len(events))
	}

	// Other sandbox unaffected.
	events, _ = store.ListEvents("sb-2")
	if len(events) != 1 {
		t.Fatalf("sb-2 events = %d, want 1", len(events))
	}
}

func TestSQLiteStore_PolicyDecisionRoundTrip(t *testing.T) {
	store := newTestStore(t)

	event := &types.TraceEvent{
		ID:        "evt-pd",
		SandboxID: "sb-1",
		EventType: types.EventTypePolicyDecision,
		PolicyDecision: &types.PolicyDecision{
			Allowed: false,
			Rule:    "no-delete-root",
			Reason:  "cannot delete root filesystem",
		},
		Timestamp: time.Now().Truncate(time.Microsecond),
	}

	store.SaveEvent(event)
	got, _ := store.GetEvent("evt-pd")

	if got.PolicyDecision == nil {
		t.Fatal("PolicyDecision is nil")
	}
	if got.PolicyDecision.Allowed {
		t.Error("expected Allowed=false")
	}
	if got.PolicyDecision.Rule != "no-delete-root" {
		t.Errorf("Rule = %q, want %q", got.PolicyDecision.Rule, "no-delete-root")
	}
}

// --- Recorder tests ---

func TestRecorder_RecordEvent_AutoID(t *testing.T) {
	store := newTestStore(t)
	rec := NewRecorder(store)

	event := &types.TraceEvent{
		SandboxID: "sb-1",
		EventType: types.EventTypeInfo,
	}

	if err := rec.RecordEvent(event); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	if event.ID == "" {
		t.Error("expected auto-generated ID")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected auto-generated timestamp")
	}

	// Verify it's persisted.
	got, err := store.GetEvent(event.ID)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got.SandboxID != "sb-1" {
		t.Errorf("SandboxID = %q, want %q", got.SandboxID, "sb-1")
	}
}

func TestRecorder_SpanLifecycle(t *testing.T) {
	store := newTestStore(t)
	rec := NewRecorder(store)

	action := testAction()
	ctx, err := rec.StartSpan("sb-1", action)
	if err != nil {
		t.Fatalf("StartSpan: %v", err)
	}

	if ctx.Event.ID == "" {
		t.Error("span context should have an event ID")
	}

	// Simulate some work.
	time.Sleep(5 * time.Millisecond)

	result := &types.ActionResult{Success: true, Output: "done"}
	if err := rec.EndSpan(ctx, result); err != nil {
		t.Fatalf("EndSpan: %v", err)
	}

	// Should have 2 events: span_start and span_end.
	events, _ := store.ListEvents("sb-1")
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	startEvent := events[0]
	endEvent := events[1]

	if startEvent.EventType != types.EventTypeSpanStart {
		t.Errorf("first event type = %q, want span_start", startEvent.EventType)
	}
	if endEvent.EventType != types.EventTypeSpanEnd {
		t.Errorf("second event type = %q, want span_end", endEvent.EventType)
	}
	if endEvent.DurationNs <= 0 {
		t.Errorf("duration = %d, want > 0", endEvent.DurationNs)
	}
	if endEvent.Result == nil || !endEvent.Result.Success {
		t.Error("end span should have the result")
	}
}

func TestRecorder_NestedSpans(t *testing.T) {
	store := newTestStore(t)
	rec := NewRecorder(store)

	parentCtx, _ := rec.StartSpan("sb-1", testAction())

	childAction := &types.Action{Type: types.ActionTypeNetwork}
	childCtx, err := rec.StartChildSpan(parentCtx, "sb-1", childAction)
	if err != nil {
		t.Fatalf("StartChildSpan: %v", err)
	}

	if childCtx.Event.ParentID != parentCtx.Event.ID {
		t.Errorf("child ParentID = %q, want %q", childCtx.Event.ParentID, parentCtx.Event.ID)
	}

	rec.EndSpan(childCtx, &types.ActionResult{Success: true})
	rec.EndSpan(parentCtx, &types.ActionResult{Success: true})

	events, _ := store.ListEvents("sb-1")
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4 (2 starts + 2 ends)", len(events))
	}
}

func TestRecorder_ConcurrentRecording(t *testing.T) {
	store := newTestStore(t)
	rec := NewRecorder(store)

	done := make(chan error, 20)
	for i := range 20 {
		go func(idx int) {
			event := &types.TraceEvent{
				SandboxID: "sb-concurrent",
				EventType: types.EventTypeAction,
			}
			done <- rec.RecordEvent(event)
		}(i)
	}

	for range 20 {
		if err := <-done; err != nil {
			t.Errorf("concurrent RecordEvent: %v", err)
		}
	}

	events, _ := store.ListEvents("sb-concurrent")
	if len(events) != 20 {
		t.Errorf("got %d events, want 20", len(events))
	}
}

// --- Replayer tests ---

func TestReplayer_LoadTrace(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Microsecond)
	store.SaveEvent(&types.TraceEvent{ID: "root", SandboxID: "sb-1", EventType: types.EventTypeSpanStart, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "child1", SandboxID: "sb-1", ParentID: "root", EventType: types.EventTypeAction, Timestamp: now.Add(time.Second)})
	store.SaveEvent(&types.TraceEvent{ID: "child2", SandboxID: "sb-1", ParentID: "root", EventType: types.EventTypeAction, Timestamp: now.Add(2 * time.Second)})

	replayer := NewReplayer(store)
	trace, err := replayer.LoadTrace("sb-1")
	if err != nil {
		t.Fatalf("LoadTrace: %v", err)
	}

	if len(trace.Events) != 3 {
		t.Fatalf("got %d events, want 3", len(trace.Events))
	}
	if len(trace.RootSpans) != 1 {
		t.Fatalf("got %d root spans, want 1", len(trace.RootSpans))
	}
	if len(trace.RootSpans[0].Children) != 2 {
		t.Fatalf("root has %d children, want 2", len(trace.RootSpans[0].Children))
	}
}

func TestReplayer_StepAndRewind(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Microsecond)
	for i := range 3 {
		store.SaveEvent(&types.TraceEvent{
			ID:        fmt.Sprintf("e%d", i),
			SandboxID: "sb-1",
			EventType: types.EventTypeAction,
			Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}

	replayer := NewReplayer(store)
	trace, _ := replayer.LoadTrace("sb-1")
	state := NewReplayState(trace)

	// Step through all events.
	e0, more := Step(state)
	if e0.ID != "e0" || !more {
		t.Errorf("step 0: id=%q more=%v", e0.ID, more)
	}

	e1, more := Step(state)
	if e1.ID != "e1" || !more {
		t.Errorf("step 1: id=%q more=%v", e1.ID, more)
	}

	e2, more := Step(state)
	if e2.ID != "e2" || more {
		t.Errorf("step 2: id=%q more=%v (want false)", e2.ID, more)
	}

	// No more events.
	eNil, more := Step(state)
	if eNil != nil || more {
		t.Error("expected nil after exhaustion")
	}

	// Rewind and step again.
	Rewind(state)
	e0Again, _ := Step(state)
	if e0Again.ID != "e0" {
		t.Errorf("after rewind: id=%q, want e0", e0Again.ID)
	}
}

func TestReplayer_GetTimeline(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Microsecond)
	store.SaveEvent(&types.TraceEvent{ID: "root", SandboxID: "sb-1", EventType: types.EventTypeSpanStart, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "child", SandboxID: "sb-1", ParentID: "root", EventType: types.EventTypeAction, Timestamp: now.Add(time.Second)})
	store.SaveEvent(&types.TraceEvent{ID: "grandchild", SandboxID: "sb-1", ParentID: "child", EventType: types.EventTypeAction, Timestamp: now.Add(2 * time.Second)})

	replayer := NewReplayer(store)
	timeline, err := replayer.GetTimeline("sb-1")
	if err != nil {
		t.Fatalf("GetTimeline: %v", err)
	}

	if len(timeline) != 3 {
		t.Fatalf("got %d entries, want 3", len(timeline))
	}

	expectedDepths := map[string]int{"root": 0, "child": 1, "grandchild": 2}
	for _, entry := range timeline {
		want := expectedDepths[entry.Event.ID]
		if entry.Depth != want {
			t.Errorf("event %q depth = %d, want %d", entry.Event.ID, entry.Depth, want)
		}
	}
}

// --- OTel export tests ---

func TestExportToOTLP_BasicFormat(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-001",
			SandboxID: "sandbox-test-1234",
			EventType: types.EventTypeAction,
			Action:    &types.Action{Type: types.ActionTypeFile},
			Result:    &types.ActionResult{Success: true},
			Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			DurationNs: int64(100 * time.Millisecond),
		},
	}

	data, err := ExportToOTLP(events)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}

	var export OTLPExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(export.ResourceSpans) != 1 {
		t.Fatal("expected 1 ResourceSpans")
	}

	rs := export.ResourceSpans[0]
	if len(rs.Resource.Attributes) == 0 {
		t.Error("expected resource attributes")
	}

	spans := rs.ScopeSpans[0].Spans
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	span := spans[0]
	if span.Name == "" {
		t.Error("span name should not be empty")
	}
	if span.StartTimeUnixNano == 0 {
		t.Error("start time should be set")
	}
	if span.Status.Code != StatusCodeOK {
		t.Errorf("status code = %d, want OK (%d)", span.Status.Code, StatusCodeOK)
	}
}

func TestExportToOTLP_ErrorStatus(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-err",
			SandboxID: "sandbox-test-1234",
			EventType: types.EventTypeError,
			Result:    &types.ActionResult{Success: false, Error: "permission denied"},
			Timestamp: time.Now(),
		},
	}

	data, _ := ExportToOTLP(events)
	var export OTLPExport
	json.Unmarshal(data, &export)

	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	if span.Status.Code != StatusCodeError {
		t.Errorf("status code = %d, want ERROR (%d)", span.Status.Code, StatusCodeError)
	}
	if span.Status.Message != "permission denied" {
		t.Errorf("status message = %q, want %q", span.Status.Message, "permission denied")
	}
}

func TestExportToOTLP_ParentChild(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "parent-span-00001",
			SandboxID: "sandbox-test-1234",
			EventType: types.EventTypeSpanStart,
			Timestamp: time.Now(),
		},
		{
			ID:        "child-span-000001",
			SandboxID: "sandbox-test-1234",
			ParentID:  "parent-span-00001",
			EventType: types.EventTypeAction,
			Timestamp: time.Now(),
		},
	}

	data, _ := ExportToOTLP(events)
	var export OTLPExport
	json.Unmarshal(data, &export)

	spans := export.ResourceSpans[0].ScopeSpans[0].Spans
	if spans[0].ParentSpanID != "" {
		t.Error("root span should have no parent")
	}
	if spans[1].ParentSpanID == "" {
		t.Error("child span should have parent span ID")
	}
}

func TestExportToOTLP_PolicyAttributes(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-policy-test1",
			SandboxID: "sandbox-test-1234",
			EventType: types.EventTypePolicyDecision,
			PolicyDecision: &types.PolicyDecision{
				Allowed: false,
				Rule:    "no-network",
			},
			Timestamp: time.Now(),
		},
	}

	data, _ := ExportToOTLP(events)
	var export OTLPExport
	json.Unmarshal(data, &export)

	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	found := false
	for _, attr := range span.Attributes {
		if attr.Key == "policy.allowed" && !attr.Value.BoolValue {
			found = true
		}
	}
	if !found {
		t.Error("expected policy.allowed=false attribute")
	}
}

