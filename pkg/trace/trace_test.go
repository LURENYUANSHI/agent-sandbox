package trace

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ---------------------------------------------------------------------------
// Recorder: in-memory
// ---------------------------------------------------------------------------

func TestRecorder_InMemory(t *testing.T) {
	rec, err := NewRecorder("")
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	defer rec.Close()

	rec.Record(types.TraceEvent{SandboxID: "sb1", Type: types.EventSandboxStarted})
	rec.Record(types.TraceEvent{SandboxID: "sb1", Type: types.EventActionRequested, ActionID: "a1"})
	rec.Record(types.TraceEvent{SandboxID: "sb2", Type: types.EventSandboxStarted})

	events, err := rec.GetEvents("sb1")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events for sb1, got %d", len(events))
	}

	events, err = rec.GetEvents("sb2")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event for sb2, got %d", len(events))
	}
}

func TestRecorder_WithStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "trace.db")

	rec, err := NewRecorder(dbPath)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}

	err = rec.Record(types.TraceEvent{
		ID:        "e1",
		SandboxID: "sb-store",
		Type:      types.EventActionExecuted,
		ActionID:  "a1",
		Timestamp: time.Now(),
		Data:      map[string]string{"result": "ok"},
		Duration:  100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	rec.Close()

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	events, err := store.Load("sb-store")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ActionID != "a1" {
		t.Errorf("action_id = %q", events[0].ActionID)
	}
	if events[0].Data["result"] != "ok" {
		t.Errorf("data = %v", events[0].Data)
	}
}

func TestRecorder_AutoFillsIDAndTimestamp(t *testing.T) {
	rec, _ := NewRecorder("")
	defer rec.Close()

	rec.Record(types.TraceEvent{SandboxID: "sb"})

	events, _ := rec.GetEvents("sb")
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}
	if events[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
	if events[0].Timestamp.IsZero() {
		t.Error("expected auto-filled timestamp")
	}
}

// ---------------------------------------------------------------------------
// Recorder: concurrent recording
// ---------------------------------------------------------------------------

func TestRecorder_ConcurrentRecording(t *testing.T) {
	rec, err := NewRecorder("")
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	defer rec.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec.Record(types.TraceEvent{
				SandboxID: "concurrent",
				Type:      types.EventActionRequested,
			})
		}()
	}
	wg.Wait()

	events, err := rec.GetEvents("concurrent")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 100 {
		t.Errorf("expected 100 events, got %d", len(events))
	}
}

func TestRecorder_CloseWithoutStore(t *testing.T) {
	rec, _ := NewRecorder("")
	if err := rec.Close(); err != nil {
		t.Errorf("Close without store should return nil: %v", err)
	}
}

