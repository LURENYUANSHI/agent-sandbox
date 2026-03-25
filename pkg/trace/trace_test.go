package trace

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestRecorder_InMemory(t *testing.T) {
	rec, err := NewRecorder("")
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	defer rec.Close()

	err = rec.Record(types.TraceEvent{
		SandboxID: "sb1",
		Type:      types.EventSandboxStarted,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	err = rec.Record(types.TraceEvent{
		SandboxID: "sb1",
		Type:      types.EventActionRequested,
		ActionID:  "a1",
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Different sandbox
	err = rec.Record(types.TraceEvent{
		SandboxID: "sb2",
		Type:      types.EventSandboxStarted,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

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

	// Verify data persisted by opening a new store
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
