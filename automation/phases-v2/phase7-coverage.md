You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing test files to understand current coverage.

## Your Task: Increase test coverage for policy (34%→80%) and trace (34%→80%)

### 1. pkg/policy/policy_test.go — ADD MORE TESTS
Current coverage is 34.3%. Add tests for:
- Test glob matching edge cases: empty pattern, pattern with multiple wildcards, exact match vs glob
- Test built-in rules: each rule individually (NoDeleteRoot, NoKillInit, NoDangerousCommands, NoPrivilegedPorts, MaxFileSizeLimit, PathTraversalProtection)
- Test rule priority ordering: higher priority wins over lower
- Test default effect when no rule matches
- Test LoadFromFile with valid and invalid YAML files (use temp files)
- Test LoadFromBytes with malformed YAML
- Test concurrent policy evaluation (goroutines)
- Test policy with empty rules list
- Test policy validation: duplicate rule IDs, invalid effects, invalid action patterns

### 2. pkg/trace/trace_test.go — ADD MORE TESTS
Current coverage is 34.2%. Add tests for:
- Test SQLite store: Save, GetBySandbox, GetByID, Query with filters (time range, event type, limit)
- Test store with empty database
- Test store with many events (100+)
- Test recorder: RecordEvent, StartSpan, EndSpan with duration calculation
- Test nested spans (parent-child via ParentID)
- Test replayer: LoadTrace, Step through all events, Rewind, GetTimeline
- Test replayer with empty trace
- Test OTel export: verify OTLP JSON format, span hierarchy, status codes
- Test concurrent recording from multiple goroutines
- Use in-memory SQLite (":memory:") for fast tests

### 3. Run coverage and verify
```bash
go test ./pkg/policy/... -cover -count=1
go test ./pkg/trace/... -cover -count=1
```
Both must be >= 75% (aim for 80%+).

### Verification:
1. `go test ./... -count=1` — all tests pass
2. `go test ./pkg/policy/... -cover` — >= 75%
3. `go test ./pkg/trace/... -cover` — >= 75%
4. Commit: `test: increase policy coverage to 80%+ and trace coverage to 80%+`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