func TestRecorder_GetEventsEmpty(t *testing.T) {
	rec, _ := NewRecorder("")
	defer rec.Close()

	events, err := rec.GetEvents("nonexistent")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

// ---------------------------------------------------------------------------
// Store: SQLite in-memory
// ---------------------------------------------------------------------------

func TestStore_InMemory(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	now := time.Now().Truncate(time.Second)
	err = store.Save(types.TraceEvent{
		ID:        "evt-1",
		SandboxID: "sb1",
		Type:      types.EventSandboxStarted,
		Timestamp: now,
		Data:      map[string]string{"key": "val"},
		Duration:  50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	events, err := store.Load("sb1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "evt-1" {
		t.Errorf("ID = %q", events[0].ID)
	}
	if events[0].Data["key"] != "val" {
		t.Errorf("Data = %v", events[0].Data)
	}
}

func TestStore_EmptyLoad(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	events, err := store.Load("nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestStore_ManyEvents(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	base := time.Now()
	for i := 0; i < 150; i++ {
		store.Save(types.TraceEvent{
			ID:        fmt.Sprintf("evt-%03d", i),
			SandboxID: "sb-many",
			Type:      types.EventActionExecuted,
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
		})
	}

	events, err := store.Load("sb-many")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 150 {
		t.Errorf("expected 150 events, got %d", len(events))
	}

	// Verify ordering by timestamp
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Errorf("events not ordered: [%d]=%v before [%d]=%v",
				i, events[i].Timestamp, i-1, events[i-1].Timestamp)
			break
		}
	}
}

func TestStore_MultipleSandboxes(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	now := time.Now()
	store.Save(types.TraceEvent{ID: "e1", SandboxID: "sb-a", Type: types.EventSandboxStarted, Timestamp: now})
	store.Save(types.TraceEvent{ID: "e2", SandboxID: "sb-b", Type: types.EventSandboxStarted, Timestamp: now})
	store.Save(types.TraceEvent{ID: "e3", SandboxID: "sb-a", Type: types.EventActionExecuted, Timestamp: now})

	eventsA, _ := store.Load("sb-a")
	eventsB, _ := store.Load("sb-b")
	if len(eventsA) != 2 {
		t.Errorf("sb-a: expected 2 events, got %d", len(eventsA))
	}
	if len(eventsB) != 1 {
		t.Errorf("sb-b: expected 1 event, got %d", len(eventsB))
	}
}

func TestStore_SaveUpsert(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	now := time.Now()
	store.Save(types.TraceEvent{ID: "e1", SandboxID: "sb", Type: types.EventSandboxStarted, Timestamp: now, ActionID: "first"})
	store.Save(types.TraceEvent{ID: "e1", SandboxID: "sb", Type: types.EventSandboxStarted, Timestamp: now, ActionID: "second"})

	events, _ := store.Load("sb")
	if len(events) != 1 {
		t.Fatalf("expected 1 event after upsert, got %d", len(events))
	}
	if events[0].ActionID != "second" {
		t.Errorf("expected upserted action_id=second, got %q", events[0].ActionID)
	}
}

func TestStore_NilData(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	store.Save(types.TraceEvent{ID: "e1", SandboxID: "sb", Type: types.EventSandboxStarted, Timestamp: time.Now()})
	events, _ := store.Load("sb")
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}
	// Data should be nil or empty when none was saved
	if events[0].Data != nil && len(events[0].Data) != 0 {
		t.Errorf("expected nil/empty data, got %v", events[0].Data)
	}
}

// ---------------------------------------------------------------------------
// Mock TraceStore for Replayer tests
// ---------------------------------------------------------------------------

type mockTraceStore struct {
	events map[string][]*types.TraceEvent
}

func newMockStore() *mockTraceStore {
	return &mockTraceStore{events: make(map[string][]*types.TraceEvent)}
}

func (m *mockTraceStore) SaveEvent(event *types.TraceEvent) error {
	m.events[event.SandboxID] = append(m.events[event.SandboxID], event)
	return nil
}

func (m *mockTraceStore) GetEvent(id string) (*types.TraceEvent, error) {
	for _, evts := range m.events {
		for _, e := range evts {
			if e.ID == id {
				return e, nil
			}
		}
	}
	return nil, nil
}

func (m *mockTraceStore) ListEvents(sandboxID string) ([]*types.TraceEvent, error) {
	return m.events[sandboxID], nil
}

func (m *mockTraceStore) QueryEvents(query types.EventQuery) ([]*types.TraceEvent, error) {
	var result []*types.TraceEvent
	evts := m.events[query.SandboxID]
	for _, e := range evts {
		if query.EventType != "" && e.Type != query.EventType {
			continue
		}
		if !query.StartTime.IsZero() && e.Timestamp.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && e.Timestamp.After(query.EndTime) {
			continue
		}
		result = append(result, e)
		if query.Limit > 0 && len(result) >= query.Limit {
			break
		}
	}
	return result, nil
}

func (m *mockTraceStore) DeleteEvents(sandboxID string) error {
	delete(m.events, sandboxID)
	return nil
}

func (m *mockTraceStore) Close() error { return nil }

// ---------------------------------------------------------------------------
// Replayer: LoadTrace, Step, Rewind
// ---------------------------------------------------------------------------

func TestReplayer_LoadTrace(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.SaveEvent(&types.TraceEvent{ID: "e1", SandboxID: "sb1", Type: types.EventSandboxStarted, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e2", SandboxID: "sb1", Type: types.EventActionRequested, Timestamp: now})

	replayer := NewReplayer(store)
	trace, err := replayer.LoadTrace("sb1")
	if err != nil {
		t.Fatalf("LoadTrace: %v", err)
	}
	if trace.SandboxID != "sb1" {
		t.Errorf("SandboxID = %q", trace.SandboxID)
	}
	if len(trace.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(trace.Events))
	}
	if len(trace.RootSpans) != 2 {
		t.Errorf("expected 2 root spans, got %d", len(trace.RootSpans))
	}
}

func TestReplayer_LoadTrace_Empty(t *testing.T) {
	store := newMockStore()
	replayer := NewReplayer(store)
	trace, err := replayer.LoadTrace("empty")
	if err != nil {
		t.Fatalf("LoadTrace: %v", err)
	}
	if len(trace.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(trace.Events))
	}
}

func TestStep_ThroughAllEvents(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.SaveEvent(&types.TraceEvent{ID: "e1", SandboxID: "sb", Type: types.EventSandboxStarted, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e2", SandboxID: "sb", Type: types.EventActionRequested, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e3", SandboxID: "sb", Type: types.EventActionExecuted, Timestamp: now})

	replayer := NewReplayer(store)
	trace, _ := replayer.LoadTrace("sb")
	state := NewReplayState(trace)

	// Step through all events
	var collected []string
	for {
		event, hasMore := Step(state)
		if event == nil {
			break
		}
		collected = append(collected, event.ID)
		if !hasMore {
			break
		}
	}

	if len(collected) != 3 {
		t.Errorf("expected 3 events, collected %d", len(collected))
	}

	// Stepping past the end returns nil
	event, hasMore := Step(state)
	if event != nil || hasMore {
		t.Error("expected nil event past end")
	}
}

func TestRewind(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.SaveEvent(&types.TraceEvent{ID: "e1", SandboxID: "sb", Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "e2", SandboxID: "sb", Timestamp: now})

	replayer := NewReplayer(store)
	trace, _ := replayer.LoadTrace("sb")
	state := NewReplayState(trace)

	// Advance
	Step(state)
	Step(state)

	// Rewind
	Rewind(state)

	// Should be back at the start
	event, _ := Step(state)
	if event == nil || event.ID != "e1" {
		t.Errorf("after rewind, expected e1, got %v", event)
	}
}

func TestStep_EmptyTrace(t *testing.T) {
	trace := &types.Trace{SandboxID: "empty", Events: []*types.TraceEvent{}}
	state := NewReplayState(trace)
	event, hasMore := Step(state)
	if event != nil || hasMore {
		t.Error("expected nil for empty trace")
	}
}

// ---------------------------------------------------------------------------
// Replayer: GetTimeline
// ---------------------------------------------------------------------------

func TestReplayer_GetTimeline(t *testing.T) {
	store := newMockStore()
	now := time.Now()
	store.SaveEvent(&types.TraceEvent{ID: "root1", SandboxID: "sb", Type: types.EventSandboxStarted, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "child1", SandboxID: "sb", ParentID: "root1", Type: types.EventActionRequested, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "grandchild1", SandboxID: "sb", ParentID: "child1", Type: types.EventActionExecuted, Timestamp: now})
	store.SaveEvent(&types.TraceEvent{ID: "root2", SandboxID: "sb", Type: types.EventSandboxStopped, Timestamp: now})

	replayer := NewReplayer(store)
	entries, err := replayer.GetTimeline("sb")
	if err != nil {
		t.Fatalf("GetTimeline: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// Check depths
	depthByID := map[string]int{}
	for _, e := range entries {
		depthByID[e.Event.ID] = e.Depth
	}
	if depthByID["root1"] != 0 {
		t.Errorf("root1 depth = %d, want 0", depthByID["root1"])
	}
	if depthByID["child1"] != 1 {
		t.Errorf("child1 depth = %d, want 1", depthByID["child1"])
	}
	if depthByID["grandchild1"] != 2 {
		t.Errorf("grandchild1 depth = %d, want 2", depthByID["grandchild1"])
	}
	if depthByID["root2"] != 0 {
		t.Errorf("root2 depth = %d, want 0", depthByID["root2"])
	}
}

func TestReplayer_GetTimeline_Empty(t *testing.T) {
	store := newMockStore()
	replayer := NewReplayer(store)
	entries, err := replayer.GetTimeline("empty")
	if err != nil {
		t.Fatalf("GetTimeline: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// buildSpanTree
// ---------------------------------------------------------------------------

func TestBuildSpanTree_FlatEvents(t *testing.T) {
	events := []*types.TraceEvent{
		{ID: "e1"},
		{ID: "e2"},
	}
	roots := buildSpanTree(events)
	if len(roots) != 2 {
		t.Errorf("expected 2 roots, got %d", len(roots))
	}
}

func TestBuildSpanTree_ParentChild(t *testing.T) {
	events := []*types.TraceEvent{
		{ID: "parent"},
		{ID: "child1", ParentID: "parent"},
		{ID: "child2", ParentID: "parent"},
	}
	roots := buildSpanTree(events)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(roots[0].Children))
	}
}

func TestBuildSpanTree_DeepNesting(t *testing.T) {
	events := []*types.TraceEvent{
		{ID: "a"},
		{ID: "b", ParentID: "a"},
		{ID: "c", ParentID: "b"},
	}
	roots := buildSpanTree(events)
	if len(roots) != 1 {
		t.Fatal("expected 1 root")
	}
	if len(roots[0].Children) != 1 {
		t.Fatal("expected 1 child")
	}
	if len(roots[0].Children[0].Children) != 1 {
		t.Fatal("expected 1 grandchild")
	}
}

func TestBuildSpanTree_OrphanBecomesRoot(t *testing.T) {
	events := []*types.TraceEvent{
		{ID: "orphan", ParentID: "missing-parent"},
	}
	roots := buildSpanTree(events)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root (orphan), got %d", len(roots))
	}
}

func TestBuildSpanTree_Empty(t *testing.T) {
	roots := buildSpanTree(nil)
	if len(roots) != 0 {
		t.Errorf("expected 0 roots, got %d", len(roots))
	}
}

// ---------------------------------------------------------------------------
// OTel Export
// ---------------------------------------------------------------------------

func TestExportToOTLP_BasicStructure(t *testing.T) {
	now := time.Now()
	events := []*types.TraceEvent{
		{
			ID:         "span-001-abcdef",
			SandboxID:  "sandbox-12345678",
			Type:       types.EventActionExecuted,
			Timestamp:  now,
			DurationNs: int64(100 * time.Millisecond),
			Action:     &types.Action{Type: types.ActionFileRead},
			Result:     &types.ActionResult{Success: true},
			Attributes: map[string]string{"custom-key": "custom-val"},
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
		t.Fatalf("expected 1 ResourceSpans, got %d", len(export.ResourceSpans))
	}
	rs := export.ResourceSpans[0]

	// Check resource attribute
	foundService := false
	for _, attr := range rs.Resource.Attributes {
		if attr.Key == "service.name" && attr.Value.StringValue == "agent-sandbox" {
			foundService = true
		}
	}
	if !foundService {
		t.Error("missing service.name attribute")
	}

	if len(rs.ScopeSpans) != 1 {
		t.Fatalf("expected 1 ScopeSpans, got %d", len(rs.ScopeSpans))
	}
	ss := rs.ScopeSpans[0]
	if ss.Scope.Name != "agent-sandbox/trace" {
		t.Errorf("scope name = %q", ss.Scope.Name)
	}

	if len(ss.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(ss.Spans))
	}
	span := ss.Spans[0]
	if span.Kind != SpanKindInternal {
		t.Errorf("kind = %d, want %d", span.Kind, SpanKindInternal)
	}
	if span.Status.Code != StatusCodeOK {
		t.Errorf("status code = %d, want %d", span.Status.Code, StatusCodeOK)
	}
}

func TestExportToOTLP_ErrorStatus(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-err-12345",
			SandboxID: "sandbox-12345678",
			Type:      types.EventActionFailed,
			Timestamp: time.Now(),
			Result:    &types.ActionResult{Success: false, Error: "something failed"},
		},
	}

	data, err := ExportToOTLP(events)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}

	var export OTLPExport
	json.Unmarshal(data, &export)
	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	if span.Status.Code != StatusCodeError {
		t.Errorf("status code = %d, want %d", span.Status.Code, StatusCodeError)
	}
	if span.Status.Message != "something failed" {
		t.Errorf("status message = %q", span.Status.Message)
	}
}

