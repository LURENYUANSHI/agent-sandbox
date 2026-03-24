package executor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestExecuteAllowedFileRead(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0o644)

	p := &types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectAllow,
	}
	engine := policy.NewEngine(p)
	store := trace.NewStore()
	recorder := trace.NewRecorder(store)
	exec := NewExecutor(engine, recorder, tmpDir)

	action := &types.Action{
		Type:   types.ActionTypeFile,
		Path:   testFile,
		FileOp: types.FileOpRead,
	}

	event, err := exec.Execute("sb-1", action)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if event.Decision != types.DecisionAllowed {
		t.Errorf("decision = %s, want allowed", event.Decision)
	}
	if event.Result != "hello" {
		t.Errorf("result = %q, want %q", event.Result, "hello")
	}
}

func TestExecuteDeniedAction(t *testing.T) {
	p := &types.Policy{
		Name:          "deny-all",
		DefaultEffect: types.EffectDeny,
	}
	engine := policy.NewEngine(p)
	store := trace.NewStore()
	recorder := trace.NewRecorder(store)
	exec := NewExecutor(engine, recorder, t.TempDir())

	action := &types.Action{
		Type:   types.ActionTypeFile,
		Path:   "/etc/passwd",
		FileOp: types.FileOpDelete,
	}

	event, err := exec.Execute("sb-1", action)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if event.Decision != types.DecisionDenied {
		t.Errorf("decision = %s, want denied", event.Decision)
	}
}

func TestExecuteFileWrite(t *testing.T) {
	tmpDir := t.TempDir()

	p := &types.Policy{
		Name:          "allow-all",
		DefaultEffect: types.EffectAllow,
	}
	engine := policy.NewEngine(p)
	store := trace.NewStore()
	recorder := trace.NewRecorder(store)
	exec := NewExecutor(engine, recorder, tmpDir)

	action := &types.Action{
		Type:    types.ActionTypeFile,
		Path:    filepath.Join(tmpDir, "output.txt"),
		FileOp:  types.FileOpWrite,
		Content: "written by executor",
	}

	event, err := exec.Execute("sb-1", action)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if event.Decision != types.DecisionAllowed {
		t.Errorf("decision = %s, want allowed", event.Decision)
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if string(data) != "written by executor" {
		t.Errorf("file content = %q, want %q", string(data), "written by executor")
	}
}
