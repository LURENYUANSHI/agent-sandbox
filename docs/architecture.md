# Architecture

## Overview

AgentSandbox is a runtime security layer for AI agents. It intercepts every action an agent attempts, evaluates it against a policy engine, executes it within an isolated sandbox, and records a complete trace for audit and replay.

## Component Diagram

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
```

## Request Flow

1. **Agent submits action** via REST API or CLI
2. **Policy Engine** evaluates the action against YAML-defined rules
3. If **denied**, the action is blocked and the denial is recorded as a trace event
4. If **allowed**, the **Executor** runs the action inside the sandbox
5. **Trace Recorder** captures the action, decision, inputs, outputs, and timing
6. Traces are stored in **SQLite** and optionally exported via **OpenTelemetry**

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Isolation | OS namespaces (Linux) / job objects (Windows) | Strong isolation without full VM overhead |
| Policy format | YAML | Human-readable, version-controllable, declarative |
| Trace format | OpenTelemetry spans | Industry standard, compatible with existing tools |
| Trace storage | SQLite | Embedded, zero-config, sufficient for single-node MVP |
| API framework | Gin | Lightweight, fast, well-documented Go HTTP framework |
| CLI framework | Cobra | Standard Go CLI library with subcommand support |
| Frontend | React + Vite | Fast builds, strong ecosystem, TypeScript support |

## Package Dependencies

```
cmd/sandbox  ──→  pkg/sandbox  ──→  pkg/executor  ──→  pkg/types
cmd/server   ──→  pkg/api      ──→  pkg/sandbox        pkg/policy
                                ──→  pkg/trace     ──→  pkg/types
```

## Security Model

- **Default deny**: Actions not explicitly allowed by policy are rejected
- **Pre-execution evaluation**: Policy is checked before any action executes
- **Complete audit trail**: Every action and decision is recorded
- **Filesystem isolation**: Sandboxes have a restricted filesystem view
- **Network filtering**: Outbound connections are controlled per-sandbox
- **Process limits**: CPU, memory, and process count limits enforced
