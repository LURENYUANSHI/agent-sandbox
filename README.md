# AgentSandbox

[![CI](https://github.com/LURENYUANSHI/agent-sandbox/actions/workflows/ci.yml/badge.svg)](https://github.com/LURENYUANSHI/agent-sandbox/actions/workflows/ci.yml)

> Open-source runtime sandbox for AI agents with policy enforcement, trace recording, and replay debugging.

## Why AgentSandbox?

AI agents are increasingly autonomous — browsing the web, executing code, managing files, and interacting with APIs. But with great power comes great risk: agents can delete production databases, exfiltrate sensitive data, or execute dangerous commands without oversight.

AgentSandbox provides a secure execution layer that:

- **Enforces fine-grained permission policies** before every action an agent takes
- **Records complete execution traces** for audit, compliance, and debugging
- **Supports trace replay** for understanding and reproducing agent behavior
- **Provides a visual dashboard** for real-time monitoring and analysis
- **Integrates with OpenTelemetry** for existing observability stacks

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

## API Reference

| Method   | Endpoint                            | Description                |
|----------|-------------------------------------|----------------------------|
| `GET`    | `/api/v1/health`                    | Health check               |
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
│   ├── api/              # REST API server and handlers
│   └── types/            # Shared types and interfaces
├── web/                  # React dashboard (Vite + TailwindCSS)
├── configs/              # Policy configuration files
├── docker/               # Dockerfile and docker-compose
├── test/                 # Integration tests and fixtures
├── docs/                 # Architecture and development docs
└── scripts/              # Development setup scripts
```

## Tech Stack

| Component         | Technology                              |
|-------------------|-----------------------------------------|
| Sandbox Runtime   | Go (OS-level isolation, syscall filter) |
| Policy Engine     | YAML-based declarative rules            |
| Trace System      | OpenTelemetry-compatible, SQLite store  |
| CLI               | Go + Cobra                              |
| API Server        | Go + Gin                                |
| Web Dashboard     | React + TypeScript + Vite + TailwindCSS |
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
