# AgentSandbox

[![CI](https://github.com/LURENYUANSHI/agent-sandbox/actions/workflows/ci.yml/badge.svg)](https://github.com/LURENYUANSHI/agent-sandbox/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-84.3%25-brightgreen)](docs/v0.3.0-evaluation.md)
[![Go Report](https://goreportcard.com/badge/github.com/LURENYUANSHI/agent-sandbox)](https://goreportcard.com/report/github.com/LURENYUANSHI/agent-sandbox)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> Open-source runtime sandbox for AI agents with policy enforcement, trace recording, and replay debugging.

## Why AgentSandbox?

AI agents are increasingly autonomous — browsing the web, executing code, managing files, and interacting with APIs. But with great power comes great risk: agents can delete production databases, exfiltrate sensitive data, or execute dangerous commands without oversight.

AgentSandbox provides a secure execution layer that:

- **Enforces fine-grained permission policies** before every action an agent takes
- **Records complete execution traces** for audit, compliance, and debugging
- **Supports trace replay** for understanding and reproducing agent behavior
- **Provides a visual dashboard** for real-time monitoring and analysis
- **Integrates with OpenTelemetry** for existing observability stacks
- **Authenticates API access** with JWT token-based security
- **Rate limits requests** to prevent abuse and ensure fair usage
- **Validates all inputs** with comprehensive request validation
- **Logs audit trails** with persistent, queryable audit records
- **Enforces resource limits** on disk usage and process counts per sandbox

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Web Dashboard                          │
│              (React + TypeScript + TailwindCSS)             │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP / WebSocket
┌──────────────────────────▼──────────────────────────────────┐
│                       REST API (Gin)                        │
│         /api/sandboxes  /api/traces  /api/replay            │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                    Sandbox Runtime                           │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │   Policy     │  │   Executor   │  │  Trace Recorder   │  │
│  │   Engine     │──│  (FS/Net/    │──│  (OpenTelemetry)  │  │
│  │  (YAML rules)│  │   Process)   │  │                   │  │
│  └─────────────┘  └──────────────┘  └─────────┬─────────┘  │
│                                                │            │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────▼─────────┐  │
│  │  OS-level   │  │   Resource   │  │   Trace Store     │  │
│  │  Isolation  │  │   Limits     │  │   (SQLite)        │  │
│  └─────────────┘  └──────────────┘  └───────────────────┘  │
└─────────────────────────────────────────────────────────────┘

Flow: Agent Action → Policy Check → Execute → Record Trace
```

## Quick Demo

Try AgentSandbox instantly with the interactive demo:

```bash
# Run the interactive demo (shows policy enforcement, tracing, and replay)
go run examples/demo/main.go

# Or use the one-command demo script
bash scripts/demo.sh
```

### More Examples

```bash
# Simulated coding agent: reads, modifies, and formats Go source files
go run examples/coding-agent/main.go

# Simulated web scraper: HTTP requests with filesystem restrictions
go run examples/web-scraper/main.go
```

## Quick Start

### Using Docker Compose (recommended)

```bash
git clone https://github.com/LURENYUANSHI/agent-sandbox.git
cd agent-sandbox

# Start the full stack
docker-compose -f docker/docker-compose.yaml up --build

# Access the dashboard at http://localhost:8080
```

### Building from Source

```bash
# Prerequisites: Go 1.25+, Node.js 22+

# Install dependencies
bash scripts/setup.sh

# Build CLI and server
make build

# Run the API server
make run-server

# Run the web dashboard (separate terminal)
make run-web
```

## MCP Server (AI Agent Integration)

AgentSandbox includes an MCP (Model Context Protocol) server that lets AI agents like Claude use the sandbox as a tool. The MCP server exposes sandbox operations over stdio using JSON-RPC 2.0.

### Setup for Claude Desktop

```bash
# Build the MCP server
make build-mcp

# Add to claude_desktop_config.json:
```

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

### MCP Tools

| Tool | Description |
|------|-------------|
| `sandbox_create` | Create an isolated sandbox with policy enforcement |
| `sandbox_exec` | Execute actions (file, network, process) with policy checks |
| `sandbox_stop` | Stop and clean up a sandbox |
| `sandbox_traces` | View execution audit trail |
| `sandbox_policy_check` | Pre-check if an action would be allowed |

See [mcp/README.md](mcp/README.md) for full documentation.

## Python SDK

A Python client library for integrating AgentSandbox into your applications:

```bash
# Install from source
pip install -e sdk/python/
```

```python
from agent_sandbox import SandboxClient

client = SandboxClient("http://localhost:8080")

# Create and use a sandbox
sandbox = client.create_sandbox(name="my-agent", policy_file="configs/default-policy.yaml")
result = client.exec_action(sandbox["id"], action_type="shell", command="echo hello")
traces = client.get_traces(sandbox["id"])
client.stop_sandbox(sandbox["id"])
```

See [sdk/python/README.md](sdk/python/README.md) for full documentation.

## CLI Usage

```bash
# Build the CLI
make build-cli

# The CLI binary is at bin/agent-sandbox
# Planned subcommands:
#   create    - Create a new sandbox instance
#   start     - Start a sandbox
#   exec      - Execute an action inside a sandbox
#   stop      - Stop a running sandbox
#   list      - List active sandboxes
#   trace     - View traces for a sandbox
#   replay    - Replay a recorded trace
```

## Security

AgentSandbox includes multiple layers of security for production deployments:

- **JWT Authentication** — Token-based API authentication. Generate tokens via `POST /api/v1/auth/token` and include them as `Authorization: Bearer <token>` headers. Configurable secret and expiry via environment variables (`AGENTSANDBOX_JWT_SECRET`, `AGENTSANDBOX_JWT_EXPIRY`).
- **Rate Limiting** — Per-IP rate limiting protects against abuse. Configurable rate and burst size via `AGENTSANDBOX_RATE_LIMIT` and `AGENTSANDBOX_RATE_BURST`.
- **Input Validation** — All API inputs are validated before processing with structured error responses detailing validation failures.
- **Audit Logging** — Every policy decision is recorded to a persistent SQLite audit log, queryable via `GET /api/v1/audit`. Supports filtering by sandbox ID, action type, effect, and time range.
- **Resource Limits** — Per-sandbox enforcement of disk usage and process count limits prevents resource exhaustion.
- **RBAC** — Role-based access control with 4 roles (admin, operator, viewer, agent) and 8 granular permissions protecting all API endpoints.

## Prometheus Metrics

AgentSandbox exposes Prometheus metrics at `GET /metrics` for monitoring:

| Metric | Type | Description |
|--------|------|-------------|
| `sandbox_created_total` | Counter | Total sandboxes created |
| `sandbox_actions_total` | Counter | Actions executed (by type and status) |
| `sandbox_policy_evaluations_total` | Counter | Policy evaluations (by effect) |
| `sandbox_api_request_duration_seconds` | Histogram | API request latency (by method and path) |

Integrate with Grafana for real-time dashboards and alerting.

## API Reference

| Method   | Endpoint                            | Description                |
|----------|-------------------------------------|----------------------------|
| `GET`    | `/api/v1/health`                    | Health check               |
| `POST`   | `/api/v1/auth/token`                | Generate JWT auth token    |
| `POST`   | `/api/v1/sandboxes`                 | Create a new sandbox       |
| `GET`    | `/api/v1/sandboxes`                 | List all sandboxes         |
| `GET`    | `/api/v1/sandboxes/:id`             | Get sandbox details        |
| `POST`   | `/api/v1/sandboxes/:id/start`       | Start a sandbox            |
| `POST`   | `/api/v1/sandboxes/:id/exec`        | Execute action in sandbox  |
| `POST`   | `/api/v1/sandboxes/:id/stop`        | Stop a sandbox             |
| `DELETE` | `/api/v1/sandboxes/:id`             | Destroy a sandbox          |
| `GET`    | `/api/v1/sandboxes/:id/traces`      | Get traces for sandbox     |
| `POST`   | `/api/v1/sandboxes/:id/replay`      | Start trace replay         |
| `GET`    | `/api/v1/sandboxes/:id/replay/next` | Get next replay event      |
| `POST`   | `/api/v1/policies/validate`         | Validate a policy          |
| `GET`    | `/api/v1/sandboxes/:id/ws`          | WebSocket trace streaming  |
| `GET`    | `/api/v1/dashboard/stats`           | Dashboard statistics       |
| `GET`    | `/api/v1/dashboard/activity`        | Recent activity feed       |
| `GET`    | `/api/v1/audit`                     | Query audit log            |
| `GET`    | `/metrics`                          | Prometheus metrics         |
| `GET`    | `/swagger/*`                        | Swagger API documentation  |

## Policy Configuration

Policies are defined in YAML and evaluated before every agent action. Place policy files in `configs/`.

```yaml
# configs/default-policy.yaml
name: default
description: Default restrictive security policy

rules:
  - name: allow-read-workspace
    effect: allow
    action: file.read
    resource: /workspace/**

  - name: deny-system-files
    effect: deny
    action: file.*
    resource: /etc/**

  - name: allow-localhost-only
    effect: allow
    action: network.connect
    resource: "127.0.0.1:*"

  - name: deny-external-network
    effect: deny
    action: network.*
    resource: "*"
```

### Example Policies

- **`configs/examples/strict.yaml`** — Denies most operations, suitable for untrusted agents
- **`configs/examples/permissive.yaml`** — Allows most operations, for trusted environments
- **`configs/examples/coding-agent.yaml`** — Filesystem and process access with restricted networking

## Project Structure

```
ai-sandbox/
├── cmd/
│   ├── sandbox/          # CLI entry point
│   └── server/           # API server entry point
├── pkg/
│   ├── sandbox/          # Core sandbox lifecycle
│   ├── executor/         # Action executor (filesystem, network, process)
│   ├── policy/           # Policy engine and YAML parser
│   ├── trace/            # Trace recording, storage, replay, OTel export
│   ├── api/              # REST API, auth, RBAC, rate limiting, validation
│   ├── metrics/          # Prometheus metrics collectors
│   └── types/            # Shared types and interfaces
├── mcp/                  # MCP server for AI agent integration
├── sdk/python/           # Python SDK client library
├── examples/             # Interactive demo examples
├── web/                  # React dashboard (Vite + TailwindCSS)
├── configs/              # Policy configuration files
├── docker/               # Dockerfile and docker-compose
├── docs/                 # Architecture docs, Swagger, evaluations
├── test/                 # Integration tests and fixtures
└── scripts/              # Development setup scripts
```

## Tech Stack

| Component         | Technology                              |
|-------------------|-----------------------------------------|
| Sandbox Runtime   | Go (OS-level isolation, syscall filter) |
| Policy Engine     | YAML-based declarative rules            |
| Trace System      | OpenTelemetry-compatible, SQLite store  |
| CLI               | Go + Cobra                              |
| API Server        | Go + Gin + RBAC + JWT                   |
| Web Dashboard     | React + TypeScript + Vite + TailwindCSS |
| MCP Server        | Go, JSON-RPC 2.0 over stdio            |
| Python SDK        | Python 3.8+, httpx, pydantic            |
| Monitoring        | Prometheus metrics + Swagger docs       |
| Build             | Makefile + Docker multi-stage           |

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean
```

See [docs/development-plan.md](docs/development-plan.md) for the full development roadmap.

## Contributing

1. Fork the repository
2. Create a feature branch from `develop`: `git checkout -b feature/my-feature develop`
3. Write tests for your changes
4. Ensure all tests pass: `make test`
5. Ensure code is formatted: `gofmt -w .`
6. Commit using [Conventional Commits](https://www.conventionalcommits.org/): `feat(scope): description`
7. Open a Pull Request targeting `develop`

### Branch Strategy

- `main` — Production releases only
- `develop` — Integration branch, all PRs target here
- `feature/*` — Feature branches created from `develop`

## License

Apache 2.0 — see [LICENSE](LICENSE) for details.
