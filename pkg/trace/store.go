package trace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// SQLiteStore implements types.TraceStore using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-based trace store.
// Use ":memory:" for in-memory databases (useful for testing).
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging sqlite db: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trace_events (
		id TEXT PRIMARY KEY,
		sandbox_id TEXT NOT NULL,
		parent_id TEXT,
		event_type TEXT NOT NULL,
		action_json TEXT,
		result_json TEXT,
		policy_decision_json TEXT,
		timestamp DATETIME NOT NULL,
		duration_ns INTEGER,
		attributes_json TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_sandbox_id ON trace_events(sandbox_id);
	CREATE INDEX IF NOT EXISTS idx_event_type ON trace_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON trace_events(timestamp);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SaveEvent persists a trace event to the store.
func (s *SQLiteStore) SaveEvent(event *types.TraceEvent) error {
	actionJSON, err := jsonMarshalNullable(event.Action)
	if err != nil {
		return fmt.Errorf("marshaling action: %w", err)
	}

	resultJSON, err := jsonMarshalNullable(event.Result)
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}

	policyJSON, err := jsonMarshalNullable(event.PolicyDecision)
	if err != nil {
		return fmt.Errorf("marshaling policy decision: %w", err)
	}

	attrsJSON, err := jsonMarshalNullable(event.Attributes)
	if err != nil {
		return fmt.Errorf("marshaling attributes: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO trace_events (id, sandbox_id, parent_id, event_type, action_json, result_json, policy_decision_json, timestamp, duration_ns, attributes_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.SandboxID,
		nullableString(event.ParentID),
		string(event.EventType),
		actionJSON,
		resultJSON,
		policyJSON,
		event.Timestamp.UTC().Format(time.RFC3339Nano),
		event.DurationNs,
		attrsJSON,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}
	return nil
}

// GetEvent retrieves a single event by ID.
func (s *SQLiteStore) GetEvent(id string) (*types.TraceEvent, error) {
	row := s.db.QueryRow(`SELECT id, sandbox_id, parent_id, event_type, action_json, result_json, policy_decision_json, timestamp, duration_ns, attributes_json FROM trace_events WHERE id = ?`, id)
	return scanEvent(row)
}

// ListEvents returns all events for a sandbox, ordered by timestamp.
func (s *SQLiteStore) ListEvents(sandboxID string) ([]*types.TraceEvent, error) {
	rows, err := s.db.Query(`SELECT id, sandbox_id, parent_id, event_type, action_json, result_json, policy_decision_json, timestamp, duration_ns, attributes_json FROM trace_events WHERE sandbox_id = ? ORDER BY timestamp ASC`, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// QueryEvents returns events matching the given query filters.
func (s *SQLiteStore) QueryEvents(query types.EventQuery) ([]*types.TraceEvent, error) {
	where := "WHERE 1=1"
	args := []any{}

	if query.SandboxID != "" {
		where += " AND sandbox_id = ?"
		args = append(args, query.SandboxID)
	}
	if query.EventType != "" {
		where += " AND event_type = ?"
		args = append(args, string(query.EventType))
	}
	if !query.StartTime.IsZero() {
		where += " AND timestamp >= ?"
		args = append(args, query.StartTime.UTC().Format(time.RFC3339Nano))
	}
	if !query.EndTime.IsZero() {
		where += " AND timestamp <= ?"
		args = append(args, query.EndTime.UTC().Format(time.RFC3339Nano))
	}
	if query.ParentID != "" {
		where += " AND parent_id = ?"
		args = append(args, query.ParentID)
	}

	q := fmt.Sprintf(`SELECT id, sandbox_id, parent_id, event_type, action_json, result_json, policy_decision_json, timestamp, duration_ns, attributes_json FROM trace_events %s ORDER BY timestamp ASC`, where)

	if query.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// DeleteEvents removes all events for a sandbox.
func (s *SQLiteStore) DeleteEvents(sandboxID string) error {
	_, err := s.db.Exec(`DELETE FROM trace_events WHERE sandbox_id = ?`, sandboxID)
	if err != nil {
		return fmt.Errorf("deleting events: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// scanner is implemented by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanEvent(s scanner) (*types.TraceEvent, error) {
	var (
		event     types.TraceEvent
		parentID  sql.NullString
		actionJSON, resultJSON, policyJSON, attrsJSON sql.NullString
		timestamp string
	)

	err := s.Scan(
		&event.ID,
		&event.SandboxID,
		&parentID,
		&event.EventType,
		&actionJSON,
		&resultJSON,
		&policyJSON,
		&timestamp,
		&event.DurationNs,
		&attrsJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found")
		}
		return nil, fmt.Errorf("scanning event: %w", err)
	}

	if parentID.Valid {
		event.ParentID = parentID.String
	}

	event.Timestamp, err = time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp: %w", err)
	}

	if actionJSON.Valid && actionJSON.String != "" {
		event.Action = &types.Action{}
		if err := json.Unmarshal([]byte(actionJSON.String), event.Action); err != nil {
			return nil, fmt.Errorf("unmarshaling action: %w", err)
		}
	}

	if resultJSON.Valid && resultJSON.String != "" {
		event.Result = &types.ActionResult{}
		if err := json.Unmarshal([]byte(resultJSON.String), event.Result); err != nil {
			return nil, fmt.Errorf("unmarshaling result: %w", err)
		}
	}

	if policyJSON.Valid && policyJSON.String != "" {
		event.PolicyDecision = &types.PolicyDecision{}
		if err := json.Unmarshal([]byte(policyJSON.String), event.PolicyDecision); err != nil {
			return nil, fmt.Errorf("unmarshaling policy decision: %w", err)
		}
	}

	if attrsJSON.Valid && attrsJSON.String != "" {
		event.Attributes = map[string]string{}
		if err := json.Unmarshal([]byte(attrsJSON.String), &event.Attributes); err != nil {
			return nil, fmt.Errorf("unmarshaling attributes: %w", err)
		}
	}

	return &event, nil
}

func scanEvents(rows *sql.Rows) ([]*types.TraceEvent, error) {
	var events []*types.TraceEvent
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return events, nil
}

func jsonMarshalNullable(v any) (sql.NullString, error) {
	if v == nil {
		return sql.NullString{}, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(data), Valid: true}, nil
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
