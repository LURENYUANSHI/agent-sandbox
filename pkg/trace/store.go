package trace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Store persists trace events to SQLite.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at path.
func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS trace_events (
			id         TEXT PRIMARY KEY,
			sandbox_id TEXT NOT NULL,
			type       TEXT NOT NULL,
			timestamp  DATETIME NOT NULL,
			action_id  TEXT,
			data       TEXT,
			duration   INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_trace_sandbox ON trace_events(sandbox_id);
	`)
	if err != nil {
		return fmt.Errorf("migrate trace_events: %w", err)
	}
	return nil
}

// Save inserts a trace event.
func (s *Store) Save(event types.TraceEvent) error {
	dataJSON, _ := json.Marshal(event.Data)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO trace_events (id, sandbox_id, type, timestamp, action_id, data, duration)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.SandboxID, string(event.Type), event.Timestamp,
		event.ActionID, string(dataJSON), int64(event.Duration),
	)
	if err != nil {
		return fmt.Errorf("save trace event: %w", err)
	}
	return nil
}

// Load retrieves all events for a sandbox, ordered by timestamp.
func (s *Store) Load(sandboxID string) ([]types.TraceEvent, error) {
	rows, err := s.db.Query(
		`SELECT id, sandbox_id, type, timestamp, action_id, data, duration
		 FROM trace_events WHERE sandbox_id = ? ORDER BY timestamp`, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("query trace events: %w", err)
	}
	defer rows.Close()

	var events []types.TraceEvent
	for rows.Next() {
		var e types.TraceEvent
		var dataJSON string
		var dur int64
		if err := rows.Scan(&e.ID, &e.SandboxID, &e.Type, &e.Timestamp, &e.ActionID, &dataJSON, &dur); err != nil {
			return nil, fmt.Errorf("scan trace event: %w", err)
		}
		e.Duration = time.Duration(dur)
		if dataJSON != "" {
			json.Unmarshal([]byte(dataJSON), &e.Data)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}
