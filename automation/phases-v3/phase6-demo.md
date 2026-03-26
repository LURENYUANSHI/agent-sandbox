You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

## Your Task: Create one-click interactive demo

### 1. examples/demo/main.go
Create a self-contained demo program that:
- Creates a sandbox with default policy
- Starts the sandbox
- Runs a sequence of actions demonstrating key features:
  - file:write to /tmp/hello.txt (allowed)
  - file:read from /tmp/hello.txt (allowed)
  - file:delete on /etc/passwd (denied by policy)
  - shell:exec "echo hello" (denied by default policy)
  - net:http to github.com (allowed)
  - proc:exec "ls" (allowed)
- Prints each action's result with colored output (green=allowed, red=denied)
- Shows the trace replay at the end
- Prints stats: X actions, Y allowed, Z denied

### 2. examples/coding-agent/main.go
A simulated coding agent that:
- Creates a sandbox with coding-agent.yaml policy
- Reads a Go source file
- Writes a modified version
- Runs `go fmt` on it
- Shows the trace of what happened

### 3. examples/web-scraper/main.go
A simulated web scraper agent that:
- Creates a sandbox with permissive network but restricted filesystem
- Makes HTTP requests to a few URLs
- Writes results to sandbox-local files
- Attempts to write outside sandbox (denied)

### 4. Update README.md
Add "Try it out" section:
```markdown
## Quick Demo
go run examples/demo/main.go
```

### 5. scripts/demo.sh
```bash
#!/bin/bash
# One-command demo: start server + run demo
echo "Starting AgentSandbox demo..."
go run examples/demo/main.go
```

### Verification:
1. `go build ./examples/demo/`
2. `go build ./examples/coding-agent/` (if it uses available packages)
3. Commit: `feat: add interactive demo examples showcasing sandbox capabilities`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
