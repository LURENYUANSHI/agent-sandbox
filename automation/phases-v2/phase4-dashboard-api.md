You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read CLAUDE.md and existing code in pkg/api/ and web/src/lib/api.ts first.

## Your Task: Implement missing Dashboard API endpoints and fix web-API mismatch

### 1. Read web/src/lib/api.ts to understand what the frontend expects
The React frontend calls these endpoints that DON'T exist yet:
- `GET /api/v1/dashboard/stats` → DashboardStats { active_sandboxes, total_actions, denied_actions, avg_response_ms }
- `GET /api/v1/dashboard/activity` → RecentActivity[] { id, type, sandbox_name, description, timestamp }

### 2. pkg/api/dashboard.go
Implement dashboard handlers:
- `handleGetDashboardStats(c *gin.Context)` — aggregate stats from all sandboxes
  - Count active sandboxes (status == running)
  - Sum total actions across all sandbox traces
  - Sum denied actions
  - Calculate average response time from trace durations
- `handleGetRecentActivity(c *gin.Context)` — return last 20 events across all sandboxes
  - Query trace store ordered by timestamp desc, limit 20
  - Map to the format the frontend expects

### 3. Update pkg/api/server.go
Register the new dashboard routes.

### 4. Ensure web/src/lib/api.ts types match the actual API response format
If there are type mismatches between what the Go API returns and what TypeScript expects, fix the TypeScript types to match.

### 5. pkg/api/dashboard_test.go
- Test stats endpoint returns correct counts
- Test activity endpoint returns recent events
- Test with no sandboxes returns zeros

### Verification:
1. `go build ./...`
2. `go test ./pkg/api/... -v -count=1`
3. `cd web && npm run build` (verify TypeScript compiles)
4. Commit: `feat(api): implement dashboard stats and activity endpoints, fix web-API type mismatch`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
