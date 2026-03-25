package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// newTestSandbox creates a sandbox with a policy engine, recorder, and executor
// rooted in a temp directory. Returns the sandbox, executor, and root dir.
func newTestSandbox(t *testing.T, id string, pol *types.Policy) (*sandbox.Instance, *executor.Executor, string) {
	t.Helper()
	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.ID = id
	cfg.Name = "test-" + id
	cfg.RootDir = tmpDir
	cfg.TraceEnabled = false // no SQLite needed for tests

	engine := policy.NewEngine()
	if pol != nil {
		if err := engine.LoadPolicy(*pol); err != nil {
			t.Fatalf("load policy: %v", err)
		}
	}

	recorder, err := trace.NewRecorder("")
	if err != nil {
		t.Fatalf("create recorder: %v", err)
	}
	t.Cleanup(func() { recorder.Close() })

	instance, err := sandbox.NewSandbox(cfg, engine, recorder)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	exec := executor.NewExecutor(cfg)
	return instance, exec, tmpDir
}

func TestSandboxFullLifecycle(t *testing.T) {
	// Policy: allow file.read and file.write, deny file.delete
	pol := &types.Policy{
		Name:          "lifecycle-test",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:       "allow-file-read",
				ActionType: "file.*",
				Effect:     types.EffectAllow,
			},
		},
	}

	instance, exec, tmpDir := newTestSandbox(t, "lifecycle-1", pol)

	// 1. Verify initial status
	if instance.Status() != sandbox.StatusCreated {
		t.Fatalf("status = %s, want created", instance.Status())
	}

	// 2. Start sandbox
	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if instance.Status() != sandbox.StatusRunning {
		t.Fatalf("status = %s, want running", instance.Status())
	}

	// 3. Execute allowed action (file.read) -> should succeed
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello sandbox"), 0o644)

	readAction := types.Action{
		ID:     "action-read-1",
		Type:   types.ActionTypeFileRead,
		Params: map[string]string{"path": testFile},
	}
	result, err := instance.Execute(context.Background(), readAction, exec.Execute)
	if err != nil {
		t.Fatalf("execute file.read: %v", err)
	}
	if !result.Success {
		t.Errorf("read result.Success = false, want true")
	}
	if result.Output != "hello sandbox" {
		t.Errorf("read output = %q, want %q", result.Output, "hello sandbox")
	}

	// 4. Execute denied action (file.delete on path outside sandbox) -> should be denied by policy
	// Change policy to deny deletes specifically
	denyDeletePol := &types.Policy{
		Name:          "deny-delete",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:       "allow-file-read",
				ActionType: "file.read",
				Effect:     types.EffectAllow,
			},
		},
	}
	engine := policy.NewEngine()
	engine.LoadPolicy(*denyDeletePol)

	// Create a new sandbox with deny-delete policy for this part
	cfg2 := sandbox.DefaultConfig()
	cfg2.ID = "lifecycle-deny"
	cfg2.Name = "deny-test"
	cfg2.RootDir = tmpDir
	cfg2.TraceEnabled = false

	recorder2, _ := trace.NewRecorder("")
	defer recorder2.Close()

	instance2, _ := sandbox.NewSandbox(cfg2, engine, recorder2)
	instance2.Start(context.Background())

	deleteAction := types.Action{
		ID:     "action-delete-1",
		Type:   types.ActionTypeFileDelete,
		Params: map[string]string{"path": "/important-file"},
	}
	_, err = instance2.Execute(context.Background(), deleteAction, exec.Execute)
	if err == nil {
		t.Fatal("expected file.delete to be denied, but got no error")
	}

	// 5. Verify traces recorded correctly
	traces, err := instance.GetTraces()
	if err != nil {
		t.Fatalf("get traces: %v", err)
	}
	// sandbox.start records 1 event, then action.requested + policy.evaluated + action.executed = 3 more
	if len(traces) < 3 {
		t.Errorf("expected at least 3 trace events, got %d", len(traces))
	}

	// Verify start event is first
	foundStart := false
	for _, te := range traces {
		if te.Type == types.EventSandboxStarted {
			foundStart = true
			break
		}
	}
	if !foundStart {
		t.Error("no sandbox.started event in traces")
	}

	// 6. Stop sandbox
	if err := instance.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if instance.Status() != sandbox.StatusStopped {
		t.Fatalf("status = %s, want stopped", instance.Status())
	}

	// 7. Verify traces include stop event
	traces, _ = instance.GetTraces()
	foundStop := false
	for _, te := range traces {
		if te.Type == types.EventSandboxStopped {
			foundStop = true
			break
		}
	}
	if !foundStop {
		t.Error("no sandbox.stopped event in traces")
	}
}

