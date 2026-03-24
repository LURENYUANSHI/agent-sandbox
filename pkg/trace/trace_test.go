package trace

import (
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func makeEvent(id, sandboxID string, decision types.Decision) *types.TraceEvent {
	now := time.Now()
	return &types.TraceEvent{
		ID:        id,
		TraceID:   "trace-1",
		SandboxID: sandboxID,
		Action: types.Action{
			ID:     "action-" + id,
			Type:   types.ActionTypeFile,
			Path:   "/tmp/test.txt",
			FileOp: types.FileOpRead,
		},
		Decision:  decision,
		StartTime: now,
		EndTime:   now.Add(10 * time.Millisecond),
	}
}

func TestStoreAndRetrieve(t *testing.T) {
	store := NewStore()
	event := makeEvent("e1", "sb-1", types.DecisionAllowed)

	if err := store.Save(event); err != nil {
		t.Fatalf("save: %v", err)
	}

	events, err := store.GetBySandbox("sb-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "e1" {
		t.Errorf("event ID = %s, want e1", events[0].ID)
	}
}

func TestRecorder(t *testing.T) {
	store := NewStore()
	recorder := NewRecorder(store)

	event := makeEvent("e1", "sb-1", types.DecisionAllowed)
	event.DurationMs = 0

	if err := recorder.Record(event); err != nil {
		t.Fatalf("record: %v", err)
	}

	if event.DurationMs == 0 {
		t.Error("recorder should set DurationMs")
	}

	events, err := recorder.GetEvents("sb-1")
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestReplayer(t *testing.T) {
	store := NewStore()
	now := time.Now()

	// Insert events out of order
	e2 := makeEvent("e2", "sb-1", types.DecisionAllowed)
	e2.StartTime = now.Add(100 * time.Millisecond)
	e1 := makeEvent("e1", "sb-1", types.DecisionDenied)
	e1.StartTime = now

	store.Save(e2)
	store.Save(e1)

	replayer := NewReplayer(store)
	events, err := replayer.Replay("sb-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ID != "e1" {
		t.Errorf("first event should be e1 (earlier), got %s", events[0].ID)
	}
}

func TestOTelExport(t *testing.T) {
	event := makeEvent("e1", "sb-1", types.DecisionAllowed)
	spans := ExportToOTel([]*types.TraceEvent{event})

	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].TraceID != "trace-1" {
		t.Errorf("traceID = %s, want trace-1", spans[0].TraceID)
	}

	data, err := ExportToJSON([]*types.TraceEvent{event})
	if err != nil {
		t.Fatalf("export JSON: %v", err)
	}
	if len(data) == 0 {
		t.Error("exported JSON should not be empty")
	}
}
