package trace

import (
	"sort"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Replayer loads and replays trace events in chronological order.
type Replayer struct {
	store *Store
}

// NewReplayer creates a replayer backed by the given store.
func NewReplayer(store *Store) *Replayer {
	return &Replayer{store: store}
}

// Replay returns events for a sandbox sorted by start time.
func (r *Replayer) Replay(sandboxID string) ([]*types.TraceEvent, error) {
	events, err := r.store.GetBySandbox(sandboxID)
	if err != nil {
		return nil, err
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})
	return events, nil
}
