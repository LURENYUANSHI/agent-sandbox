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
