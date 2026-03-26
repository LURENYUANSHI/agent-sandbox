"""Unit tests for AgentSandboxClient."""

import json
import unittest
from unittest.mock import patch, MagicMock

from agent_sandbox.client import AgentSandboxClient, AgentSandboxError
from agent_sandbox.types import Sandbox, ActionResult, TraceEvent


def mock_response(status_code=200, json_data=None):
    resp = MagicMock()
    resp.status_code = status_code
    resp.content = json.dumps(json_data).encode() if json_data else b""
    resp.text = json.dumps(json_data) if json_data else ""
    resp.json.return_value = json_data or {}
    return resp


class TestClientInit(unittest.TestCase):
    def test_default_url(self):
        client = AgentSandboxClient()
        self.assertEqual(client.base_url, "http://localhost:8080")

    def test_custom_url_strips_trailing_slash(self):
        client = AgentSandboxClient("http://example.com:9090/")
        self.assertEqual(client.base_url, "http://example.com:9090")

    def test_auth_header_set(self):
        client = AgentSandboxClient(token="test-token")
        self.assertEqual(client.session.headers["Authorization"], "Bearer test-token")

    def test_no_auth_header_without_token(self):
        client = AgentSandboxClient()
        self.assertNotIn("Authorization", client.session.headers)


class TestHealth(unittest.TestCase):
    @patch.object(AgentSandboxClient, "__init__", lambda self, **kw: setattr(self, "base_url", "http://localhost:8080") or setattr(self, "session", MagicMock()))
    def test_health(self):
        client = AgentSandboxClient()
        client.session.get.return_value = mock_response(200, {"status": "ok"})
        result = client.health()
        self.assertEqual(result["status"], "ok")
        client.session.get.assert_called_once_with("http://localhost:8080/health")


class TestSandboxOperations(unittest.TestCase):
    def setUp(self):
        self.client = AgentSandboxClient()
        self.client.session = MagicMock()

    def test_create_sandbox(self):
        self.client.session.post.return_value = mock_response(201, {
            "id": "sb-123", "name": "test", "status": "created", "root_dir": "/tmp/sb", "created_at": "2026-01-01T00:00:00Z"
        })
        sb = self.client.create_sandbox("test", root_dir="/tmp/sb")
        self.assertIsInstance(sb, Sandbox)
        self.assertEqual(sb.id, "sb-123")
        self.assertEqual(sb.name, "test")
        self.assertEqual(sb.status, "created")
        call_args = self.client.session.post.call_args
        self.assertEqual(call_args[1]["json"]["name"], "test")
        self.assertEqual(call_args[1]["json"]["root_dir"], "/tmp/sb")

    def test_create_sandbox_minimal(self):
        self.client.session.post.return_value = mock_response(201, {
            "id": "sb-456", "name": "minimal", "status": "created"
        })
        sb = self.client.create_sandbox("minimal")
        self.assertEqual(sb.id, "sb-456")
        payload = self.client.session.post.call_args[1]["json"]
        self.assertNotIn("policy_file", payload)
        self.assertNotIn("root_dir", payload)

    def test_list_sandboxes(self):
        self.client.session.get.return_value = mock_response(200, {
            "sandboxes": [
                {"id": "sb-1", "name": "a", "status": "running"},
                {"id": "sb-2", "name": "b", "status": "stopped"},
            ]
        })
        result = self.client.list_sandboxes()
        self.assertEqual(len(result), 2)
        self.assertIsInstance(result[0], Sandbox)
        self.assertEqual(result[0].id, "sb-1")
        self.assertEqual(result[1].status, "stopped")

    def test_get_sandbox(self):
        self.client.session.get.return_value = mock_response(200, {
            "id": "sb-123", "name": "test", "status": "running"
        })
        sb = self.client.get_sandbox("sb-123")
        self.assertEqual(sb.id, "sb-123")

    def test_start_sandbox(self):
        self.client.session.post.return_value = mock_response(200, {
            "id": "sb-123", "status": "running"
        })
        sb = self.client.start_sandbox("sb-123")
        self.assertEqual(sb.status, "running")

    def test_stop_sandbox(self):
        self.client.session.post.return_value = mock_response(200, {
            "id": "sb-123", "status": "stopped"
        })
        sb = self.client.stop_sandbox("sb-123")
        self.assertEqual(sb.status, "stopped")

    def test_destroy_sandbox(self):
        self.client.session.delete.return_value = mock_response(200, {
            "id": "sb-123", "destroyed": True
        })
        self.client.destroy_sandbox("sb-123")
        self.client.session.delete.assert_called_once()


