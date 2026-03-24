package trace

import (
	"fmt"
	"sync"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Recorder captures trace events and persists them to a store.
type Recorder struct {
	store *Store
	mu    sync.Mutex
}

// NewRecorder creates a recorder backed by the given store.
func NewRecorder(store *Store) *Recorder {
	return &Recorder{store: store}
}

// Record saves a trace event. It sets EndTime and DurationMs if not already set.
func (r *Recorder) Record(event *types.TraceEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.EndTime.IsZero() {
		event.EndTime = time.Now()
	}
	if event.DurationMs == 0 && !event.StartTime.IsZero() {
		event.DurationMs = event.EndTime.Sub(event.StartTime).Milliseconds()
	}
	return r.store.Save(event)
}

// GetEvents returns all events for a sandbox.
func (r *Recorder) GetEvents(sandboxID string) ([]*types.TraceEvent, error) {
	return r.store.GetBySandbox(sandboxID)
}
