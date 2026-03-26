You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code in cmd/sandbox/main.go.

## Your Task: Add CLI integration tests

### 1. cmd/sandbox/main_test.go
Test all CLI subcommands by capturing output:
```go
func TestCLI(t *testing.T) {
    // Helper to run CLI command and capture output
    runCLI := func(args ...string) (string, error) {
        // Reset cobra root command, set args, capture output
    }
}
```

Tests:
- `agent-sandbox version` → prints version string
- `agent-sandbox create --name test` → creates sandbox, prints ID
- `agent-sandbox list` → shows created sandbox
- `agent-sandbox list --output json` → valid JSON output
- `agent-sandbox start <id>` → starts sandbox
- `agent-sandbox exec <id> file:read --param path=/tmp/test` → executes action
- `agent-sandbox trace <id>` → shows trace events
- `agent-sandbox stop <id>` → stops sandbox
- `agent-sandbox policy validate configs/default-policy.yaml` → validates policy
- `agent-sandbox --help` → shows help text
- Unknown command → error message
- Invalid sandbox ID → error message

### 2. Test approach
Since CLI uses in-process sandbox registry, tests can create real sandboxes in temp directories and verify full lifecycle through CLI interface.

### Verification:
1. `go build ./...`
2. `go test ./cmd/sandbox/... -v -count=1`
3. `go test ./cmd/sandbox/... -cover` — should be >= 50%
4. Commit: `test(cli): add integration tests for all CLI subcommands`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
