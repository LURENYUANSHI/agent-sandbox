package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// --- mock policy engine ---

type mockPolicy struct {
	allowAll bool
}

func (m *mockPolicy) Evaluate(action types.Action) types.PolicyDecision {
	return types.PolicyDecision{
		Allowed: m.allowAll,
		Rule:    "mock",
		Reason:  "mock policy",
	}
}

func (m *mockPolicy) LoadPolicy(p types.Policy) error { return nil }

// --- mock recorder ---

type mockRecorder struct {
	mu     sync.Mutex
	events []types.TraceEvent
}

func (r *mockRecorder) Record(e types.TraceEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func (r *mockRecorder) GetEvents(sandboxID string) ([]types.TraceEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []types.TraceEvent
	for _, e := range r.events {
		if e.SandboxID == sandboxID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *mockRecorder) Close() error { return nil }

// --- helpers ---

func tempConfig(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	return Config{
		ID:             "test-sandbox-1",
		Name:           "test",
		RootDir:        filepath.Join(dir, "root"),
		MaxMemoryMB:    512,
		MaxCPUPercent:  50,
		MaxDiskMB:      1024,
		MaxProcesses:   10,
		TimeoutSeconds: 5,
		TraceEnabled:   true,
		TracePath:      filepath.Join(dir, "trace.db"),
	}
}

// --- tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxMemoryMB != 512 {
		t.Errorf("expected MaxMemoryMB=512, got %d", cfg.MaxMemoryMB)
	}
	if cfg.NetworkEnabled {
		t.Error("expected NetworkEnabled=false by default")
	}
	if !cfg.TraceEnabled {
		t.Error("expected TraceEnabled=true by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:    "missing root dir",
			modify:  func(c *Config) { c.RootDir = "" },
			wantErr: true,
		},
		{
			name:    "zero memory",
			modify:  func(c *Config) { c.MaxMemoryMB = 0 },
			wantErr: true,
		},
		{
			name:    "cpu over 100",
			modify:  func(c *Config) { c.MaxCPUPercent = 101 },
			wantErr: true,
		},
		{
			name:    "zero timeout",
			modify:  func(c *Config) { c.TimeoutSeconds = 0 },
			wantErr: true,
		},
		{
			name:    "trace enabled without path",
			modify:  func(c *Config) { c.TracePath = "" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tempConfig(t)
			tt.modify(&cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSandbox_Lifecycle(t *testing.T) {
	cfg := tempConfig(t)
	rec := &mockRecorder{}
	pol := &mockPolicy{allowAll: true}

	sb, err := NewSandbox(cfg, pol, rec)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	if sb.Status() != StatusCreated {
		t.Fatalf("expected status created, got %s", sb.Status())
	}

	// Start
	ctx := context.Background()
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if sb.Status() != StatusRunning {
		t.Fatalf("expected status running, got %s", sb.Status())
	}

	// Verify root dir was created
	if _, err := os.Stat(cfg.RootDir); os.IsNotExist(err) {
		t.Fatal("sandbox root dir was not created")
	}

	// Execute a simple action
	action := types.Action{
		ID:   "act-1",
		Type: types.ActionTypeFileRead,
		Params: map[string]string{
			"path": "test.txt",
		},
	}
	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		return &types.ActionResult{ActionID: a.ID, Success: true, Output: "hello"}, nil
	}

	result, err := sb.Execute(ctx, action, executeFn)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}

	// Stop
	if err := sb.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if sb.Status() != StatusStopped {
		t.Fatalf("expected status stopped, got %s", sb.Status())
	}

	// Check trace events recorded
	events, _ := rec.GetEvents(cfg.ID)
	if len(events) < 4 {
		t.Errorf("expected at least 4 trace events, got %d", len(events))
	}

	// Verify event types
	eventTypes := make(map[types.EventType]bool)
	for _, e := range events {
		eventTypes[e.Type] = true
	}
	expected := []types.EventType{
		types.EventSandboxStarted,
		types.EventActionRequested,
		types.EventPolicyEvaluated,
		types.EventActionExecuted,
		types.EventSandboxStopped,
	}
	for _, et := range expected {
		if !eventTypes[et] {
			t.Errorf("missing event type: %s", et)
		}
	}
}

func TestSandbox_PolicyDenied(t *testing.T) {
	cfg := tempConfig(t)
	rec := &mockRecorder{}
	pol := &mockPolicy{allowAll: false}

	sb, err := NewSandbox(cfg, pol, rec)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	ctx := context.Background()
	sb.Start(ctx)

	action := types.Action{
		ID:   "act-denied",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    "secret.txt",
			"content": "nope",
		},
	}

	executeCalled := false
	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		executeCalled = true
		return &types.ActionResult{ActionID: a.ID, Success: true}, nil
	}

	_, err = sb.Execute(ctx, action, executeFn)
	if err == nil {
		t.Fatal("expected error for denied action")
	}
	if executeCalled {
		t.Error("execute function should not have been called for denied action")
	}

	// Verify denied event recorded
	events, _ := rec.GetEvents(cfg.ID)
	hasDenied := false
	for _, e := range events {
		if e.Type == types.EventActionDenied {
			hasDenied = true
		}
	}
	if !hasDenied {
		t.Error("expected action.denied event")
	}
}

func TestSandbox_ConcurrentExecution(t *testing.T) {
	cfg := tempConfig(t)
	rec := &mockRecorder{}
	pol := &mockPolicy{allowAll: true}

	sb, err := NewSandbox(cfg, pol, rec)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	ctx := context.Background()
	sb.Start(ctx)

	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		time.Sleep(10 * time.Millisecond)
		return &types.ActionResult{ActionID: a.ID, Success: true}, nil
	}

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			action := types.Action{
				ID:     "concurrent-" + string(rune('0'+idx)),
				Type:   types.ActionTypeFileRead,
				Params: map[string]string{"path": "test.txt"},
			}
			_, errs[idx] = sb.Execute(ctx, action, executeFn)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent execution %d failed: %v", i, err)
		}
	}
}

func TestSandbox_ExecuteWhenNotRunning(t *testing.T) {
	cfg := tempConfig(t)
	rec := &mockRecorder{}
	pol := &mockPolicy{allowAll: true}

	sb, _ := NewSandbox(cfg, pol, rec)

	action := types.Action{ID: "x", Type: types.ActionTypeFileRead}
	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		return &types.ActionResult{}, nil
	}

	_, err := sb.Execute(context.Background(), action, executeFn)
	if err == nil {
		t.Fatal("expected error when executing on non-running sandbox")
	}
}

func TestNewSandbox_NilDeps(t *testing.T) {
	cfg := tempConfig(t)

	_, err := NewSandbox(cfg, nil, &mockRecorder{})
	if err == nil {
		t.Error("expected error for nil policy engine")
	}

	_, err = NewSandbox(cfg, &mockPolicy{}, nil)
	if err == nil {
		t.Error("expected error for nil recorder")
	}
}