func TestExportToOTLP_NoResult(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-unset-1234",
			SandboxID: "sandbox-12345678",
			Type:      types.EventSandboxStarted,
			Timestamp: time.Now(),
		},
	}

	data, err := ExportToOTLP(events)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}

	var export OTLPExport
	json.Unmarshal(data, &export)
	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	if span.Status.Code != StatusCodeUnset {
		t.Errorf("status code = %d, want %d (unset)", span.Status.Code, StatusCodeUnset)
	}
}

func TestExportToOTLP_ParentSpan(t *testing.T) {
	now := time.Now()
	events := []*types.TraceEvent{
		{ID: "parent-span-1234", SandboxID: "sandbox-12345678", Type: types.EventSandboxStarted, Timestamp: now},
		{ID: "child-span-12345", SandboxID: "sandbox-12345678", ParentID: "parent-span-1234", Type: types.EventActionExecuted, Timestamp: now},
	}

	data, err := ExportToOTLP(events)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}

	var export OTLPExport
	json.Unmarshal(data, &export)
	spans := export.ResourceSpans[0].ScopeSpans[0].Spans
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}
	if spans[0].ParentSpanID != "" {
		t.Errorf("parent span should have empty ParentSpanID, got %q", spans[0].ParentSpanID)
	}
	if spans[1].ParentSpanID == "" {
		t.Error("child span should have ParentSpanID set")
	}
}

