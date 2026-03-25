You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code in pkg/sandbox/ and pkg/executor/.

## Your Task: Enforce resource limits and add audit logging

### 1. pkg/sandbox/resource.go
Resource limit enforcement:
- `type ResourceMonitor struct` — tracks resource usage per sandbox
- `NewResourceMonitor(config Config) *ResourceMonitor`
- `CheckDiskUsage(sandboxRoot string) (int64, error)` — calculate total disk usage in bytes
- `CheckProcessCount(sandboxID string) (int, error)` — count running processes for sandbox
- `EnforceLimit(action Action, config Config) error` — check if action would exceed limits:
  - file:write → check disk usage + new file size <= MaxDiskMB
  - proc:exec → check process count < MaxProcesses
  - Return descriptive error if limit exceeded

### 2. Update pkg/sandbox/sandbox.go
- Create ResourceMonitor during sandbox Start
- Call ResourceMonitor.EnforceLimit before executor runs, after policy check
- Record resource limit violations as trace events (new EventType: "resource.exceeded")

### 3. pkg/trace/audit.go
Audit logging for compliance:
- `type AuditLogger struct` — persistent audit log
- `NewAuditLogger(dbPath string) *AuditLogger` — uses SQLite table `audit_log`
- Schema: id, timestamp, sandbox_id, action_type, resource, effect, rule_id, reason, user_id
- `LogDecision(sandboxID, action, decision PolicyDecision, userID string) error`
- `QueryAuditLog(filter AuditFilter) ([]AuditEntry, error)`
- All policy decisions (allow AND deny) get logged

### 4. Update pkg/api/handlers.go
- Add `GET /api/v1/audit` endpoint to query audit logs
- Support query params: sandbox_id, action_type, effect, start_time, end_time, limit

### 5. pkg/types/event.go
Add new event type:
```go
EventResourceExceeded EventType = "resource.exceeded"
```

### 6. Tests
- pkg/sandbox/resource_test.go: test disk usage check, process count, limit enforcement
- pkg/trace/audit_test.go: test audit log save and query

### Verification:
1. `go build ./...`
2. `go test ./... -count=1`
3. Commit: `feat: enforce resource limits and add persistent audit logging for compliance`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
