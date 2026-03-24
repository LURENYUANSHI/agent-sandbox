package executor

import (
	"fmt"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Executor runs actions within a sandbox, checking policy and recording traces.
type Executor struct {
	policyEngine *policy.Engine
	recorder     *trace.Recorder
	basePath     string
	nextID       int
}

// NewExecutor creates an executor bound to a policy engine and trace recorder.
func NewExecutor(engine *policy.Engine, recorder *trace.Recorder, basePath string) *Executor {
	return &Executor{
		policyEngine: engine,
		recorder:     recorder,
		basePath:     basePath,
	}
}

// Execute runs an action after policy evaluation, recording the trace event.
func (e *Executor) Execute(sandboxID string, action *types.Action) (*types.TraceEvent, error) {
	e.nextID++
	startTime := time.Now()

	if action.Timestamp.IsZero() {
		action.Timestamp = startTime
	}

	effect, reason := e.policyEngine.Evaluate(action)

	event := &types.TraceEvent{
		ID:        fmt.Sprintf("evt-%s-%d", sandboxID, e.nextID),
		TraceID:   fmt.Sprintf("trace-%s", sandboxID),
		SandboxID: sandboxID,
		Action:    *action,
		Reason:    reason,
		StartTime: startTime,
	}

	if effect == types.EffectDeny {
		event.Decision = types.DecisionDenied
		event.Error = fmt.Sprintf("action denied by rule: %s", reason)
		event.EndTime = time.Now()
		event.DurationMs = event.EndTime.Sub(event.StartTime).Milliseconds()
		if err := e.recorder.Record(event); err != nil {
			return nil, fmt.Errorf("recording denied event: %w", err)
		}
		return event, nil
	}

	event.Decision = types.DecisionAllowed

	var execErr error
	switch action.Type {
	case types.ActionTypeFile:
		event.Result, execErr = executeFile(e.basePath, action)
	case types.ActionTypeNetwork:
		event.Result, execErr = executeNetwork(action)
	case types.ActionTypeProcess:
		event.Result, execErr = executeProcess(action)
	case types.ActionTypeShell:
		event.Result, execErr = executeProcess(action)
	default:
		execErr = fmt.Errorf("unknown action type: %s", action.Type)
	}

	if execErr != nil {
		event.Error = execErr.Error()
	}

	event.EndTime = time.Now()
	event.DurationMs = event.EndTime.Sub(event.StartTime).Milliseconds()

	if err := e.recorder.Record(event); err != nil {
		return nil, fmt.Errorf("recording event: %w", err)
	}

	return event, nil
}

// ReloadPolicy hot-reloads the policy engine with a new policy.
func (e *Executor) ReloadPolicy(p *types.Policy) {
	e.policyEngine.LoadPolicy(p)
}
