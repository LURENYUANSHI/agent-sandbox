# AgentSandbox MCP Server

MCP (Model Context Protocol) server that exposes AgentSandbox operations as tools for AI agents like Claude.

## Build

```bash
make build-mcp
```

The binary is output to `bin/agent-sandbox-mcp`.

## Configuration for Claude Desktop

Add to your `claude_desktop_config.json`:

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

Or with an absolute path:

```json
{
  "mcpServers": {
    "agent-sandbox": {
      "command": "/path/to/bin/agent-sandbox-mcp",
      "args": ["--policy", "/path/to/configs/default-policy.yaml"]
    }
  }
}
```

## Available Tools

### sandbox_create

Create and start a new isolated sandbox for executing agent operations.

**Input:**
- `name` (string, required) — Human-readable name for the sandbox.
- `policy` (string, optional) — Path to a YAML policy file. Uses the server default if omitted.

**Output:** `{sandbox_id, name, status, root_dir}`

### sandbox_exec

Execute an action inside a sandbox with policy enforcement.

**Input:**
- `sandbox_id` (string, required) — ID of the target sandbox.
- `action_type` (string, required) — One of: `file.read`, `file.write`, `file.delete`, `net.http`, `net.connect`, `process.exec`, `shell.exec`
- `params` (object, required) — Action parameters:
  - File operations: `{path, content}`
  - Network HTTP: `{url, method}`
  - Network TCP: `{host, port}`
  - Process/Shell: `{command, args}`

**Output:** `{success, output, error, exit_code}` or `{success: false, error, policy_decision}` if denied.

### sandbox_stop

Stop a running sandbox and release its resources.

**Input:**
- `sandbox_id` (string, required) — ID of the sandbox to stop.

**Output:** `{sandbox_id, status}`

### sandbox_traces

Retrieve the execution audit trail for a sandbox.

**Input:**
- `sandbox_id` (string, required) — ID of the target sandbox.
- `limit` (integer, optional) — Max events to return (default: 50).

**Output:** `{sandbox_id, event_count, events[]}`

### sandbox_policy_check

Pre-check whether an action would be allowed or denied by policy, without executing it.

**Input:**
- `action_type` (string, required) — Action type to check.
- `resource` (string, required) — Resource path or target.

**Output:** `{effect, allowed, rule, reason}`

## Protocol

The MCP server communicates via JSON-RPC 2.0 over stdio (newline-delimited JSON on stdin/stdout). Diagnostic logs go to stderr.

Supported methods:
- `initialize` — Protocol handshake
- `tools/list` — List available tools
- `tools/call` — Invoke a tool
- `ping` — Health check

## Environment Variables

The server respects the same environment variables as the API server for sandbox defaults:

- `SANDBOX_DEFAULT_TIMEOUT_SEC` — Default action timeout
- `SANDBOX_DEFAULT_MAX_MEMORY_MB` — Memory limit per sandbox
- `SANDBOX_DEFAULT_MAX_DISK_MB` — Disk limit per sandbox
- `SANDBOX_DEFAULT_MAX_PROCS` — Process count limit per sandbox
