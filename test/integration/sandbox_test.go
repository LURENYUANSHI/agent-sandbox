package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestSandboxFullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("sandbox test data"), 0o644)

	// Use a policy that allows file reads/writes in the actual temp directory
	testPolicy := &types.Policy{
		Name:          "lifecycle-test",
		Description:   "Policy for lifecycle integration test",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:    "allow-tmpdir-read",
				Effect:  types.EffectAllow,
				Actions: []types.ActionType{types.ActionTypeFile},
				Paths:   []string{filepath.Join(tmpDir, "*")},
				FileOps: []types.FileOp{types.FileOpRead, types.FileOpList, types.FileOpWrite},
			},
		},
	}

	// 1. Create sandbox with test policy
	sb, err := sandbox.New(&sandbox.Config{
		ID:       "lifecycle-test",
		BasePath: tmpDir,
		Policy:   testPolicy,
	})
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	if sb.State != sandbox.StateCreated {
		t.Fatalf("state = %s, want created", sb.State)
	}

	// 2. Start sandbox
	if err := sb.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if sb.State != sandbox.StateRunning {
		t.Fatalf("state = %s, want running", sb.State)
	}

	// 3. Execute allowed action (file read in /tmp) -> should succeed
	readEvent, err := sb.Execute(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   testFile,
		FileOp: types.FileOpRead,
	})
	if err != nil {
		t.Fatalf("execute read: %v", err)
	}
	if readEvent.Decision != types.DecisionAllowed {
		t.Errorf("read decision = %s, want allowed", readEvent.Decision)
	}
	if readEvent.Result != "sandbox test data" {
		t.Errorf("read result = %q, want %q", readEvent.Result, "sandbox test data")
	}

	// 4. Execute denied action (file delete on /) -> should be denied
	deleteEvent, err := sb.Execute(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   "/important-file",
		FileOp: types.FileOpDelete,
	})
	if err != nil {
		t.Fatalf("execute delete: %v", err)
	}
	if deleteEvent.Decision != types.DecisionDenied {
		t.Errorf("delete decision = %s, want denied", deleteEvent.Decision)
	}

	// 5. Verify traces recorded correctly
	traces, err := sb.GetTraces()
	if err != nil {
		t.Fatalf("get traces: %v", err)
	}
	if len(traces) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(traces))
	}
	if traces[0].Decision != types.DecisionAllowed {
		t.Errorf("trace[0] decision = %s, want allowed", traces[0].Decision)
	}
	if traces[1].Decision != types.DecisionDenied {
		t.Errorf("trace[1] decision = %s, want denied", traces[1].Decision)
	}

	// 6. Stop sandbox
	if err := sb.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if sb.State != sandbox.StateStopped {
		t.Fatalf("state = %s, want stopped", sb.State)
	}

	// 7. Replay traces and verify order
	replayed, err := sb.ReplayTraces()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(replayed) != 2 {
		t.Fatalf("replayed %d events, want 2", len(replayed))
	}
	if !replayed[0].StartTime.Before(replayed[1].StartTime) &&
		!replayed[0].StartTime.Equal(replayed[1].StartTime) {
		t.Error("replayed events not in chronological order")
	}
}

func TestSandboxConcurrentExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 10; i++ {
		f := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		os.WriteFile(f, []byte("data"), 0o644)
	}

	p := &types.Policy{
		Name:          "allow-all",
		DefaultEffect: types.EffectAllow,
	}

	sb, err := sandbox.New(&sandbox.Config{
		ID:       "concurrent-test",
		BasePath: tmpDir,
		Policy:   p,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sb.Start()
	defer sb.Stop()

	// Launch 10 concurrent actions
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			f := filepath.Join(tmpDir, "file"+string(rune('0'+idx))+".txt")
			_, err := sb.Execute(&types.Action{
				Type:   types.ActionTypeFile,
				Path:   f,
				FileOp: types.FileOpRead,
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent execution error: %v", err)
	}

	// Verify all traced correctly
	traces, err := sb.GetTraces()
	if err != nil {
		t.Fatalf("get traces: %v", err)
	}
	if len(traces) != 10 {
		t.Errorf("expected 10 traces, got %d", len(traces))
	}
}

func TestSandboxPolicyHotReload(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("data"), 0o644)

	// Start with strict policy
	sb, err := sandbox.New(&sandbox.Config{
		ID:       "hotreload-test",
		BasePath: tmpDir,
		Policy:   policy.StrictPolicy(),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sb.Start()
	defer sb.Stop()

	// Execute write action -> should be denied (strict only allows reads)
	event1, err := sb.Execute(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   testFile,
		FileOp: types.FileOpWrite,
	})
	if err != nil {
		t.Fatalf("execute write: %v", err)
	}
	if event1.Decision != types.DecisionDenied {
		t.Errorf("strict policy: write decision = %s, want denied", event1.Decision)
	}

	// Hot-reload permissive policy
	sb.ReloadPolicy(policy.PermissivePolicy())

	// Execute same action -> should succeed with permissive policy
	event2, err := sb.Execute(&types.Action{
		Type:    types.ActionTypeFile,
		Path:    testFile,
		FileOp:  types.FileOpWrite,
		Content: "updated content",
	})
	if err != nil {
		t.Fatalf("execute write after reload: %v", err)
	}
	if event2.Decision != types.DecisionAllowed {
		t.Errorf("permissive policy: write decision = %s, want allowed", event2.Decision)
	}
}

func TestSandboxPolicyFromYAML(t *testing.T) {
	policyPath := filepath.Join("..", "..", "test", "fixtures", "sample-policy.yaml")
	// Try relative path from test/integration directory
	if _, err := os.Stat(policyPath); err != nil {
		// Try from project root
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

	engine := policy.NewEngine(p)
	effect, _ := engine.Evaluate(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   "/tmp/test.txt",
		FileOp: types.FileOpRead,
	})
	if effect != types.EffectAllow {
		t.Errorf("tmp read effect = %s, want allow", effect)
	}

	effect, _ = engine.Evaluate(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   "/etc/passwd",
		FileOp: types.FileOpDelete,
	})
	if effect != types.EffectDeny {
		t.Errorf("root delete effect = %s, want deny", effect)
	}
}
