# AgentSandbox - 8-Hour Overnight Development Plan

## Time Budget: 00:00 - 08:00 (8 hours)

---

## Phase 1: Project Initialization (00:00 - 00:30, 30 min)
**Goal**: Set up project skeleton, go modules, basic structure

### Tasks:
- Initialize git repo
- Create go.mod with module name `github.com/LURENYUANSHI/agent-sandbox`
- Create all directory structure from CLAUDE.md
- Set up Makefile with build/test/lint targets
- Create LICENSE (Apache 2.0)
- Write initial README.md
- Initialize web/ with Vite + React + TypeScript
- Create docker/Dockerfile and docker-compose.yaml skeleton
- **Checkpoint**: `git commit` with project skeleton

---

## Phase 2: Core Types & Interfaces (00:30 - 01:15, 45 min)
**Goal**: Define all shared types, interfaces, and contracts

### Tasks:
- `pkg/types/action.go` - Action types (FileAction, NetworkAction, ProcessAction, ShellAction)
- `pkg/types/event.go` - TraceEvent, Span, TraceContext
- `pkg/types/policy.go` - Policy, Rule, Permission, Effect (Allow/Deny)
- Define interfaces: Sandbox, Executor, PolicyEngine, TraceRecorder, TraceStore
- Write comprehensive godoc comments
- **Checkpoint**: `git commit` with core types

---

## Phase 3: Policy Engine (01:15 - 02:15, 60 min)
**Goal**: Build the policy evaluation engine

### Tasks:
- `pkg/policy/engine.go` - PolicyEngine that evaluates actions against rules
- `pkg/policy/rules.go` - Built-in rules (no-delete-root, no-outbound-network, no-kill-process, etc.)
- `pkg/policy/parser.go` - YAML policy file parser
- `configs/default-policy.yaml` - Default restrictive policy
- `configs/examples/*.yaml` - Example policies for different use cases
- `pkg/policy/policy_test.go` - Comprehensive table-driven tests
- **Checkpoint**: `git commit` with policy engine, all tests passing

---

## Phase 4: Trace System (02:15 - 03:15, 60 min)
**Goal**: Build trace recording, storage, and replay

### Tasks:
- `pkg/trace/recorder.go` - Record actions with timestamps, inputs, outputs, decisions
- `pkg/trace/store.go` - SQLite storage for traces (create tables, CRUD operations)
- `pkg/trace/replayer.go` - Load and replay traces step-by-step
- `pkg/trace/otel.go` - Export traces to OpenTelemetry format
- `pkg/trace/trace_test.go` - Tests for all trace operations
- **Checkpoint**: `git commit` with trace system, all tests passing

---

## Phase 5: Sandbox Runtime & Executor (03:15 - 04:30, 75 min)
**Goal**: Build the core sandbox and executor

### Tasks:
- `pkg/sandbox/config.go` - Sandbox configuration (resource limits, allowed paths, network rules)
- `pkg/sandbox/sandbox.go` - Sandbox lifecycle: Create, Start, Execute, Stop, Destroy
- `pkg/executor/executor.go` - Execute actions within sandbox, check policy before each action
- `pkg/executor/filesystem.go` - Filesystem operations with path validation and chroot
- `pkg/executor/network.go` - Network operations with allowlist/denylist
- `pkg/executor/process.go` - Process execution with resource limits
- `pkg/sandbox/sandbox_test.go` - Sandbox lifecycle tests
- `pkg/executor/executor_test.go` - Executor tests with mock policies
- **Checkpoint**: `git commit` with sandbox runtime, all tests passing

---

## Phase 6: CLI & API Server (04:30 - 05:30, 60 min)
**Goal**: Build the CLI interface and REST API

### Tasks:
- `cmd/sandbox/main.go` - CLI with subcommands: create, start, exec, stop, list, trace, replay
- `pkg/api/server.go` - Gin HTTP server setup with CORS, logging
- `pkg/api/handlers.go` - REST endpoints:
  - POST /api/sandboxes - Create sandbox
  - GET /api/sandboxes - List sandboxes
  - POST /api/sandboxes/:id/exec - Execute action
  - GET /api/sandboxes/:id/traces - Get traces
  - POST /api/sandboxes/:id/replay - Start replay
  - DELETE /api/sandboxes/:id - Destroy sandbox
- `pkg/api/middleware.go` - Request logging, error handling
- `cmd/server/main.go` - API server entry point
- `pkg/api/api_test.go` - API handler tests
- **Checkpoint**: `git commit` with CLI and API, all tests passing

---

## Phase 7: Web Dashboard (05:30 - 06:45, 75 min)
**Goal**: Build the React dashboard for visualization

### Tasks:
- `web/src/lib/api.ts` - TypeScript API client
- `web/src/components/Dashboard.tsx` - Main dashboard with stats
- `web/src/components/SandboxList.tsx` - List/manage active sandboxes
- `web/src/components/TraceViewer.tsx` - Visual trace timeline (tree view of actions)
- `web/src/components/PolicyEditor.tsx` - Visual policy editor with YAML preview
- `web/src/components/ReplayControls.tsx` - Step-through replay with play/pause/step
- `web/src/App.tsx` - Router and layout
- Style with TailwindCSS, dark mode
- **Checkpoint**: `git commit` with web dashboard

---

## Phase 8: Integration Tests & Docker (06:45 - 07:30, 45 min)
**Goal**: End-to-end testing and containerization

### Tasks:
- `test/integration/sandbox_test.go` - Full lifecycle integration test
- `test/integration/api_test.go` - API integration test
- `test/fixtures/` - Test policy files and sample traces
- `docker/Dockerfile` - Multi-stage build (Go backend + React frontend)
- `docker/docker-compose.yaml` - Full stack compose file
- Run all tests, fix any failures
- **Checkpoint**: `git commit` with integration tests and Docker

---

## Phase 9: Documentation, Polish & Evaluation (07:30 - 08:00, 30 min)
**Goal**: Final polish and self-evaluation

### Tasks:
- Write comprehensive README.md with:
  - Project description and motivation
  - Architecture diagram (ASCII)
  - Quick start guide
  - Usage examples
  - API reference summary
  - Contributing guide
- Run full test suite, record coverage
- Build Docker image, verify it works
- Generate evaluation report:
  - Lines of code
  - Test coverage percentage
  - Number of passing tests
  - Features completed vs planned
  - Known issues/limitations
  - Suggestions for Phase 2 development
- Final `git commit` and tag v0.1.0
- Write evaluation report to `docs/evaluation-report.md`

---

## Quality Gates (checked after each phase)
1. All Go code passes `go vet` and `gofmt`
2. All existing tests pass
3. No compilation errors
4. Git commit after each phase with descriptive message
5. If a phase fails quality gate, fix before moving to next phase

## Recovery Strategy
- If a phase takes longer than allocated, reduce scope of that phase
- If tests fail, fix them before committing
- If stuck on a problem > 15 min, simplify the approach and move on
- All progress is saved via git commits, so no work is lost
