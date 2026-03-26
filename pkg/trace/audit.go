package trace

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// AuditEntry represents a single row in the audit log.
type AuditEntry struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	SandboxID  string    `json:"sandbox_id"`
	ActionType string    `json:"action_type"`
	Resource   string    `json:"resource"`
	Effect     string    `json:"effect"`
	RuleID     string    `json:"rule_id"`
	Reason     string    `json:"reason"`
	UserID     string    `json:"user_id"`
}

// AuditFilter defines query parameters for filtering audit log entries.
type AuditFilter struct {
	SandboxID  string    `json:"sandbox_id,omitempty"`
	ActionType string    `json:"action_type,omitempty"`
	Effect     string    `json:"effect,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Limit      int       `json:"limit,omitempty"`
}

// AuditStats contains aggregate information about the audit log.
type AuditStats struct {
	TotalEntries  int64     `json:"total_entries"`
	OldestEntry   time.Time `json:"oldest_entry"`
	DiskUsageBytes int64    `json:"disk_usage_bytes"`
}

// AuditLogger provides persistent audit logging to SQLite for compliance.
type AuditLogger struct {
	db            *sql.DB
	retentionDays int
	stopCh        chan struct{}
	stopOnce      sync.Once
}

// NewAuditLogger opens (or creates) the SQLite database and initializes the audit_log table.
func NewAuditLogger(dbPath string) (*AuditLogger, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open audit db %s: %w", dbPath, err)
	}
	if err := migrateAudit(db); err != nil {
		db.Close()
		return nil, err
	}
	return &AuditLogger{db: db, retentionDays: 90, stopCh: make(chan struct{})}, nil
}

func migrateAudit(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   DATETIME NOT NULL,
			sandbox_id  TEXT NOT NULL,
			action_type TEXT NOT NULL,
			resource    TEXT,
			effect      TEXT NOT NULL,
			rule_id     TEXT,
			reason      TEXT,
			user_id     TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_audit_sandbox ON audit_log(sandbox_id);
		CREATE INDEX IF NOT EXISTS idx_audit_effect ON audit_log(effect);
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
	`)
	if err != nil {
		return fmt.Errorf("migrate audit_log: %w", err)
	}
	return nil
}

// LogDecision records a policy decision (allow or deny) to the audit log.
func (a *AuditLogger) LogDecision(sandboxID, actionType, resource, effect, ruleID, reason, userID string) error {
	_, err := a.db.Exec(
		`INSERT INTO audit_log (timestamp, sandbox_id, action_type, resource, effect, rule_id, reason, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC(), sandboxID, actionType, resource, effect, ruleID, reason, userID,
	)
	if err != nil {
		return fmt.Errorf("log audit decision: %w", err)
	}
	return nil
}

// QueryAuditLog retrieves audit entries matching the given filter.
func (a *AuditLogger) QueryAuditLog(filter AuditFilter) ([]AuditEntry, error) {
	query := `SELECT id, timestamp, sandbox_id, action_type, resource, effect, rule_id, reason, user_id
		FROM audit_log WHERE 1=1`
	var args []interface{}

	if filter.SandboxID != "" {
		query += " AND sandbox_id = ?"
		args = append(args, filter.SandboxID)
	}
	if filter.ActionType != "" {
		query += " AND action_type = ?"
		args = append(args, filter.ActionType)
	}
	if filter.Effect != "" {
		query += " AND effect = ?"
		args = append(args, filter.Effect)
	}
	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var resource, ruleID, reason, userID sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.SandboxID, &e.ActionType,
			&resource, &e.Effect, &ruleID, &reason, &userID); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		e.Resource = resource.String
		e.RuleID = ruleID.String
		e.Reason = reason.String
		e.UserID = userID.String
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// SetRetentionDays configures how long to keep audit entries.
func (a *AuditLogger) SetRetentionDays(days int) {
	if days > 0 {
		a.retentionDays = days
	}
}

// Rotate deletes audit entries older than the retention period and returns the count deleted.
func (a *AuditLogger) Rotate() (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -a.retentionDays)
	result, err := a.db.Exec(`DELETE FROM audit_log WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("rotate audit log: %w", err)
	}
	return result.RowsAffected()
}

// StartAutoRotation starts a background goroutine that runs Rotate at the given interval.
func (a *AuditLogger) StartAutoRotation(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.Rotate()
			case <-a.stopCh:
				return
			}
		}
	}()
}

// StopAutoRotation stops the background rotation goroutine.
func (a *AuditLogger) StopAutoRotation() {
	a.stopOnce.Do(func() {
		close(a.stopCh)
	})
}

// GetStats returns aggregate information about the audit log.
func (a *AuditLogger) GetStats() AuditStats {
	var stats AuditStats

	a.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&stats.TotalEntries)

	var oldest sql.NullString
	a.db.QueryRow(`SELECT MIN(timestamp) FROM audit_log`).Scan(&oldest)
	if oldest.Valid && oldest.String != "" {
		// SQLite MIN() on a Go time.Time value produces "2006-01-02 15:04:05.999999 +0000 UTC"
		formats := []string{
			"2006-01-02 15:04:05.999999 +0000 UTC",
			"2006-01-02 15:04:05.999999999 +0000 UTC",
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, oldest.String); err == nil {
				stats.OldestEntry = t
				break
			}
		}
	}

	// Estimate disk usage: ~200 bytes per row is a reasonable approximation for this schema.
	stats.DiskUsageBytes = stats.TotalEntries * 200

	return stats
}

// Close closes the underlying database.
func (a *AuditLogger) Close() error {
	a.StopAutoRotation()
	return a.db.Close()
}
