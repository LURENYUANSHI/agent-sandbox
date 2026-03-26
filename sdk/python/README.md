# AgentSandbox Python SDK

Python client library for the [AgentSandbox](https://github.com/LURENYUANSHI/agent-sandbox) API.

## Installation

```bash
pip install agent-sandbox
```

Or install from source:

```bash
cd sdk/python
pip install -e .
```

## Quick Start

```python
from agent_sandbox import AgentSandboxClient

client = AgentSandboxClient("http://localhost:8080", token="your-jwt-token")

# Create and start a sandbox
sandbox = client.create_sandbox("my-agent")
client.start_sandbox(sandbox.id)

# Execute an action
result = client.execute(sandbox.id, "file:read", {"path": "/tmp/test.txt"})
print(result.output)

# View traces
traces = client.get_traces(sandbox.id)
for event in traces:
    print(event.type, event.timestamp)

# Clean up
client.stop_sandbox(sandbox.id)
client.destroy_sandbox(sandbox.id)
```

## API Reference

### `AgentSandboxClient(base_url, token=None)`

| Method | Description |
|---|---|
| `health()` | Check API server health |
| `create_sandbox(name, policy_file="", root_dir="")` | Create a new sandbox |
| `list_sandboxes()` | List all sandboxes |
| `get_sandbox(sandbox_id)` | Get sandbox details |
| `start_sandbox(sandbox_id)` | Start a sandbox |
| `stop_sandbox(sandbox_id)` | Stop a sandbox |
| `destroy_sandbox(sandbox_id)` | Destroy a sandbox |
| `execute(sandbox_id, action_type, params=None)` | Execute an action in a sandbox |
| `get_traces(sandbox_id)` | Get trace events for a sandbox |
| `get_dashboard_stats()` | Get dashboard statistics |
| `get_audit_log(sandbox_id=None, limit=100)` | Query audit log |

### Error Handling

```python
from agent_sandbox.client import AgentSandboxError

try:
    client.get_sandbox("nonexistent")
except AgentSandboxError as e:
    print(e.status_code, e.message)
```