func TestExportToOTLP_PolicyDecisionAttributes(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-policy-1234",
			SandboxID: "sandbox-12345678",
			Type:      types.EventPolicyEvaluated,
			Timestamp: time.Now(),
			PolicyDecision: &types.PolicyDecision{
				Allowed: true,
				Rule:    "allow-all",
			},
		},
	}

	data, err := ExportToOTLP(events)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}

	var export OTLPExport
	json.Unmarshal(data, &export)
	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]

	foundAllowed := false
	foundRule := false
	for _, attr := range span.Attributes {
		if attr.Key == "policy.allowed" && attr.Value.BoolValue {
			foundAllowed = true
		}
		if attr.Key == "policy.rule" && attr.Value.StringValue == "allow-all" {
			foundRule = true
		}
	}
	if !foundAllowed {
		t.Error("missing policy.allowed attribute")
	}
	if !foundRule {
		t.Error("missing policy.rule attribute")
	}
}

func TestExportToOTLP_Empty(t *testing.T) {
	data, err := ExportToOTLP(nil)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}
	var export OTLPExport
	json.Unmarshal(data, &export)
	spans := export.ResourceSpans[0].ScopeSpans[0].Spans
	if len(spans) != 0 {
		t.Errorf("expected 0 spans for nil events, got %d", len(spans))
	}
}

