package trace

import (
	"fmt"
	"sync"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Recorder implements types.TraceRecorder with in-memory storage
// and optional persistence to a Store.
type Recorder struct {
	mu     sync.RWMutex
	events []types.TraceEvent
	store  *Store
}

// NewRecorder creates a recorder. If dbPath is non-empty, events are
// also persisted to SQLite.
func NewRecorder(dbPath string) (*Recorder, error) {
	r := &Recorder{}
	if dbPath != "" {
		store, err := NewStore(dbPath)
		if err != nil {
			return nil, fmt.Errorf("open trace store: %w", err)
		}
		r.store = store
	}
	return r, nil
}

// Record adds a trace event.
func (r *Recorder) Record(event types.TraceEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}

	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()

	if r.store != nil {
		return r.store.Save(event)
	}
	return nil
}

// GetEvents returns all events for a given sandbox ID.
func (r *Recorder) GetEvents(sandboxID string) ([]types.TraceEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []types.TraceEvent
	for _, e := range r.events {
		if e.SandboxID == sandboxID {
			result = append(result, e)
		}
	}
	return result, nil
}

// Close releases resources.
func (r *Recorder) Close() error {
	if r.store != nil {
		return r.store.Close()
	}
	return nil
}
