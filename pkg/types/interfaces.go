package types

import (
	"context"
)

// SandboxStatus represents the current lifecycle state of a sandbox.
type SandboxStatus string

const (
	// StatusCreated indicates the sandbox has been initialized but not started.
	StatusCreated SandboxStatus = "created"
	// StatusRunning indicates the sandbox is active and accepting actions.
	StatusRunning SandboxStatus = "running"
	// StatusStopped indicates the sandbox has been stopped.
	StatusStopped SandboxStatus = "stopped"
	// StatusError indicates the sandbox encountered a fatal error.
	StatusError SandboxStatus = "error"
)

// Sandbox defines the interface for managing a sandbox lifecycle.
// A sandbox provides an isolated execution environment for AI agent actions.
type Sandbox interface {
	// ID returns the unique identifier of this sandbox.
	ID() string
	// Start initializes the sandbox environment and begins accepting actions.
	Start(ctx context.Context) error
	// Execute runs an action within the sandbox boundaries, returning its result.
	Execute(ctx context.Context, action Action) (*ActionResult, error)
	// Stop shuts down the sandbox, releasing all resources.
	Stop(ctx context.Context) error
	// Status returns the current lifecycle state of the sandbox.
	Status() SandboxStatus
}
