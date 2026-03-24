package executor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func testConfig(t *testing.T) sandbox.Config {
	t.Helper()
	return sandbox.Config{
		ID:             "test-exec",
		RootDir:        t.TempDir(),
		MaxMemoryMB:    512,
		MaxCPUPercent:  50,
		MaxDiskMB:      1024,
		MaxProcesses:   10,
		TimeoutSeconds: 5,
		NetworkEnabled: false,
	}
}

// --- Filesystem tests ---

func TestFilesystem_ReadWriteDelete(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir)
	ctx := context.Background()

	// Write a file
	writeAction := types.Action{
		ID:   "w1",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    "hello.txt",
			"content": "hello world",
		},
	}
	result, err := fs.ExecuteFileWrite(ctx, writeAction)
	if err != nil {
		t.Fatalf("FileWrite: %v", err)
	}
	if !result.Success {
		t.Error("expected write success")
	}

	// Read it back
	readAction := types.Action{
		ID:   "r1",
		Type: types.ActionTypeFileRead,
		Params: map[string]string{
			"path": "hello.txt",
		},
	}
	result, err = fs.ExecuteFileRead(ctx, readAction)
	if err != nil {
		t.Fatalf("FileRead: %v", err)
	}
	if result.Output != "hello world" {
		t.Errorf("expected 'hello world', got %q", result.Output)
	}

	// Delete it
	delAction := types.Action{
		ID:   "d1",
		Type: types.ActionTypeFileDelete,
		Params: map[string]string{
			"path": "hello.txt",
		},
	}
	result, err = fs.ExecuteFileDelete(ctx, delAction)
	if err != nil {
		t.Fatalf("FileDelete: %v", err)
	}
	if !result.Success {
		t.Error("expected delete success")
	}

	// Verify deleted
	_, err = fs.ExecuteFileRead(ctx, readAction)
	if err == nil {
		t.Error("expected error reading deleted file")
	}
}

func TestFilesystem_PathEscape(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir)
	ctx := context.Background()

	// Attempt directory traversal
	escapeAction := types.Action{
		ID:   "escape",
		Type: types.ActionTypeFileRead,
		Params: map[string]string{
			"path": "../../etc/passwd",
		},
	}
	_, err := fs.ExecuteFileRead(ctx, escapeAction)
	if err == nil {
		t.Error("expected error for path escape attempt")
	}
}

func TestFilesystem_NestedDirectories(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir)
	ctx := context.Background()

	writeAction := types.Action{
		ID:   "nested",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    "a/b/c/deep.txt",
			"content": "nested content",
		},
	}
	_, err := fs.ExecuteFileWrite(ctx, writeAction)
	if err != nil {
		t.Fatalf("nested write: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(cfg.RootDir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("read nested file: %v", err)
	}
	if string(data) != "nested content" {
		t.Errorf("got %q", string(data))
	}
}

func TestFilesystem_DeleteSandboxRoot(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir)
	ctx := context.Background()

	delAction := types.Action{
		ID:   "del-root",
		Type: types.ActionTypeFileDelete,
		Params: map[string]string{
			"path": ".",
		},
	}
	_, err := fs.ExecuteFileDelete(ctx, delAction)
	if err == nil {
		t.Error("expected error when deleting sandbox root")
	}
}

func TestFilesystem_MissingParams(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir)
	ctx := context.Background()

	_, err := fs.ExecuteFileRead(ctx, types.Action{ID: "x", Params: map[string]string{}})
	if err == nil {
		t.Error("expected error for missing path param")
	}

	_, err = fs.ExecuteFileWrite(ctx, types.Action{ID: "x", Params: map[string]string{}})
	if err == nil {
		t.Error("expected error for missing path param on write")
	}

	_, err = fs.ExecuteFileDelete(ctx, types.Action{ID: "x", Params: map[string]string{}})
	if err == nil {
		t.Error("expected error for missing path param on delete")
	}
}

// --- Process tests ---

