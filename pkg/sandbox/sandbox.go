package sandbox

import (
	"fmt"
	"os"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// State represents the lifecycle state of a sandbox.
type State string

const (
	StateCreated   State = "created"
	StateRunning   State = "running"
	StateStopped   State = "stopped"
	StateDestroyed State = "destroyed"
)

// Sandbox manages an isolated execution environment.
type Sandbox struct {
	ID       string `json:"id"`
	State    State  `json:"state"`
	BasePath string `json:"base_path"`

	executor *executor.Executor
	engine   *policy.Engine
	store    *trace.Store
	recorder *trace.Recorder
	replayer *trace.Replayer
	mu       sync.Mutex
}

// New creates a sandbox from the given config.
func New(cfg *Config) (*Sandbox, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("sandbox ID is required")
	}

	p := cfg.Policy
	if p == nil {
		p = policy.DefaultPolicy()
	}

	basePath := cfg.BasePath
	if basePath == "" {
		basePath = os.TempDir()
	}

	engine := policy.NewEngine(p)
	store := trace.NewStore()
	recorder := trace.NewRecorder(store)
	replayer := trace.NewReplayer(store)
	exec := executor.NewExecutor(engine, recorder, basePath)

	return &Sandbox{
		ID:       cfg.ID,
		State:    StateCreated,
		BasePath: basePath,
		executor: exec,
		engine:   engine,
		store:    store,
		recorder: recorder,
		replayer: replayer,
	}, nil
}

// Start transitions the sandbox to running state.
func (s *Sandbox) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateCreated && s.State != StateStopped {
		return fmt.Errorf("cannot start sandbox in state %s", s.State)
	}
	s.State = StateRunning
	return nil
}

// Execute runs an action inside the sandbox.
func (s *Sandbox) Execute(action *types.Action) (*types.TraceEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateRunning {
		return nil, fmt.Errorf("sandbox is not running (state: %s)", s.State)
	}
	return s.executor.Execute(s.ID, action)
}

// Stop transitions the sandbox to stopped state.
func (s *Sandbox) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != StateRunning {
		return fmt.Errorf("cannot stop sandbox in state %s", s.State)
	}
	s.State = StateStopped
	return nil
}

// Destroy permanently destroys the sandbox.
func (s *Sandbox) Destroy() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == StateDestroyed {
		return fmt.Errorf("sandbox already destroyed")
	}
	s.State = StateDestroyed
	return nil
}

// GetTraces returns all trace events for this sandbox.
func (s *Sandbox) GetTraces() ([]*types.TraceEvent, error) {
	return s.recorder.GetEvents(s.ID)
}

// ReplayTraces returns trace events in chronological order.
func (s *Sandbox) ReplayTraces() ([]*types.TraceEvent, error) {
	return s.replayer.Replay(s.ID)
}
