You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read CLAUDE.md and existing code in pkg/api/ first.

## Your Task: Add Rate Limiting and Input Validation

### 1. pkg/api/ratelimit.go
Rate limiting middleware:
- `NewRateLimiter(requestsPerSecond float64, burst int) gin.HandlerFunc`
- Use `golang.org/x/time/rate` package (go get it)
- Per-IP rate limiting using a sync.Map of limiters
- Return 429 Too Many Requests when exceeded
- Include Retry-After header
- Clean up stale entries periodically (goroutine with ticker)

### 2. pkg/api/validation.go
Input validation helpers:
- `ValidateCreateSandbox(req)` - validate name (1-128 chars, alphanumeric+dash), policy file path exists
- `ValidateExecAction(req)` - validate action type is known, params are not empty, resource has no null bytes
- `ValidatePolicy(policy)` - validate rule IDs unique, effects valid, action patterns valid
- Return structured validation errors: `{"errors": [{"field": "name", "message": "..."}]}`

### 3. Update pkg/api/handlers.go
- Apply validation to create sandbox, exec action, and validate policy endpoints
- Return 400 with validation error details

### 4. Update pkg/api/server.go
- Add rate limiter config (RPS, burst) to ServerConfig
- Apply rate limiter middleware

### 5. Tests
- pkg/api/ratelimit_test.go: test rate limiting works, test 429 returned
- pkg/api/validation_test.go: test all validation rules, valid and invalid inputs
- Update existing api_test.go if needed

### Verification:
1. `go build ./...`
2. `go test ./pkg/api/... -v -count=1`
3. Commit: `feat(api): add rate limiting and input validation middleware`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