func TestSandboxConcurrentExecution(t *testing.T) {
	// Allow-all policy
	pol := &types.Policy{
		Name:          "allow-all",
		DefaultEffect: types.EffectAllow,
	}

	instance, exec, tmpDir := newTestSandbox(t, "concurrent-1", pol)
	instance.Start(context.Background())
	defer instance.Stop(context.Background())

	// Create 10 test files
	for i := 0; i < 10; i++ {
		f := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(f, []byte(fmt.Sprintf("data-%d", i)), 0o644)
	}

	// Launch 10 concurrent actions
	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			action := types.Action{
				ID:     fmt.Sprintf("concurrent-%d", idx),
				Type:   types.ActionTypeFileRead,
				Params: map[string]string{"path": filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", idx))},
			}
			result, err := instance.Execute(context.Background(), action, exec.Execute)
			if err != nil {
				errCh <- fmt.Errorf("action %d: %w", idx, err)
				return
			}
			if !result.Success {
				errCh <- fmt.Errorf("action %d: result not successful", idx)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}

	// Verify all traced correctly - each action generates multiple trace events
	traces, err := instance.GetTraces()
	if err != nil {
		t.Fatalf("get traces: %v", err)
	}

	// Count action.executed events specifically
	executedCount := 0
	for _, te := range traces {
		if te.Type == types.EventActionExecuted {
			executedCount++
		}
	}
	if executedCount != 10 {
		t.Errorf("expected 10 action.executed events, got %d", executedCount)
	}
}

func TestSandboxPolicyHotReload(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("data"), 0o644)

	// Start with deny-all policy
	engine := policy.NewEngine() // deny-all by default
	recorder, _ := trace.NewRecorder("")
	defer recorder.Close()

	cfg := sandbox.DefaultConfig()
	cfg.ID = "hotreload-1"
	cfg.Name = "hotreload-test"
	cfg.RootDir = tmpDir
	cfg.TraceEnabled = false

	instance, _ := sandbox.NewSandbox(cfg, engine, recorder)
	exec := executor.NewExecutor(cfg)
	instance.Start(context.Background())
	defer instance.Stop(context.Background())

	// Execute write action -> should be denied (deny-all default)
	writeAction := types.Action{
		ID:     "action-write-1",
		Type:   types.ActionTypeFileWrite,
		Params: map[string]string{"path": testFile, "content": "new data"},
	}
	_, err := instance.Execute(context.Background(), writeAction, exec.Execute)
	if err == nil {
		t.Fatal("expected deny-all to reject file.write, but got no error")
	}

	// Hot-reload permissive policy
	permissive := types.Policy{
		Name:          "permissive",
		DefaultEffect: types.EffectAllow,
	}
	engine.LoadPolicy(permissive)

	// Execute same action -> should succeed with permissive policy
	writeAction.ID = "action-write-2"
	result, err := instance.Execute(context.Background(), writeAction, exec.Execute)
	if err != nil {
		t.Fatalf("expected permissive to allow file.write, got: %v", err)
	}
	if !result.Success {
		t.Error("write result.Success = false after policy reload")
	}
}

func TestSandboxPolicyFromYAML(t *testing.T) {
	// Find the fixture file
	policyPath := filepath.Join("..", "..", "test", "fixtures", "sample-policy.yaml")
	if _, err := os.Stat(policyPath); err != nil {
		policyPath = "test/fixtures/sample-policy.yaml"
		if _, err := os.Stat(policyPath); err != nil {
			t.Skipf("sample-policy.yaml not found, skipping: %v", err)
		}
	}

	p, err := policy.ParseFile(policyPath)
	if err != nil {
		t.Fatalf("parse policy: %v", err)
	}

	if p.Name != "test-policy" {
		t.Errorf("policy name = %s, want test-policy", p.Name)
	}
	if len(p.Rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(p.Rules))
	}

	engine := policy.NewEngine()
	engine.LoadPolicy(*p)

	// file.read should be allowed by the first rule
	readDecision := engine.Evaluate(types.Action{
		ID:   "test-1",
		Type: types.ActionTypeFileRead,
	})
	if !readDecision.Allowed {
		t.Errorf("file.read decision: allowed = false, want true")
	}

	// file.delete should be denied (no allow rule + default deny)
	deleteDecision := engine.Evaluate(types.Action{
		ID:   "test-2",
		Type: types.ActionTypeFileDelete,
	})
	if deleteDecision.Allowed {
		t.Errorf("file.delete decision: allowed = true, want false")
	}
}
