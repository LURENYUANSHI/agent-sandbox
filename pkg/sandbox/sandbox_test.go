package sandbox

import (
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestSandboxLifecycle(t *testing.T) {
	sb, err := New(&Config{ID: "test-1", Policy: policy.DefaultPolicy()})
	if err != nil {
		t.Fatalf("new sandbox: %v", err)
	}

	if sb.State != StateCreated {
		t.Fatalf("state = %s, want created", sb.State)
	}

	if err := sb.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if sb.State != StateRunning {
		t.Fatalf("state = %s, want running", sb.State)
	}

	if err := sb.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if sb.State != StateStopped {
		t.Fatalf("state = %s, want stopped", sb.State)
	}

	if err := sb.Destroy(); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if sb.State != StateDestroyed {
		t.Fatalf("state = %s, want destroyed", sb.State)
	}
}

func TestSandboxExecute(t *testing.T) {
	sb, _ := New(&Config{
		ID:       "test-2",
		BasePath: t.TempDir(),
		Policy:   policy.DefaultPolicy(),
	})
	sb.Start()

	event, err := sb.Execute(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   "/tmp/test.txt",
		FileOp: types.FileOpRead,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if event.Decision != types.DecisionAllowed {
		t.Errorf("decision = %s, want allowed", event.Decision)
	}
}

func TestSandboxExecuteWhenNotRunning(t *testing.T) {
	sb, _ := New(&Config{ID: "test-3", Policy: policy.DefaultPolicy()})

	_, err := sb.Execute(&types.Action{
		Type:   types.ActionTypeFile,
		Path:   "/tmp/test.txt",
		FileOp: types.FileOpRead,
	})
	if err == nil {
		t.Fatal("expected error executing on non-running sandbox")
	}
}

func TestSandboxInvalidStateTransitions(t *testing.T) {
	sb, _ := New(&Config{ID: "test-4", Policy: policy.DefaultPolicy()})

	if err := sb.Stop(); err == nil {
		t.Error("expected error stopping created sandbox")
	}

	sb.Start()
	if err := sb.Start(); err == nil {
		t.Error("expected error starting running sandbox")
	}

	sb.Stop()
	sb.Destroy()
	if err := sb.Destroy(); err == nil {
		t.Error("expected error destroying already destroyed sandbox")
	}
}
