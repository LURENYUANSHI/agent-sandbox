You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code in pkg/trace/audit.go.

## Your Task: Add audit log rotation and API coverage improvement

### 1. pkg/trace/audit.go — Add rotation
- `SetRetentionDays(days int)` — configure how long to keep audit entries
- `Rotate() (int64, error)` — delete entries older than retention period, return count deleted
- `StartAutoRotation(interval time.Duration)` — background goroutine that runs Rotate periodically
- `StopAutoRotation()` — stop the background goroutine
- `GetStats() AuditStats` — return total entries, oldest entry date, disk usage estimate

### 2. Update cmd/server/main.go
- Add `--audit-retention-days` flag (default: 90)
- Start auto-rotation on server start (check daily)
- Stop on shutdown

### 3. pkg/api/api_test.go — Improve API test coverage
Add tests for endpoints that currently lack coverage:
- Test dashboard stats endpoint with multiple sandboxes
- Test dashboard activity endpoint
- Test audit endpoint with query params (filter by sandbox_id, effect, time range)
- Test auth token endpoint
- Test rate limiter returns 429
- Test validation errors return structured 400 responses
- Test WebSocket upgrade

Target: API coverage from 69% to 80%+

### 4. pkg/trace/audit_test.go — Add rotation tests
- Test Rotate deletes old entries
- Test Rotate keeps recent entries
- Test GetStats returns correct counts
- Test auto-rotation starts and stops cleanly

### Verification:
1. `go build ./...`
2. `go test ./... -count=1`
3. `go test ./pkg/api/... -cover` — should be >= 75%
4. Commit: `feat(audit): add log rotation with configurable retention, improve API test coverage`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
