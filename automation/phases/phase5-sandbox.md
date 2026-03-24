You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

Read CLAUDE.md, docs/development-plan.md, and ALL existing code in pkg/ to understand the full context.

## Your Task: Phase 5 - Sandbox Runtime & Executor

Build the core sandbox runtime and the action executor.

### 1. pkg/sandbox/config.go
Sandbox configuration:
```go
type Config struct {
    ID              string            `json:"id" yaml:"id"`
    Name            string            `json:"name" yaml:"name"`
    RootDir         string            `json:"root_dir" yaml:"root_dir"`        // sandbox filesystem root
    PolicyFile      string            `json:"policy_file" yaml:"policy_file"`  // path to policy YAML
    MaxMemoryMB     int               `json:"max_memory_mb" yaml:"max_memory_mb"`
    MaxCPUPercent   int               `json:"max_cpu_percent" yaml:"max_cpu_percent"`
    MaxDiskMB       int               `json:"max_disk_mb" yaml:"max_disk_mb"`
    MaxProcesses    int               `json:"max_processes" yaml:"max_processes"`
    TimeoutSeconds  int               `json:"timeout_seconds" yaml:"timeout_seconds"`
    NetworkEnabled  bool              `json:"network_enabled" yaml:"network_enabled"`
    AllowedPaths    []string          `json:"allowed_paths" yaml:"allowed_paths"`
    DeniedPaths     []string          `json:"denied_paths" yaml:"denied_paths"`
    Environment     map[string]string `json:"environment" yaml:"environment"`
    TraceEnabled    bool              `json:"trace_enabled" yaml:"trace_enabled"`
    TracePath       string            `json:"trace_path" yaml:"trace_path"`      // SQLite DB path
}
```
- `DefaultConfig() Config` - sensible defaults
- `Validate() error` - validate configuration

### 2. pkg/sandbox/sandbox.go
Implement Sandbox interface:
- `NewSandbox(cfg Config, policyEngine PolicyEngine, recorder TraceRecorder) (*SandboxInstance, error)`
- `Start(ctx)` - prepare sandbox environment (create root dir, init trace DB)
- `Execute(ctx, action)`:
  1. Record action.requested event
  2. Evaluate action against policy engine
  3. Record policy.evaluated event
  4. If denied: record action.denied, return error
  5. If allowed: dispatch to appropriate executor
  6. Record action.executed or action.failed
  7. Return result
- `Stop(ctx)` - cleanup, record sandbox.stopped
- `Status()` - return current status
- `GetTraces()` - return all trace events for this sandbox
- Thread-safe with mutex

### 3. pkg/executor/executor.go
Main executor that dispatches to specific handlers:
- `NewExecutor(config Config) *Executor`
- `Execute(ctx, sandbox, action) (*ActionResult, error)` - dispatch based on ActionType
- Central timeout enforcement
- Capture stdout/stderr for all executed actions

### 4. pkg/executor/filesystem.go
Filesystem operations executor:
- `ExecuteFileRead(ctx, action) (*ActionResult, error)` - read file, return content
- `ExecuteFileWrite(ctx, action) (*ActionResult, error)` - write file
- `ExecuteFileDelete(ctx, action) (*ActionResult, error)` - delete file
- Path validation: resolve symlinks, ensure path is within sandbox root
- Size limits enforcement

### 5. pkg/executor/network.go
Network operations executor:
- `ExecuteNetHTTP(ctx, action) (*ActionResult, error)` - HTTP request with timeout
- `ExecuteNetConnect(ctx, action) (*ActionResult, error)` - TCP connection check
- Domain/IP allowlist enforcement
- Response size limits

### 6. pkg/executor/process.go
Process execution:
- `ExecuteProcess(ctx, action) (*ActionResult, error)` - run process with resource limits
- `ExecuteShell(ctx, action) (*ActionResult, error)` - run shell command (more restricted)
- Capture stdout/stderr
- Enforce timeout
- Kill process on timeout or cancellation

### 7. Tests
- `pkg/sandbox/sandbox_test.go`:
  - Test full lifecycle: create -> start -> execute -> stop
  - Test policy enforcement (denied actions return error)
  - Test trace recording during execution
  - Test concurrent execution
- `pkg/executor/executor_test.go`:
  - Test filesystem operations (read, write, delete)
  - Test path validation (no escape from sandbox root)
  - Test process execution with timeout
  - Test network operations

Use temporary directories for sandbox roots in tests.

### Verification:
1. `go test ./pkg/sandbox/... ./pkg/executor/... -v -count=1`
2. `go test ./... -cover` - full project coverage check
3. `go vet ./...`
4. Git commit: "feat: implement sandbox runtime with executor, filesystem/network/process isolation"
