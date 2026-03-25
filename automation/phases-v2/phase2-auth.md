You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read CLAUDE.md and existing code in pkg/api/ first.

## Your Task: Add JWT Authentication to API

### 1. Install dependency
```
go get github.com/golang-jwt/jwt/v5
```

### 2. pkg/api/auth.go
Create authentication middleware:
- `NewAuthMiddleware(secret string, enabled bool) gin.HandlerFunc`
- If `enabled=false`, skip auth (for dev mode)
- Extract Bearer token from Authorization header
- Validate JWT signature and expiration
- Set user claims in gin context
- Return 401 for invalid/missing token
- Health endpoint `/api/v1/health` always bypasses auth

### 3. pkg/api/token.go
Token generation helper:
- `GenerateToken(secret string, userID string, expiry time.Duration) (string, error)`
- Include standard claims: sub, iat, exp
- Include custom claim: role (admin/viewer)

### 4. Update pkg/api/server.go
- Add `AuthEnabled bool` and `AuthSecret string` to ServerConfig
- Apply auth middleware to all routes except health
- Add `POST /api/v1/auth/token` endpoint for token generation (dev/testing)

### 5. Update cmd/server/main.go
- Add `--auth-enabled` flag (default: false)
- Add `--auth-secret` flag (default: auto-generated random string)
- Print the generated secret and a sample token on startup when auth is enabled

### 6. pkg/api/auth_test.go
Tests:
- Request without token → 401
- Request with invalid token → 401
- Request with expired token → 401
- Request with valid token → passes through
- Health endpoint without token → 200
- Auth disabled → all requests pass

### Verification:
1. `go build ./...`
2. `go test ./pkg/api/... -v -count=1`
3. Commit: `feat(api): add JWT authentication middleware with token generation`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
