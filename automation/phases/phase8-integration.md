You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

## Repository Info
- **GitHub Repo**: https://github.com/LURENYUANSHI/agent-sandbox
- **Remote**: origin → LURENYUANSHI/agent-sandbox
- **Branch Strategy**: main (production) + develop (integration), all work on feature branches from develop
- **Current working directory**: /c/Users/Administrator/ai-sandbox
- **Feishu notifications**: bash automation/feishu-notify.sh "<event>" "<title>" "<detail>"
- After completing your work, send a Feishu notification: bash automation/feishu-notify.sh "phase_complete" "Phase N completed" "description"



Read CLAUDE.md and docs/development-plan.md. Read existing code to understand what's been built.

## Your Task: Phase 8 - Integration Tests & Docker

### 1. test/integration/sandbox_test.go
Full lifecycle integration test:
```go
func TestSandboxFullLifecycle(t *testing.T) {
    // 1. Create sandbox with default policy
    // 2. Start sandbox
    // 3. Execute allowed action (file read in /tmp) -> should succeed
    // 4. Execute denied action (file delete on /) -> should be denied
    // 5. Verify traces recorded correctly
    // 6. Stop sandbox
    // 7. Replay traces and verify order
}

func TestSandboxConcurrentExecution(t *testing.T) {
    // Launch 10 concurrent actions, verify all traced correctly
}

func TestSandboxPolicyHotReload(t *testing.T) {
    // Start with strict policy, execute denied action
    // Load permissive policy, execute same action -> should succeed
}
```

### 2. test/integration/api_test.go
API integration test (use httptest):
```go
func TestAPIFullWorkflow(t *testing.T) {
    // 1. POST /sandboxes -> create
    // 2. POST /sandboxes/:id/start -> start
    // 3. POST /sandboxes/:id/exec -> execute action
    // 4. GET /sandboxes/:id/traces -> verify traces
    // 5. POST /sandboxes/:id/stop -> stop
    // 6. DELETE /sandboxes/:id -> destroy
}

func TestAPIPolicyValidation(t *testing.T) {
    // POST valid policy -> 200
    // POST invalid policy -> 400 with error details
}

func TestAPIErrorCases(t *testing.T) {
    // GET nonexistent sandbox -> 404
    // Execute on stopped sandbox -> 400
    // Invalid action type -> 400
}
```

### 3. test/fixtures/
- `sample-policy.yaml` - test policy file
- `sample-trace.json` - sample trace data for replay tests

### 4. docker/Dockerfile
Multi-stage build:
```dockerfile
# Stage 1: Build Go backend
FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /bin/agent-sandbox ./cmd/sandbox/
RUN CGO_ENABLED=1 go build -o /bin/agent-sandbox-server ./cmd/server/

# Stage 2: Build React frontend
FROM node:22-alpine AS web-builder
WORKDIR /app
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache sqlite-libs ca-certificates
COPY --from=go-builder /bin/agent-sandbox /usr/local/bin/
COPY --from=go-builder /bin/agent-sandbox-server /usr/local/bin/
COPY --from=web-builder /app/dist /usr/share/agent-sandbox/web
COPY configs/ /etc/agent-sandbox/configs/
EXPOSE 8080
ENTRYPOINT ["agent-sandbox-server"]
CMD ["--port", "8080", "--static-dir", "/usr/share/agent-sandbox/web"]
```

### 5. docker/docker-compose.yaml
```yaml
version: "3.8"
services:
  agent-sandbox:
    build:
      context: ..
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
    volumes:
      - sandbox-data:/var/lib/agent-sandbox
      - ./configs:/etc/agent-sandbox/configs
    environment:
      - SANDBOX_LOG_LEVEL=info
      - SANDBOX_TRACE_DIR=/var/lib/agent-sandbox/traces
volumes:
  sandbox-data:
```

### 6. Update Makefile
Ensure all targets work:
- `make build` - build Go binaries
- `make test` - run unit tests
- `make test-integration` - run integration tests
- `make lint` - run go vet
- `make docker-build` - build Docker image
- `make docker-run` - run with docker-compose

### Verification:
1. `make test` - all unit tests pass
2. `make test-integration` - all integration tests pass
3. `make build` - binaries build
4. `make docker-build` - Docker image builds (if Docker is available)
5. `go vet ./...`
6. Git commit: "feat: add integration tests, Docker multi-stage build, and docker-compose"
