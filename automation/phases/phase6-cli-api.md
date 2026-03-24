You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

Read CLAUDE.md, docs/development-plan.md, and ALL existing code to understand the full context.

## Your Task: Phase 6 - CLI & API Server

### 1. Install dependencies
```
go get github.com/spf13/cobra
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors
go get github.com/google/uuid
```

### 2. cmd/sandbox/main.go
CLI using cobra with subcommands:

```
agent-sandbox create [--name NAME] [--policy POLICY_FILE] [--root DIR]
agent-sandbox start <sandbox-id>
agent-sandbox exec <sandbox-id> <action-type> [--param key=value]...
agent-sandbox stop <sandbox-id>
agent-sandbox list [--status STATUS]
agent-sandbox trace <sandbox-id> [--format json|table|otel]
agent-sandbox replay <sandbox-id> [--step]
agent-sandbox policy validate <policy-file>
agent-sandbox version
```

Each command should:
- Print clear output (tables for list, JSON for trace)
- Handle errors gracefully with helpful messages
- Support --output json flag for machine-readable output

### 3. pkg/api/server.go
Gin HTTP server:
- `NewServer(config ServerConfig) *Server`
- CORS middleware (allow localhost origins for dev)
- Request logging middleware
- Error handling middleware
- Graceful shutdown
- WebSocket endpoint for real-time trace streaming

### 4. pkg/api/handlers.go
REST API handlers:
```
POST   /api/v1/sandboxes              - Create sandbox
GET    /api/v1/sandboxes              - List sandboxes
GET    /api/v1/sandboxes/:id          - Get sandbox details
POST   /api/v1/sandboxes/:id/start    - Start sandbox
POST   /api/v1/sandboxes/:id/exec     - Execute action in sandbox
POST   /api/v1/sandboxes/:id/stop     - Stop sandbox
DELETE /api/v1/sandboxes/:id          - Destroy sandbox
GET    /api/v1/sandboxes/:id/traces   - Get trace events
POST   /api/v1/sandboxes/:id/replay   - Start replay session
GET    /api/v1/sandboxes/:id/replay/next - Get next replay event
POST   /api/v1/policies/validate      - Validate a policy file
GET    /api/v1/health                 - Health check
WS     /api/v1/sandboxes/:id/ws       - WebSocket for real-time traces
```

Use proper HTTP status codes, JSON request/response bodies.

### 5. pkg/api/middleware.go
- RequestID middleware (add X-Request-ID header)
- Logger middleware (log method, path, status, duration)
- Recovery middleware (catch panics)
- CORS middleware

### 6. cmd/server/main.go
API server entry point:
- Parse config from environment variables or flags
- Start server on configurable port (default 8080)
- Graceful shutdown on SIGINT/SIGTERM

### 7. pkg/api/api_test.go
- Test each endpoint with httptest
- Test error cases (404, 400, etc.)
- Test WebSocket connection

### Verification:
1. `go build ./cmd/sandbox/` - CLI builds
2. `go build ./cmd/server/` - Server builds
3. `go test ./pkg/api/... -v -count=1` - API tests pass
4. Run `./cmd/sandbox/sandbox --help` to verify CLI
5. `go vet ./...`
6. Git commit: "feat: implement CLI with cobra and REST API with gin, WebSocket trace streaming"