func TestExportToOTLP_ShortSandboxID(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-short-12345",
			SandboxID: "short",
			Type:      types.EventSandboxStarted,
			Timestamp: time.Now(),
		},
	}

	data, err := ExportToOTLP(events)
	if err != nil {
		t.Fatalf("ExportToOTLP: %v", err)
	}

	var export OTLPExport
	json.Unmarshal(data, &export)
	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	// Should use zero trace ID since sandbox ID is too short
	if span.TraceID != "00000000000000000000000000000000" {
		t.Errorf("expected zero traceID for short sandbox ID, got %q", span.TraceID)
	}
}

func TestExportToOTLP_SpanNameWithAction(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-name-123456",
			SandboxID: "sandbox-12345678",
			Type:      types.EventActionExecuted,
			Timestamp: time.Now(),
			Action:    &types.Action{Type: types.ActionFileRead},
		},
	}

	data, _ := ExportToOTLP(events)
	var export OTLPExport
	json.Unmarshal(data, &export)
	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	expected := "file:read.action.executed"
	if span.Name != expected {
		t.Errorf("span name = %q, want %q", span.Name, expected)
	}
}

func TestExportToOTLP_SpanNameWithoutAction(t *testing.T) {
	events := []*types.TraceEvent{
		{
			ID:        "span-noact-12345",
			SandboxID: "sandbox-12345678",
			Type:      types.EventSandboxStarted,
			Timestamp: time.Now(),
		},
	}

	data, _ := ExportToOTLP(events)
	var export OTLPExport
	json.Unmarshal(data, &export)
	span := export.ResourceSpans[0].ScopeSpans[0].Spans[0]
	if span.Name != "sandbox.started" {
		t.Errorf("span name = %q, want %q", span.Name, "sandbox.started")
	}
}

