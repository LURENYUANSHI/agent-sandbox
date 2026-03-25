package types

import (
	"context"
)

// SandboxStatus represents the current lifecycle state of a sandbox.
type SandboxStatus string

const (
	StatusCreated SandboxStatus = "created"
	StatusRunning SandboxStatus = "running"
	StatusStopped SandboxStatus = "stopped"
	StatusError   SandboxStatus = "error"
)

// Sandbox defines the interface for managing a sandbox lifecycle.
type Sandbox interface {
	ID() string
	Start(ctx context.Context) error
	Execute(ctx context.Context, action Action) (*ActionResult, error)
	Stop(ctx context.Context) error
	Status() SandboxStatus
}
