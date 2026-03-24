# AgentSandbox

An open-source runtime sandbox for AI agents that enforces safe execution boundaries, records traces, supports replay/debugging, and provides fine-grained permission control.

## Features

- **Sandbox Isolation** - OS-level process isolation with filesystem chroot
- **Policy Engine** - YAML-based declarative security policies evaluated before each action
- **Trace Recording** - OpenTelemetry-compatible trace recording for all agent actions
- **Trace Replay** - Step-through replay and debugging of recorded traces
- **Web Dashboard** - React-based UI for managing sandboxes, viewing traces, and editing policies
- **REST API** - Full API for programmatic sandbox management

## Tech Stack

- **Runtime**: Go (container isolation, process control, syscall filtering)
- **Traces**: OpenTelemetry-compatible, stored in SQLite
- **CLI**: Go + Cobra
- **Web**: React + TypeScript + Vite + TailwindCSS
- **API**: Go + Gin
- **Build**: Makefile + Docker

## Quick Start

```bash
# Build
make build

# Run the API server
make run-server

# Run the web dashboard
make run-web

# Run tests
make test
```

## Project Structure

See [CLAUDE.md](CLAUDE.md) for detailed project structure and conventions.

## License

Apache 2.0 - see [LICENSE](LICENSE) for details.
