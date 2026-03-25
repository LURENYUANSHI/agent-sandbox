package sandbox

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Status represents the lifecycle state of a sandbox.
type Status string

const (
	StatusCreated  Status = "created"
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
	StatusError    Status = "error"
)

// Instance is a running sandbox that executes actions under policy control.
type Instance struct {
	mu       sync.Mutex
	config   Config
	status   Status
	policy   types.PolicyEngine
	recorder types.TraceRecorder
	created  time.Time
}

// NewSandbox creates a new sandbox instance. Call Start() to prepare it for execution.
func NewSandbox(cfg Config, policyEngine types.PolicyEngine, recorder types.TraceRecorder) (*Instance, error) {
	if policyEngine == nil {
		return nil, fmt.Errorf("policy engine is required")
	}
	if recorder == nil {
		return nil, fmt.Errorf("trace recorder is required")
	}
	return &Instance{
		config:   cfg,
		status:   StatusCreated,
		policy:   policyEngine,
		recorder: recorder,
		created:  time.Now(),
	}, nil
}

// Start prepares the sandbox environment (creates root dir, etc.).
func (s *Instance) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusRunning {
		return fmt.Errorf("sandbox already running")
	}

	if err := os.MkdirAll(s.config.RootDir, 0o755); err != nil {
		s.status = StatusError
		return fmt.Errorf("create sandbox root dir: %w", err)
	}

	s.status = StatusRunning

	s.recorder.Record(types.TraceEvent{
		SandboxID: s.config.ID,
		Type:      types.EventSandboxStarted,
		Data: map[string]string{
			"root_dir": s.config.RootDir,
			"name":     s.config.Name,
		},
	})

	return nil
}

// Execute runs an action through the policy engine and, if allowed, dispatches it
// to the provided executeFn. This allows the sandbox to remain decoupled from the
// executor implementation.
func (s *Instance) Execute(ctx context.Context, action types.Action, executeFn func(ctx context.Context, action types.Action) (*types.ActionResult, error)) (*types.ActionResult, error) {
	s.mu.Lock()
	if s.status != StatusRunning {
		s.mu.Unlock()
		return nil, fmt.Errorf("sandbox is not running (status: %s)", s.status)
	}
	s.mu.Unlock()

	// 1. Record action requested
	s.recorder.Record(types.TraceEvent{
		SandboxID: s.config.ID,
		Type:      types.EventActionRequested,
		ActionID:  action.ID,
		Data:      map[string]string{"type": string(action.Type)},
	})

	// 2. Evaluate policy
	decision := s.policy.Evaluate(action)

	// 3. Record policy decision
	s.recorder.Record(types.TraceEvent{
		SandboxID: s.config.ID,
		Type:      types.EventPolicyEvaluated,
		ActionID:  action.ID,
		Data: map[string]string{
			"allowed": fmt.Sprintf("%t", decision.Allowed),
			"rule":    decision.Rule,
			"reason":  decision.Reason,
		},
	})

	// 4. If denied, record and return error
	if !decision.Allowed {
		s.recorder.Record(types.TraceEvent{
			SandboxID: s.config.ID,
			Type:      types.EventActionDenied,
			ActionID:  action.ID,
			Data:      map[string]string{"reason": decision.Reason},
		})
		return nil, fmt.Errorf("action denied by policy: %s", decision.Reason)
	}

	// 5. Execute with timeout
	timeout := time.Duration(s.config.TimeoutSeconds) * time.Second
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	result, err := executeFn(execCtx, action)
	elapsed := time.Since(start)

	// 6. Record outcome
	if err != nil {
		s.recorder.Record(types.TraceEvent{
			SandboxID: s.config.ID,
			Type:      types.EventActionFailed,
			ActionID:  action.ID,
			Duration:  elapsed,
			Data:      map[string]string{"error": err.Error()},
		})
		return nil, fmt.Errorf("execute action: %w", err)
	}

	s.recorder.Record(types.TraceEvent{
		SandboxID: s.config.ID,
		Type:      types.EventActionExecuted,
		ActionID:  action.ID,
		Duration:  elapsed,
		Data:      map[string]string{"success": fmt.Sprintf("%t", result.Success)},
	})

	return result, nil
}

// Stop shuts down the sandbox and records the event.
func (s *Instance) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != StatusRunning {
		return fmt.Errorf("sandbox is not running (status: %s)", s.status)
	}

	s.status = StatusStopped

	s.recorder.Record(types.TraceEvent{
		SandboxID: s.config.ID,
		Type:      types.EventSandboxStopped,
	})

	return nil
}

// Status returns the current sandbox status.
func (s *Instance) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Config returns the sandbox configuration.
func (s *Instance) Config() Config {
	return s.config
}

// GetTraces returns all trace events for this sandbox.
func (s *Instance) GetTraces() ([]types.TraceEvent, error) {
	return s.recorder.GetEvents(s.config.ID)
}