func TestProcess_SimpleCommand(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 5*time.Second)
	ctx := context.Background()

	var action types.Action
	if runtime.GOOS == "windows" {
		action = types.Action{
			ID:   "echo",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "cmd",
				"args":    "/c echo hello",
			},
		}
	} else {
		action = types.Action{
			ID:   "echo",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "echo",
				"args":    "hello",
			},
		}
	}

	result, err := proc.ExecuteProcess(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteProcess: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, output: %s", result.Output)
	}
}

func TestProcess_Timeout(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 500*time.Millisecond)
	ctx := context.Background()

	var action types.Action
	if runtime.GOOS == "windows" {
		action = types.Action{
			ID:   "sleep",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "cmd",
				"args":    "/c ping -n 3 127.0.0.1",
			},
		}
	} else {
		action = types.Action{
			ID:   "sleep",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "sleep",
				"args":    "10",
			},
		}
	}

	result, err := proc.ExecuteProcess(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteProcess: %v", err)
	}
	if result.Success {
		t.Error("expected timeout failure")
	}
}

func TestProcess_Shell(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 5*time.Second)
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo hello"
	} else {
		cmd = "echo hello"
	}

	action := types.Action{
		ID:   "shell",
		Type: types.ActionTypeShell,
		Params: map[string]string{
			"command": cmd,
		},
	}

	result, err := proc.ExecuteShell(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteShell: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got exit code %d", result.ExitCode)
	}
}

// --- Network tests ---

func TestNetwork_DisabledRejects(t *testing.T) {
	net := NewNetworkExecutor(false)
	ctx := context.Background()

	action := types.Action{
		ID:   "http",
		Type: types.ActionTypeNetHTTP,
		Params: map[string]string{
			"url": "http://example.com",
		},
	}
	_, err := net.ExecuteNetHTTP(ctx, action)
	if err == nil {
		t.Error("expected error when network is disabled")
	}

	connAction := types.Action{
		ID:   "conn",
		Type: types.ActionTypeNetConnect,
		Params: map[string]string{
			"host": "example.com",
			"port": "80",
		},
	}
	_, err = net.ExecuteNetConnect(ctx, connAction)
	if err == nil {
		t.Error("expected error when network is disabled")
	}
}

func TestNetwork_MissingParams(t *testing.T) {
	net := NewNetworkExecutor(true)
	ctx := context.Background()

	_, err := net.ExecuteNetHTTP(ctx, types.Action{ID: "x", Params: map[string]string{}})
	if err == nil {
		t.Error("expected error for missing url")
	}

	_, err = net.ExecuteNetConnect(ctx, types.Action{ID: "x", Params: map[string]string{"host": "a"}})
	if err == nil {
		t.Error("expected error for missing port")
	}
}

// --- Executor dispatch tests ---

func TestExecutor_Dispatch(t *testing.T) {
	cfg := testConfig(t)
	exec := NewExecutor(cfg)
	ctx := context.Background()

	// Write a file through the executor
	writeAction := types.Action{
		ID:   "dispatch-write",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    "dispatched.txt",
			"content": "via executor",
		},
	}
	result, err := exec.Execute(ctx, writeAction)
	if err != nil {
		t.Fatalf("Execute file write: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Read it back
	readAction := types.Action{
		ID:   "dispatch-read",
		Type: types.ActionTypeFileRead,
		Params: map[string]string{
			"path": "dispatched.txt",
		},
	}
	result, err = exec.Execute(ctx, readAction)
	if err != nil {
		t.Fatalf("Execute file read: %v", err)
	}
	if result.Output != "via executor" {
		t.Errorf("expected 'via executor', got %q", result.Output)
	}
}

func TestExecutor_UnsupportedType(t *testing.T) {
	cfg := testConfig(t)
	exec := NewExecutor(cfg)

	action := types.Action{
		ID:   "bad",
		Type: "unknown.type",
	}
	_, err := exec.Execute(context.Background(), action)
	if err == nil {
		t.Error("expected error for unsupported action type")
	}
}
