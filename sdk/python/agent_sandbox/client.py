"""Python SDK client for the AgentSandbox API."""

import requests
from typing import Optional, List

from .types import Sandbox, ActionResult, TraceEvent, Action, PolicyDecision


class AgentSandboxError(Exception):
    """Raised when the API returns an error response."""

    def __init__(self, status_code: int, message: str):
        self.status_code = status_code
        self.message = message
        super().__init__(f"HTTP {status_code}: {message}")


class AgentSandboxClient:
    """Client for interacting with the AgentSandbox API server."""

    def __init__(self, base_url: str = "http://localhost:8080", token: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
        if token:
            self.session.headers["Authorization"] = f"Bearer {token}"

    def _url(self, path: str) -> str:
        return f"{self.base_url}/api/v1{path}"

    def _handle_response(self, resp: requests.Response) -> dict:
        if resp.status_code >= 400:
            try:
                body = resp.json()
                msg = body.get("error", resp.text)
            except (ValueError, KeyError):
                msg = resp.text
            raise AgentSandboxError(resp.status_code, msg)
        if resp.status_code == 204 or not resp.content:
            return {}
        return resp.json()

    @staticmethod
    def _parse_sandbox(data: dict) -> Sandbox:
        return Sandbox(
            id=data.get("id", ""),
            name=data.get("name", ""),
            status=data.get("status", ""),
            root_dir=data.get("root_dir", ""),
            created_at=data.get("created_at", ""),
        )

    @staticmethod
    def _parse_action(data: dict) -> Action:
        return Action(
            id=data.get("id", ""),
            type=data.get("type", ""),
            resource=data.get("resource", ""),
            params=data.get("params") or {},
            metadata=data.get("metadata") or {},
            timestamp=data.get("timestamp", ""),
        )

    @staticmethod
    def _parse_action_result(data: dict) -> ActionResult:
        return ActionResult(
            action_id=data.get("action_id", ""),
            success=data.get("success", False),
            output=data.get("output", ""),
            error=data.get("error", ""),
            exit_code=data.get("exit_code", 0),
            duration=data.get("duration", 0),
            bytes_read=data.get("bytes_read", 0),
            bytes_written=data.get("bytes_written", 0),
        )

    @staticmethod
    def _parse_trace_event(data: dict) -> TraceEvent:
        action = None
        if data.get("action"):
            action = AgentSandboxClient._parse_action(data["action"])

        result = None
        if data.get("result"):
            result = AgentSandboxClient._parse_action_result(data["result"])

        policy_decision = None
        if data.get("policy_decision"):
            pd = data["policy_decision"]
            policy_decision = PolicyDecision(
                effect=pd.get("effect", ""),
                allowed=pd.get("allowed", False),
                rule=pd.get("rule", ""),
                reason=pd.get("reason", ""),
            )

        return TraceEvent(
            id=data.get("id", ""),
            sandbox_id=data.get("sandbox_id", ""),
            parent_id=data.get("parent_id", ""),
            type=data.get("type", ""),
            action=action,
            action_id=data.get("action_id", ""),
            result=result,
            policy_decision=policy_decision,
            timestamp=data.get("timestamp", ""),
            duration=data.get("duration", 0),
            data=data.get("data") or {},
            attributes=data.get("attributes") or {},
        )

    def health(self) -> dict:
        resp = self.session.get(f"{self.base_url}/health")
        return self._handle_response(resp)

    def create_sandbox(self, name: str, policy_file: str = "", root_dir: str = "") -> Sandbox:
        payload = {"name": name}
        if policy_file:
            payload["policy_file"] = policy_file
        if root_dir:
            payload["root_dir"] = root_dir
        resp = self.session.post(self._url("/sandboxes"), json=payload)
        data = self._handle_response(resp)
        return self._parse_sandbox(data)

    def list_sandboxes(self) -> List[Sandbox]:
        resp = self.session.get(self._url("/sandboxes"))
        data = self._handle_response(resp)
        return [self._parse_sandbox(s) for s in data.get("sandboxes", [])]

    def get_sandbox(self, sandbox_id: str) -> Sandbox:
        resp = self.session.get(self._url(f"/sandboxes/{sandbox_id}"))
        data = self._handle_response(resp)
        return self._parse_sandbox(data)

    def start_sandbox(self, sandbox_id: str) -> Sandbox:
        resp = self.session.post(self._url(f"/sandboxes/{sandbox_id}/start"))
        data = self._handle_response(resp)
        return self._parse_sandbox(data)

    def stop_sandbox(self, sandbox_id: str) -> Sandbox:
        resp = self.session.post(self._url(f"/sandboxes/{sandbox_id}/stop"))
        data = self._handle_response(resp)
        return self._parse_sandbox(data)

    def destroy_sandbox(self, sandbox_id: str) -> None:
        resp = self.session.delete(self._url(f"/sandboxes/{sandbox_id}"))
        self._handle_response(resp)

    def execute(self, sandbox_id: str, action_type: str, params: dict = None) -> ActionResult:
        payload = {"type": action_type}
        if params:
            payload["params"] = params
        resp = self.session.post(self._url(f"/sandboxes/{sandbox_id}/exec"), json=payload)
        data = self._handle_response(resp)
        return self._parse_action_result(data)

    def get_traces(self, sandbox_id: str) -> List[TraceEvent]:
        resp = self.session.get(self._url(f"/sandboxes/{sandbox_id}/traces"))
        data = self._handle_response(resp)
        return [self._parse_trace_event(e) for e in data.get("events", [])]

    def get_dashboard_stats(self) -> dict:
        resp = self.session.get(self._url("/dashboard/stats"))
        return self._handle_response(resp)

    def get_audit_log(self, sandbox_id: str = None, limit: int = 100) -> List[dict]:
        params = {"limit": limit}
        if sandbox_id:
            params["sandbox_id"] = sandbox_id
        resp = self.session.get(self._url("/audit"), params=params)
        data = self._handle_response(resp)
        return data.get("entries", [])
