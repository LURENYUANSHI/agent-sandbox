package trace

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAuditLogger_LogAndQuery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Log some decisions
	err = logger.LogDecision("sb-1", "file:write", "/tmp/test.txt", "allow", "rule-1", "matched allow rule", "user-1")
	if err != nil {
		t.Fatalf("LogDecision: %v", err)
	}
	err = logger.LogDecision("sb-1", "proc:exec", "/bin/bash", "deny", "rule-2", "process denied", "user-1")
	if err != nil {
		t.Fatalf("LogDecision: %v", err)
	}
	err = logger.LogDecision("sb-2", "file:read", "/etc/passwd", "deny", "rule-3", "path denied", "user-2")
	if err != nil {
		t.Fatalf("LogDecision: %v", err)
	}

	// Query all
	entries, err := logger.QueryAuditLog(AuditFilter{})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Query by sandbox_id
	entries, err = logger.QueryAuditLog(AuditFilter{SandboxID: "sb-1"})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for sb-1, got %d", len(entries))
	}

	// Query by effect
	entries, err = logger.QueryAuditLog(AuditFilter{Effect: "deny"})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 deny entries, got %d", len(entries))
	}

	// Query by action_type
	entries, err = logger.QueryAuditLog(AuditFilter{ActionType: "file:write"})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 file:write entry, got %d", len(entries))
	}

	// Query with limit
	entries, err = logger.QueryAuditLog(AuditFilter{Limit: 1})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry with limit, got %d", len(entries))
	}
}

func TestAuditLogger_QueryByTimeRange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_time_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Log a decision
	err = logger.LogDecision("sb-1", "file:read", "/tmp/a.txt", "allow", "r1", "ok", "u1")
	if err != nil {
		t.Fatalf("LogDecision: %v", err)
	}

	now := time.Now().UTC()

	// Query with start_time in the past should find it
	entries, err := logger.QueryAuditLog(AuditFilter{
		StartTime: now.Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	// Query with start_time in the future should find nothing
	entries, err = logger.QueryAuditLog(AuditFilter{
		StartTime: now.Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestAuditLogger_EntryFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_fields_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	err = logger.LogDecision("sb-99", "shell:exec", "ls -la", "allow", "rule-x", "shell allowed", "admin")
	if err != nil {
		t.Fatalf("LogDecision: %v", err)
	}

	entries, err := logger.QueryAuditLog(AuditFilter{SandboxID: "sb-99"})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.SandboxID != "sb-99" {
		t.Errorf("SandboxID: got %s, want sb-99", e.SandboxID)
	}
	if e.ActionType != "shell:exec" {
		t.Errorf("ActionType: got %s, want shell:exec", e.ActionType)
	}
	if e.Resource != "ls -la" {
		t.Errorf("Resource: got %s, want ls -la", e.Resource)
	}
	if e.Effect != "allow" {
		t.Errorf("Effect: got %s, want allow", e.Effect)
	}
	if e.RuleID != "rule-x" {
		t.Errorf("RuleID: got %s, want rule-x", e.RuleID)
	}
	if e.Reason != "shell allowed" {
		t.Errorf("Reason: got %s, want shell allowed", e.Reason)
	}
	if e.UserID != "admin" {
		t.Errorf("UserID: got %s, want admin", e.UserID)
	}
	if e.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if e.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

func TestAuditLogger_RotateDeletesOldEntries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_rotate_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Insert an old entry directly with a timestamp 100 days ago
	oldTime := time.Now().UTC().AddDate(0, 0, -100)
	_, err = logger.db.Exec(
		`INSERT INTO audit_log (timestamp, sandbox_id, action_type, resource, effect, rule_id, reason, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		oldTime, "sb-old", "file:read", "/old", "allow", "r1", "old entry", "u1",
	)
	if err != nil {
		t.Fatalf("insert old entry: %v", err)
	}

	// Insert a recent entry
	err = logger.LogDecision("sb-new", "file:write", "/new", "allow", "r2", "new entry", "u2")
	if err != nil {
		t.Fatalf("LogDecision: %v", err)
	}

	logger.SetRetentionDays(90)
	deleted, err := logger.Rotate()
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	entries, _ := logger.QueryAuditLog(AuditFilter{})
	if len(entries) != 1 {
		t.Errorf("expected 1 remaining entry, got %d", len(entries))
	}
	if entries[0].SandboxID != "sb-new" {
		t.Errorf("expected remaining entry to be sb-new, got %s", entries[0].SandboxID)
	}
}

func TestAuditLogger_RotateKeepsRecentEntries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_rotate_keep_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Insert only recent entries
	logger.LogDecision("sb-1", "file:read", "/a", "allow", "r1", "", "u1")
	logger.LogDecision("sb-2", "file:write", "/b", "deny", "r2", "", "u2")

	logger.SetRetentionDays(90)
	deleted, err := logger.Rotate()
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	entries, _ := logger.QueryAuditLog(AuditFilter{})
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestAuditLogger_GetStats(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_stats_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Empty DB
	stats := logger.GetStats()
	if stats.TotalEntries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.TotalEntries)
	}

	// Add entries
	logger.LogDecision("sb-1", "file:read", "/a", "allow", "r1", "", "u1")
	logger.LogDecision("sb-2", "file:write", "/b", "deny", "r2", "", "u2")
	logger.LogDecision("sb-3", "proc:exec", "/c", "allow", "r3", "", "u3")

	stats = logger.GetStats()
	if stats.TotalEntries != 3 {
		t.Errorf("expected 3 entries, got %d", stats.TotalEntries)
	}
	if stats.OldestEntry.IsZero() {
		t.Error("expected non-zero oldest entry")
	}
	if stats.DiskUsageBytes != 3*200 {
		t.Errorf("expected disk usage %d, got %d", 3*200, stats.DiskUsageBytes)
	}
}

func TestAuditLogger_AutoRotation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_autorotate_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Start and stop should not panic or block
	logger.StartAutoRotation(50 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	logger.StopAutoRotation()

	// Calling StopAutoRotation again should be safe (idempotent)
	logger.StopAutoRotation()
}

func TestAuditLogger_CombinedFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_combo_test.db")
	logger, err := NewAuditLogger(dbPath)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	logger.LogDecision("sb-1", "file:write", "/a", "allow", "r1", "", "u1")
	logger.LogDecision("sb-1", "file:write", "/b", "deny", "r2", "", "u1")
	logger.LogDecision("sb-2", "file:write", "/c", "deny", "r2", "", "u2")

	// sandbox + effect filter
	entries, err := logger.QueryAuditLog(AuditFilter{
		SandboxID: "sb-1",
		Effect:    "deny",
	})
	if err != nil {
		t.Fatalf("QueryAuditLog: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for sb-1+deny, got %d", len(entries))
	}
}
