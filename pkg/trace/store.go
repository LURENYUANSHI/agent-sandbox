package trace

import (
	"fmt"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Store is an in-memory trace event store.
type Store struct {
	events map[string][]*types.TraceEvent // keyed by sandbox ID
	mu     sync.RWMutex
}

// NewStore creates a new in-memory trace store.
func NewStore() *Store {
	return &Store{
		events: make(map[string][]*types.TraceEvent),
	}
}

// Save persists a trace event.
func (s *Store) Save(event *types.TraceEvent) error {
	if event.SandboxID == "" {
		return fmt.Errorf("sandbox_id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[event.SandboxID] = append(s.events[event.SandboxID], event)
	return nil
}

// GetBySandbox returns all events for a sandbox, ordered by start time.
func (s *Store) GetBySandbox(sandboxID string) ([]*types.TraceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.events[sandboxID]
	result := make([]*types.TraceEvent, len(events))
	copy(result, events)
	return result, nil
}

// GetByTrace returns all events with a specific trace ID.
func (s *Store) GetByTrace(traceID string) ([]*types.TraceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*types.TraceEvent
	for _, events := range s.events {
		for _, e := range events {
			if e.TraceID == traceID {
				result = append(result, e)
			}
		}
	}
	return result, nil
}

// GetAll returns every stored event.
func (s *Store) GetAll() ([]*types.TraceEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*types.TraceEvent
	for _, events := range s.events {
		result = append(result, events...)
	}
	return result, nil
}
