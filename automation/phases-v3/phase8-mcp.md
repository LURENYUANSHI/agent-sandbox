You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

## Your Task: Create MCP (Model Context Protocol) Server integration

This is the highest-value feature — it allows AI agents (Claude, etc.) to use the sandbox as a tool via MCP.

### 1. mcp/server.go
Implement an MCP server that exposes sandbox operations as tools:

```go
package main

// MCP Tool definitions:
// - sandbox_create: Create a new sandbox
// - sandbox_exec: Execute an action in a sandbox (file read/write, network, process)
// - sandbox_traces: Get execution traces for a sandbox
// - sandbox_policy: Validate or load a policy

// The MCP server communicates via stdio (JSON-RPC over stdin/stdout)
// It wraps the existing sandbox, policy, executor, and trace packages
```

Tools to expose:
1. **sandbox_create** — input: {name, policy}, output: {sandbox_id, status}
2. **sandbox_exec** — input: {sandbox_id, action_type, params}, output: {success, output, error, policy_decision}
3. **sandbox_stop** — input: {sandbox_id}, output: {status}
4. **sandbox_traces** — input: {sandbox_id, limit}, output: {events[]}
5. **sandbox_policy_check** — input: {action_type, resource}, output: {effect, reason, rule}

### 2. Implementation approach
Use the MCP protocol (JSON-RPC 2.0 over stdio):
- Read JSON-RPC requests from stdin
- Process tool calls by invoking sandbox packages directly (in-process, no HTTP)
- Write JSON-RPC responses to stdout
- Handle `initialize`, `tools/list`, `tools/call` methods

### 3. mcp/main.go
Entry point that starts the MCP server:
```go
func main() {
    server := NewMCPServer()
    server.Run() // reads stdin, writes stdout
}
```

### 4. mcp/README.md
Usage instructions:
```markdown
# AgentSandbox MCP Server

## Configuration for Claude Desktop
Add to claude_desktop_config.json:
```json
{
  "mcpServers": {
    "agent-sandbox": {
      "command": "agent-sandbox-mcp",
      "args": ["--policy", "configs/default-policy.yaml"]
    }
  }
}
```

## Available Tools
- sandbox_create: Create isolated sandbox for agent operations
- sandbox_exec: Execute actions with policy enforcement
- sandbox_stop: Stop and clean up sandbox
- sandbox_traces: View execution audit trail
- sandbox_policy_check: Pre-check if an action would be allowed
```

### 5. Update Makefile
Add `build-mcp` target.

### 6. Update main README.md
Add MCP section explaining this is usable as an MCP tool for Claude/LLMs.

### Verification:
1. `go build ./mcp/`
2. `go build ./...`
3. Commit: `feat: add MCP server for AI agent integration (Claude Desktop, LLM tools)`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
