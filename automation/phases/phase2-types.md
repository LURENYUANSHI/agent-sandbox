You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox.

## Repository Info
- **GitHub Repo**: https://github.com/LURENYUANSHI/agent-sandbox
- **Remote**: origin → LURENYUANSHI/agent-sandbox
- **Branch Strategy**: main (production) + develop (integration), all work on feature branches from develop
- **Current working directory**: /c/Users/Administrator/ai-sandbox
- **Feishu notifications**: bash automation/feishu-notify.sh "<event>" "<title>" "<detail>"
- **DO NOT push to remote, DO NOT create PRs, DO NOT create issues.** The orchestrator handles all git remote operations.
- Only make local git commits. The orchestrator will push, create PRs, and merge.
- After completing your work, send a Feishu notification: bash automation/feishu-notify.sh "phase_complete" "Phase N completed" "description"



Read CLAUDE.md and docs/development-plan.md first. Then read existing code to understand what Phase 1 created.

## Your Task: Phase 2 - Core Types & Interfaces

Create all shared types and interfaces. These are the foundation for the entire project.

### 1. pkg/types/action.go
Define action types that represent what an AI agent can do:
```go
package types

type ActionType string
const (
    ActionFileRead    ActionType = "file:read"
    ActionFileWrite   ActionType = "file:write"
    ActionFileDelete  ActionType = "file:delete"
    ActionNetConnect  ActionType = "net:connect"
    ActionNetListen   ActionType = "net:listen"
    ActionNetHTTP     ActionType = "net:http"
    ActionProcExec    ActionType = "proc:exec"
    ActionProcKill    ActionType = "proc:kill"
    ActionShellExec   ActionType = "shell:exec"
)

type Action struct {
    ID        string            `json:"id"`
    Type      ActionType        `json:"type"`
    Params    map[string]string `json:"params"`    // e.g., {"path": "/etc/passwd", "mode": "r"}
    Metadata  map[string]string `json:"metadata"`  // agent-provided context
    Timestamp time.Time         `json:"timestamp"`
}
```

### 2. pkg/types/event.go
Define trace event types:
```go
type EventType string
const (
    EventActionRequested EventType = "action.requested"
    EventPolicyEvaluated EventType = "policy.evaluated"
    EventActionExecuted  EventType = "action.executed"
    EventActionDenied    EventType = "action.denied"
    EventActionFailed    EventType = "action.failed"
    EventSandboxCreated  EventType = "sandbox.created"
    EventSandboxStopped  EventType = "sandbox.stopped"
)

type TraceEvent struct {
    ID          string                 `json:"id"`
    SandboxID   string                 `json:"sandbox_id"`
    ParentID    string                 `json:"parent_id,omitempty"`
    Type        EventType              `json:"type"`
    Action      *Action                `json:"action,omitempty"`
    Result      *ActionResult          `json:"result,omitempty"`
    PolicyDecision *PolicyDecision     `json:"policy_decision,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
    Duration    time.Duration          `json:"duration,omitempty"`
    Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

type ActionResult struct {
    Success bool   `json:"success"`
    Output  string `json:"output,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

### 3. pkg/types/policy.go
Define policy types:
```go
type Effect string
const (
    EffectAllow Effect = "allow"
    EffectDeny  Effect = "deny"
    EffectAudit Effect = "audit"  // allow but log
)

type Rule struct {
    ID          string     `json:"id" yaml:"id"`
    Name        string     `json:"name" yaml:"name"`
    Description string     `json:"description" yaml:"description"`
    Actions     []string   `json:"actions" yaml:"actions"`       // glob patterns: "file:*", "net:http"
    Resources   []string   `json:"resources" yaml:"resources"`   // resource patterns: "/tmp/**", "*.example.com"
    Effect      Effect     `json:"effect" yaml:"effect"`
    Priority    int        `json:"priority" yaml:"priority"`     // higher = evaluated first
    Conditions  map[string]string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

type Policy struct {
    Name        string `json:"name" yaml:"name"`
    Version     string `json:"version" yaml:"version"`
    Description string `json:"description" yaml:"description"`
    DefaultEffect Effect `json:"default_effect" yaml:"default_effect"`
    Rules       []Rule `json:"rules" yaml:"rules"`
}

type PolicyDecision struct {
    Effect    Effect `json:"effect"`
    Rule      *Rule  `json:"rule,omitempty"`
    Reason    string `json:"reason"`
    Timestamp time.Time `json:"timestamp"`
}
```

### 4. Define Interfaces
Create `pkg/types/interfaces.go`:
```go
type Sandbox interface {
    ID() string
    Start(ctx context.Context) error
    Execute(ctx context.Context, action Action) (*ActionResult, error)
    Stop(ctx context.Context) error
    Status() SandboxStatus
}

type PolicyEngine interface {
    Evaluate(ctx context.Context, action Action) (*PolicyDecision, error)
    LoadPolicy(policy Policy) error
    ListPolicies() []Policy
}

type TraceRecorder interface {
    RecordEvent(event TraceEvent) error
    StartSpan(sandboxID string, action Action) (SpanContext, error)
    EndSpan(ctx SpanContext, result *ActionResult) error
}

type TraceStore interface {
    Save(event TraceEvent) error
    GetBySandbox(sandboxID string) ([]TraceEvent, error)
    GetByID(eventID string) (*TraceEvent, error)
    Query(filter TraceFilter) ([]TraceEvent, error)
}

type SandboxStatus string
const (
    StatusCreated  SandboxStatus = "created"
    StatusRunning  SandboxStatus = "running"
    StatusStopped  SandboxStatus = "stopped"
    StatusError    SandboxStatus = "error"
)

type SpanContext struct {
    TraceID   string
    SpanID    string
    StartTime time.Time
}

type TraceFilter struct {
    SandboxID string
    EventType EventType
    StartTime *time.Time
    EndTime   *time.Time
    Limit     int
}
```

### Verification:
1. Run `go build ./...` to verify compilation
2. Run `go vet ./...` to check for issues
3. Git commit with message "feat: define core types, interfaces, and action model"

IMPORTANT: Make sure all types are complete, well-documented, and compile correctly.
