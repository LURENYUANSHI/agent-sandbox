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



Read CLAUDE.md and docs/development-plan.md first. Read the existing types in pkg/types/ to understand the interfaces you must implement.

## Your Task: Phase 3 - Policy Engine

Build the complete policy evaluation engine.

### 1. pkg/policy/engine.go
Implement `PolicyEngine` interface:
- `NewPolicyEngine(defaultPolicy Policy) *Engine`
- `Evaluate(ctx, action)` - match action against rules by priority order, return first match or default effect
- Support glob matching for action types ("file:*" matches "file:read", "file:write")
- Support path glob matching for resources ("/tmp/**" matches "/tmp/foo/bar")
- Thread-safe (use sync.RWMutex)

### 2. pkg/policy/rules.go
Built-in safety rules that are ALWAYS enforced regardless of user policy:
- `NoDeleteRoot` - deny file:delete on / or system paths
- `NoKillInit` - deny proc:kill on PID 1
- `NoDangerousCommands` - deny shell:exec for rm -rf /, mkfs, dd if=/dev/zero, etc.
- `NoPrivilegedPorts` - deny net:listen on ports < 1024
- `MaxFileSizeLimit` - deny file:write if size > configurable limit
- `PathTraversalProtection` - deny if path contains ../ outside sandbox root

### 3. pkg/policy/parser.go
- Parse YAML policy files into Policy struct
- Validate policy (no conflicting rules, valid effects, valid action patterns)
- `LoadFromFile(path string) (*Policy, error)`
- `LoadFromBytes(data []byte) (*Policy, error)`

### 4. configs/default-policy.yaml
```yaml
name: default
version: "1.0"
description: "Default restrictive policy - deny by default, allow specific safe operations"
default_effect: deny
rules:
  - id: allow-tmp-read-write
    name: "Allow /tmp read/write"
    actions: ["file:read", "file:write"]
    resources: ["/tmp/**", "/sandbox/**"]
    effect: allow
    priority: 10
  - id: allow-http-outbound
    name: "Allow HTTP to safe domains"
    actions: ["net:http"]
    resources: ["*.github.com", "*.githubusercontent.com", "*.npmjs.org", "*.pypi.org"]
    effect: allow
    priority: 10
  - id: allow-local-process
    name: "Allow safe local commands"
    actions: ["proc:exec"]
    resources: ["ls", "cat", "echo", "grep", "find", "head", "tail", "wc"]
    effect: allow
    priority: 10
  - id: deny-all-shell
    name: "Deny direct shell execution"
    actions: ["shell:exec"]
    resources: ["*"]
    effect: deny
    priority: 100
```

### 5. Example policies in configs/examples/
Create strict.yaml, permissive.yaml, and coding-agent.yaml with appropriate rules.

### 6. pkg/policy/policy_test.go
Comprehensive table-driven tests:
- Test basic allow/deny evaluation
- Test glob matching (action patterns, resource patterns)
- Test priority ordering (higher priority wins)
- Test built-in rules cannot be overridden
- Test default effect when no rule matches
- Test YAML parsing and validation
- Test thread safety (concurrent evaluation)
- Aim for >90% coverage

### Verification:
1. `go test ./pkg/policy/... -v -count=1` - all tests pass
2. `go test ./pkg/policy/... -cover` - check coverage
3. `go vet ./...` - no issues
4. Git commit: "feat: implement policy engine with YAML parsing, built-in rules, and glob matching"
