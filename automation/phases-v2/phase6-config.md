You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code to find all hardcoded values.

## Your Task: Make hardcoded values configurable

### 1. pkg/config/config.go (NEW)
Create a centralized configuration struct:
```go
type AppConfig struct {
    Server   ServerConfig
    Sandbox  SandboxDefaults
    Executor ExecutorConfig
    Policy   PolicyConfig
}

type SandboxDefaults struct {
    DefaultRootDir     string
    DefaultTimeoutSec  int
    DefaultMaxMemoryMB int
    DefaultMaxDiskMB   int
    DefaultMaxProcs    int
    TracePath          string
}

type ExecutorConfig struct {
    MaxReadSizeMB      int     // default 10
    MaxWriteSizeMB     int     // default 10
    MaxResponseSizeMB  int     // default 5
    HTTPTimeoutSec     int     // default 30
    TCPTimeoutSec      int     // default 10
}

type PolicyConfig struct {
    MaxFileSizeMB       int    // default 100
    PrivilegedPortLimit int    // default 1024
}
```

- `LoadFromEnv() *AppConfig` — load from environment variables with sensible defaults
- `LoadFromFile(path string) (*AppConfig, error)` — load from YAML config file
- Environment variables: `SANDBOX_MAX_READ_SIZE_MB`, `SANDBOX_HTTP_TIMEOUT_SEC`, etc.

### 2. Update consumers
- Update pkg/executor/filesystem.go to use ExecutorConfig instead of hardcoded 10MB
- Update pkg/executor/network.go to use ExecutorConfig instead of hardcoded 5MB/30s/10s
- Update pkg/policy/rules.go to use PolicyConfig instead of hardcoded 100MB/1024
- Update pkg/api/middleware.go CORS to read allowed origins from config or env var `SANDBOX_CORS_ORIGINS`
- Update cmd/server/main.go to load AppConfig

### 3. configs/app.yaml (example config file)
```yaml
server:
  port: 8080
  auth_enabled: false
  cors_origins: ["http://localhost:3000", "http://localhost:5173"]
sandbox:
  default_timeout_sec: 300
  default_max_memory_mb: 512
executor:
  max_read_size_mb: 10
  max_response_size_mb: 5
  http_timeout_sec: 30
policy:
  max_file_size_mb: 100
  privileged_port_limit: 1024
```

### 4. pkg/config/config_test.go
- Test loading from env vars
- Test defaults when no env vars set
- Test YAML loading

### Verification:
1. `go build ./...`
2. `go test ./... -count=1`
3. Commit: `refactor: centralize configuration, replace all hardcoded values with configurable options`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
