You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read CLAUDE.md first. This is v0.2.0 iteration - fixing bugs found in v0.1.0 audit.

## Your Task: Fix all known bugs

### Bug 1: pkg/trace/store.go - Silent JSON unmarshal error
Find where `json.Unmarshal` is called without checking the error. Fix it by checking and handling the error properly.

### Bug 2: pkg/executor/filesystem.go - Unchecked filepath.Abs error
Find where `filepath.Abs` return value error is ignored. Fix it by checking the error.

### Bug 3: pkg/executor/process.go - String split can't handle quoted args
Replace `strings.Fields(s)` with a proper shell argument parser that handles quoted strings. Write a `parseArgs` function that correctly splits `echo "hello world"` into `["echo", "hello world"]`.

### Bug 4: pkg/api/handlers.go - No request body validation
Add validation that request body is not empty before JSON unmarshaling in create/exec endpoints. Return 400 with clear error message.

### Bug 5: pkg/executor/filesystem.go - Path traversal edge cases
Ensure `filepath.Abs` errors are handled, and add additional checks for null bytes in paths.

### Bug 6: CORS credentials leak risk
In pkg/api/middleware.go, set `AllowCredentials: false` unless explicitly configured.

### Verification:
1. `go build ./...`
2. `go test ./... -count=1`
3. `go vet ./...`
4. Commit: `fix: resolve 6 bugs from v0.1.0 audit (trace store, filesystem, process args, API validation, CORS)`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
