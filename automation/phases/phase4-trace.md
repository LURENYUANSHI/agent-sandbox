You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

## Repository Info
- **GitHub Repo**: https://github.com/LURENYUANSHI/agent-sandbox
- **Remote**: origin → LURENYUANSHI/agent-sandbox
- **Branch Strategy**: main (production) + develop (integration), all work on feature branches from develop
- **Current working directory**: /c/Users/Administrator/ai-sandbox
- **Feishu notifications**: bash automation/feishu-notify.sh "<event>" "<title>" "<detail>"
- After completing your work, send a Feishu notification: bash automation/feishu-notify.sh "phase_complete" "Phase N completed" "description"



Read CLAUDE.md and docs/development-plan.md first. Read existing code in pkg/types/ and pkg/policy/ to understand the foundation.

## Your Task: Phase 4 - Trace System

Build the complete trace recording, storage, and replay system.

### 1. pkg/trace/recorder.go
Implement `TraceRecorder` interface:
- `NewRecorder(store TraceStore) *Recorder`
- `RecordEvent(event)` - save event to store with auto-generated ID and timestamp
- `StartSpan(sandboxID, action)` - create a new span for an action execution
- `EndSpan(ctx, result)` - close span, calculate duration, save to store
- Support nested spans (parent-child via ParentID)
- Thread-safe for concurrent recording

### 2. pkg/trace/store.go
SQLite-based trace storage:
- `NewSQLiteStore(dbPath string) (*SQLiteStore, error)` - auto-create tables
- Schema:
  ```sql
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
  CREATE INDEX idx_sandbox_id ON trace_events(sandbox_id);
  CREATE INDEX idx_event_type ON trace_events(event_type);
  CREATE INDEX idx_timestamp ON trace_events(timestamp);
  ```
- Implement all TraceStore interface methods
- Use `database/sql` with `github.com/mattn/go-sqlite3` driver
- JSON encode/decode for nested objects

### 3. pkg/trace/replayer.go
Trace replay engine:
- `NewReplayer(store TraceStore) *Replayer`
- `LoadTrace(sandboxID string) (*Trace, error)` - load full trace as tree structure
- `Step(trace *Trace) (*TraceEvent, bool)` - iterate one event at a time
- `Rewind(trace *Trace)` - reset to beginning
- `GetTimeline(sandboxID string) ([]TimelineEntry, error)` - flat chronological view
- Build parent-child tree from flat events

### 4. pkg/trace/otel.go
OpenTelemetry export:
- Convert TraceEvent to OTLP spans
- `ExportToOTLP(events []TraceEvent) ([]byte, error)` - serialize to OTLP JSON
- Map our event types to OTel span attributes
- Support for span status (OK, ERROR based on ActionResult)

### 5. pkg/trace/trace_test.go
Tests:
- Test recorder creates events with correct IDs and timestamps
- Test span lifecycle (start -> end with duration)
- Test nested spans (parent-child relationships)
- Test SQLite store CRUD operations
- Test query filtering (by sandbox, event type, time range)
- Test replayer step-through and rewind
- Test OTel export format
- Use in-memory SQLite (":memory:") for fast tests

### Verification:
1. `go test ./pkg/trace/... -v -count=1` - all tests pass
2. `go test ./pkg/trace/... -cover` - check coverage
3. `go vet ./...` - no issues
4. Git commit: "feat: implement trace recording, SQLite storage, replay engine, and OTel export"