class TestExecute(unittest.TestCase):
    def setUp(self):
        self.client = AgentSandboxClient()
        self.client.session = MagicMock()

    def test_execute_action(self):
        self.client.session.post.return_value = mock_response(200, {
            "action_id": "act-1", "success": True, "output": "hello world", "exit_code": 0, "duration": 150
        })
        result = self.client.execute("sb-123", "file:read", {"path": "/tmp/test.txt"})
        self.assertIsInstance(result, ActionResult)
        self.assertTrue(result.success)
        self.assertEqual(result.output, "hello world")
        payload = self.client.session.post.call_args[1]["json"]
        self.assertEqual(payload["type"], "file:read")
        self.assertEqual(payload["params"]["path"], "/tmp/test.txt")

    def test_execute_without_params(self):
        self.client.session.post.return_value = mock_response(200, {
            "action_id": "act-2", "success": True, "output": ""
        })
        self.client.execute("sb-123", "shell:exec")
        payload = self.client.session.post.call_args[1]["json"]
        self.assertNotIn("params", payload)

    def test_execute_denied(self):
        self.client.session.post.return_value = mock_response(403, {
            "error": "action denied by policy"
        })
        with self.assertRaises(AgentSandboxError) as ctx:
            self.client.execute("sb-123", "net:connect", {"host": "evil.com"})
        self.assertEqual(ctx.exception.status_code, 403)


class TestTraces(unittest.TestCase):
    def setUp(self):
        self.client = AgentSandboxClient()
        self.client.session = MagicMock()

    def test_get_traces(self):
        self.client.session.get.return_value = mock_response(200, {
            "events": [
                {
                    "id": "evt-1", "sandbox_id": "sb-123", "type": "action.executed",
                    "action": {"id": "act-1", "type": "file:read", "params": {"path": "/tmp"}},
                    "result": {"action_id": "act-1", "success": True, "output": "data"},
                },
                {
                    "id": "evt-2", "sandbox_id": "sb-123", "type": "policy.evaluated",
                    "policy_decision": {"allowed": True, "effect": "allow", "rule": "rule-1"},
                },
            ]
        })
        events = self.client.get_traces("sb-123")
        self.assertEqual(len(events), 2)
        self.assertIsInstance(events[0], TraceEvent)
        self.assertEqual(events[0].action.type, "file:read")
        self.assertTrue(events[0].result.success)
        self.assertTrue(events[1].policy_decision.allowed)

    def test_get_traces_empty(self):
        self.client.session.get.return_value = mock_response(200, {"events": []})
        events = self.client.get_traces("sb-123")
        self.assertEqual(events, [])


class TestDashboardAndAudit(unittest.TestCase):
    def setUp(self):
        self.client = AgentSandboxClient()
        self.client.session = MagicMock()

    def test_get_dashboard_stats(self):
        self.client.session.get.return_value = mock_response(200, {
            "total_sandboxes": 5, "active": 2
        })
        stats = self.client.get_dashboard_stats()
        self.assertEqual(stats["total_sandboxes"], 5)

    def test_get_audit_log(self):
        self.client.session.get.return_value = mock_response(200, {
            "entries": [{"id": "aud-1", "action_type": "file:read"}]
        })
        entries = self.client.get_audit_log(sandbox_id="sb-123", limit=50)
        self.assertEqual(len(entries), 1)
        call_args = self.client.session.get.call_args
        self.assertEqual(call_args[1]["params"]["sandbox_id"], "sb-123")
        self.assertEqual(call_args[1]["params"]["limit"], 50)

    def test_get_audit_log_defaults(self):
        self.client.session.get.return_value = mock_response(200, {"entries": []})
        self.client.get_audit_log()
        call_args = self.client.session.get.call_args
        self.assertNotIn("sandbox_id", call_args[1]["params"])
        self.assertEqual(call_args[1]["params"]["limit"], 100)


class TestErrorHandling(unittest.TestCase):
    def setUp(self):
        self.client = AgentSandboxClient()
        self.client.session = MagicMock()

    def test_404_raises(self):
        self.client.session.get.return_value = mock_response(404, {"error": "not found"})
        with self.assertRaises(AgentSandboxError) as ctx:
            self.client.get_sandbox("nonexistent")
        self.assertEqual(ctx.exception.status_code, 404)
        self.assertIn("not found", ctx.exception.message)

    def test_500_raises(self):
        self.client.session.get.return_value = mock_response(500, {"error": "internal error"})
        with self.assertRaises(AgentSandboxError) as ctx:
            self.client.list_sandboxes()
        self.assertEqual(ctx.exception.status_code, 500)

    def test_401_unauthorized(self):
        self.client.session.post.return_value = mock_response(401, {"error": "unauthorized"})
        with self.assertRaises(AgentSandboxError) as ctx:
            self.client.create_sandbox("test")
        self.assertEqual(ctx.exception.status_code, 401)

    def test_non_json_error_body(self):
        resp = MagicMock()
        resp.status_code = 502
        resp.content = b"Bad Gateway"
        resp.text = "Bad Gateway"
        resp.json.side_effect = ValueError("No JSON")
        self.client.session.get.return_value = resp
        with self.assertRaises(AgentSandboxError) as ctx:
            self.client.list_sandboxes()
        self.assertEqual(ctx.exception.status_code, 502)
        self.assertIn("Bad Gateway", ctx.exception.message)


if __name__ == "__main__":
    unittest.main()
