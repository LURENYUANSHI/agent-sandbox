You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

## Your Task: Create Python SDK client library

### 1. sdk/python/agent_sandbox/__init__.py
```python
from .client import AgentSandboxClient
from .types import Sandbox, Action, ActionResult, TraceEvent, Policy
__version__ = "0.3.0"
```

### 2. sdk/python/agent_sandbox/client.py
```python
import requests
from typing import Optional, List
from .types import *

class AgentSandboxClient:
    def __init__(self, base_url: str = "http://localhost:8080", token: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        if token:
            self.session.headers["Authorization"] = f"Bearer {token}"

    def health(self) -> dict: ...
    def create_sandbox(self, name: str, policy_file: str = "", root_dir: str = "") -> Sandbox: ...
    def list_sandboxes(self) -> List[Sandbox]: ...
    def get_sandbox(self, sandbox_id: str) -> Sandbox: ...
    def start_sandbox(self, sandbox_id: str) -> Sandbox: ...
    def stop_sandbox(self, sandbox_id: str) -> Sandbox: ...
    def destroy_sandbox(self, sandbox_id: str) -> None: ...
    def execute(self, sandbox_id: str, action_type: str, params: dict = None) -> ActionResult: ...
    def get_traces(self, sandbox_id: str) -> List[TraceEvent]: ...
    def get_dashboard_stats(self) -> dict: ...
    def get_audit_log(self, sandbox_id: str = None, limit: int = 100) -> List[dict]: ...
```

### 3. sdk/python/agent_sandbox/types.py
Pydantic or dataclass models matching the Go types.

### 4. sdk/python/setup.py / pyproject.toml
```toml
[project]
name = "agent-sandbox"
version = "0.3.0"
description = "Python SDK for AgentSandbox - AI Agent Security Sandbox"
requires-python = ">=3.9"
dependencies = ["requests>=2.28.0"]
```

### 5. sdk/python/README.md
Quick start:
```python
from agent_sandbox import AgentSandboxClient

client = AgentSandboxClient("http://localhost:8080", token="your-jwt-token")
sandbox = client.create_sandbox("my-agent")
client.start_sandbox(sandbox.id)
result = client.execute(sandbox.id, "file:read", {"path": "/tmp/test.txt"})
print(result)
```

### 6. sdk/python/tests/test_client.py
Unit tests using unittest.mock to mock HTTP responses:
- Test create/list/get/start/stop/destroy sandbox
- Test execute action
- Test auth header is sent
- Test error handling (4xx, 5xx)

### Verification:
1. `cd sdk/python && python -m pytest tests/ -v` (or just verify syntax)
2. `go build ./...` (Go code unchanged)
3. Commit: `feat: add Python SDK client library for AgentSandbox API`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
