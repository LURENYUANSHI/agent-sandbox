# AI Agent Security Sandbox (AgentSandbox)

## Project Overview
An open-source runtime sandbox for AI agents that enforces safe execution boundaries, records traces, supports replay/debugging, and provides fine-grained permission control.

## Tech Stack
- **Sandbox Runtime**: Go (container isolation, process control, syscall filtering)
- **Trace System**: OpenTelemetry-compatible, stored in SQLite
- **CLI**: Go (cobra)
- **Web Dashboard**: React + TypeScript + Vite + TailwindCSS
- **API Server**: Go (gin)
- **Tests**: Go testing + pytest for integration tests
- **Build**: Makefile + Docker

## Project Structure
```
ai-sandbox/
├── CLAUDE.md                 # This file
├── README.md                 # Project README
├── LICENSE                   # Apache 2.0
├── Makefile                  # Build commands
├── go.mod / go.sum           # Go modules
├── cmd/
│   ├── sandbox/              # Main CLI entry point
│   │   └── main.go
│   └── server/               # API server entry point
│       └── main.go
├── pkg/
│   ├── sandbox/              # Core sandbox engine
│   │   ├── sandbox.go        # Sandbox lifecycle (create, start, stop, destroy)
│   │   ├── config.go         # Sandbox configuration
│   │   ├── policy.go         # Permission policies (allow/deny rules)
│   │   └── sandbox_test.go
│   ├── executor/             # Command/action executor inside sandbox
│   │   ├── executor.go       # Execute actions within sandbox boundaries
│   │   ├── filesystem.go     # Filesystem access control
│   │   ├── network.go        # Network access control
│   │   ├── process.go        # Process execution control
│   │   └── executor_test.go
│   ├── trace/                # Trace recording and replay
│   │   ├── recorder.go       # Record all actions and decisions
│   │   ├── replayer.go       # Replay recorded traces
│   │   ├── store.go          # SQLite trace storage
│   │   ├── otel.go           # OpenTelemetry export
│   │   └── trace_test.go
│   ├── policy/               # Policy engine
│   │   ├── engine.go         # Policy evaluation engine
│   │   ├── rules.go          # Built-in safety rules
│   │   ├── parser.go         # Policy file parser (YAML)
│   │   └── policy_test.go
│   ├── api/                  # REST API
│   │   ├── server.go         # HTTP server setup
│   │   ├── handlers.go       # API route handlers
│   │   ├── middleware.go      # Auth, logging middleware
│   │   └── api_test.go
│   └── types/                # Shared types
│       ├── action.go         # Action types (file, network, process, shell)
│       ├── event.go          # Trace event types
│       └── policy.go         # Policy types
├── web/                      # React dashboard
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/
│   │   │   ├── Dashboard.tsx       # Main dashboard
│   │   │   ├── TraceViewer.tsx     # Trace visualization
│   │   │   ├── PolicyEditor.tsx    # Visual policy editor
│   │   │   ├── SandboxList.tsx     # Active sandboxes
│   │   │   └── ReplayControls.tsx  # Trace replay UI
│   │   └── lib/
│   │       └── api.ts              # API client
│   └── index.html
├── configs/
│   ├── default-policy.yaml   # Default security policy
│   └── examples/             # Example policies
│       ├── strict.yaml
│       ├── permissive.yaml
│       └── coding-agent.yaml
├── docs/
│   └── architecture.md       # Architecture documentation
├── scripts/
│   └── setup.sh              # Dev environment setup
├── test/
│   ├── integration/          # Integration tests
│   │   ├── sandbox_test.go
│   │   └── api_test.go
│   └── fixtures/             # Test fixtures
│       ├── sample-policy.yaml
│       └── sample-trace.json
└── docker/
    ├── Dockerfile            # Production image
    └── docker-compose.yaml   # Dev environment
```

## Architecture Key Decisions
1. **Sandbox Isolation**: Use OS-level process isolation (namespaces on Linux, job objects on Windows) + filesystem chroot
2. **Policy Engine**: YAML-based declarative policies, evaluated before each action
3. **Trace Format**: OpenTelemetry-compatible spans with custom attributes for agent actions
4. **Storage**: SQLite for traces (simple, embedded, no external deps for MVP)
5. **API**: REST with WebSocket for real-time trace streaming

## Coding Conventions
- Go: Follow standard Go conventions, use `gofmt`
- React: Functional components, hooks, TypeScript strict mode
- Error handling: Always wrap errors with context using `fmt.Errorf("...: %w", err)`
- Tests: Table-driven tests in Go, minimum 80% coverage for core packages
- Commits: Conventional commits format

## Development Phases
See `docs/development-plan.md` for the 8-hour overnight development plan.
