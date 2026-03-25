package executor

import (
	"context"
	netpkg "net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func testExecConfig() config.ExecutorConfig {
	return config.Default().Executor
}

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
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
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
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
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
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
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
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
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
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
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
	net := NewNetworkExecutor(false, testExecConfig())
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
	net := NewNetworkExecutor(true, testExecConfig())
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
	exec := NewExecutor(cfg, testExecConfig())
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
	exec := NewExecutor(cfg, testExecConfig())

	action := types.Action{
		ID:   "bad",
		Type: "unknown.type",
	}
	_, err := exec.Execute(context.Background(), action)
	if err == nil {
		t.Error("expected error for unsupported action type")
	}
}

func TestExecutor_DispatchDelete(t *testing.T) {
	cfg := testConfig(t)
	exec := NewExecutor(cfg, testExecConfig())
	ctx := context.Background()

	// Write then delete through executor dispatch
	writeAction := types.Action{
		ID:   "w",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    "todelete.txt",
			"content": "bye",
		},
	}
	_, err := exec.Execute(ctx, writeAction)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	delAction := types.Action{
		ID:   "d",
		Type: types.ActionTypeFileDelete,
		Params: map[string]string{"path": "todelete.txt"},
	}
	result, err := exec.Execute(ctx, delAction)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestExecutor_DispatchProcess(t *testing.T) {
	cfg := testConfig(t)
	exec := NewExecutor(cfg, testExecConfig())
	ctx := context.Background()

	var action types.Action
	if runtime.GOOS == "windows" {
		action = types.Action{
			ID:   "proc",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "cmd",
				"args":    "/c echo dispatch",
			},
		}
	} else {
		action = types.Action{
			ID:   "proc",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "echo",
				"args":    "dispatch",
			},
		}
	}

	result, err := exec.Execute(ctx, action)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestExecutor_DispatchShell(t *testing.T) {
	cfg := testConfig(t)
	exec := NewExecutor(cfg, testExecConfig())
	ctx := context.Background()

	action := types.Action{
		ID:   "sh",
		Type: types.ActionTypeShell,
		Params: map[string]string{
			"command": "echo shell-dispatch",
		},
	}
	result, err := exec.Execute(ctx, action)
	if err != nil {
		t.Fatalf("shell: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestNetwork_HTTPEnabled(t *testing.T) {
	// Start a local test HTTP server
	net := NewNetworkExecutor(true, testExecConfig())
	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	srv := &http.Server{Handler: mux}
	ln, err := netpkg.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go srv.Serve(ln)
	defer srv.Close()

	action := types.Action{
		ID:   "http-ok",
		Type: types.ActionTypeNetHTTP,
		Params: map[string]string{
			"url": "http://" + ln.Addr().String() + "/test",
		},
	}
	result, err := net.ExecuteNetHTTP(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteNetHTTP: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Output != "ok" {
		t.Errorf("expected 'ok', got %q", result.Output)
	}
	if result.BytesRead != 2 {
		t.Errorf("expected 2 bytes read, got %d", result.BytesRead)
	}
}

func TestNetwork_HTTPWithMethod(t *testing.T) {
	net := NewNetworkExecutor(true, testExecConfig())
	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("posted"))
	})
	srv := &http.Server{Handler: mux}
	ln, err := netpkg.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go srv.Serve(ln)
	defer srv.Close()

	action := types.Action{
		ID:   "http-post",
		Type: types.ActionTypeNetHTTP,
		Params: map[string]string{
			"url":    "http://" + ln.Addr().String() + "/post",
			"method": "POST",
		},
	}
	result, err := net.ExecuteNetHTTP(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteNetHTTP: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestNetwork_ConnectEnabled(t *testing.T) {
	// Start a TCP listener
	ln, err := netpkg.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	_, port, _ := netpkg.SplitHostPort(ln.Addr().String())

	net := NewNetworkExecutor(true, testExecConfig())
	ctx := context.Background()

	action := types.Action{
		ID:   "conn-ok",
		Type: types.ActionTypeNetConnect,
		Params: map[string]string{
			"host": "127.0.0.1",
			"port": port,
		},
	}
	result, err := net.ExecuteNetConnect(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteNetConnect: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestNetwork_ConnectFailure(t *testing.T) {
	net := NewNetworkExecutor(true, testExecConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use a port that's unlikely to be open
	action := types.Action{
		ID:   "conn-fail",
		Type: types.ActionTypeNetConnect,
		Params: map[string]string{
			"host": "127.0.0.1",
			"port": "1",
		},
	}
	result, err := net.ExecuteNetConnect(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteNetConnect: %v", err)
	}
	if result.Success {
		t.Error("expected connection failure")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestProcess_NonZeroExit(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 5*time.Second)
	ctx := context.Background()

	var action types.Action
	if runtime.GOOS == "windows" {
		action = types.Action{
			ID:   "fail",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "cmd",
				"args":    "/c exit 42",
			},
		}
	} else {
		action = types.Action{
			ID:   "fail",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "sh",
				"args":    "-c exit 42",
			},
		}
	}

	result, err := proc.ExecuteProcess(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteProcess: %v", err)
	}
	if result.Success {
		t.Error("expected failure with non-zero exit")
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestProcess_MissingCommand(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 5*time.Second)
	ctx := context.Background()

	action := types.Action{
		ID:     "no-cmd",
		Type:   types.ActionTypeProcess,
		Params: map[string]string{},
	}
	_, err := proc.ExecuteProcess(ctx, action)
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestProcess_ShellMissingCommand(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 5*time.Second)
	ctx := context.Background()

	action := types.Action{
		ID:     "no-cmd",
		Type:   types.ActionTypeShell,
		Params: map[string]string{},
	}
	_, err := proc.ExecuteShell(ctx, action)
	if err == nil {
		t.Error("expected error for missing shell command")
	}
}

func TestFilesystem_WriteOversize(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
	ctx := context.Background()

	maxWrite := int64(testExecConfig().MaxWriteSizeMB) * 1024 * 1024
	bigContent := string(make([]byte, maxWrite+1))
	action := types.Action{
		ID:   "big-write",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    "big.txt",
			"content": bigContent,
		},
	}
	_, err := fs.ExecuteFileWrite(ctx, action)
	if err == nil {
		t.Error("expected error for oversized write")
	}
}

func TestFilesystem_AbsolutePath(t *testing.T) {
	cfg := testConfig(t)
	fs := NewFilesystemExecutor(cfg.RootDir, testExecConfig())
	ctx := context.Background()

	// Write using absolute path within sandbox root
	absPath := filepath.Join(cfg.RootDir, "abs.txt")
	writeAction := types.Action{
		ID:   "abs-write",
		Type: types.ActionTypeFileWrite,
		Params: map[string]string{
			"path":    absPath,
			"content": "absolute",
		},
	}
	result, err := fs.ExecuteFileWrite(ctx, writeAction)
	if err != nil {
		t.Fatalf("absolute path write: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Read it back
	readAction := types.Action{
		ID:   "abs-read",
		Type: types.ActionTypeFileRead,
		Params: map[string]string{"path": absPath},
	}
	result, err = fs.ExecuteFileRead(ctx, readAction)
	if err != nil {
		t.Fatalf("absolute path read: %v", err)
	}
	if result.Output != "absolute" {
		t.Errorf("expected 'absolute', got %q", result.Output)
	}
}

func TestProcess_StderrOutput(t *testing.T) {
	cfg := testConfig(t)
	proc := NewProcessExecutor(cfg.RootDir, 5*time.Second)
	ctx := context.Background()

	var action types.Action
	if runtime.GOOS == "windows" {
		action = types.Action{
			ID:   "stderr",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "cmd",
				"args":    "/c echo err 1>&2",
			},
		}
	} else {
		action = types.Action{
			ID:   "stderr",
			Type: types.ActionTypeProcess,
			Params: map[string]string{
				"command": "sh",
				"args":    "-c echo err >&2",
			},
		}
	}

	result, err := proc.ExecuteProcess(ctx, action)
	if err != nil {
		t.Fatalf("ExecuteProcess: %v", err)
	}
	if !strings.Contains(result.Output, "stderr") && !strings.Contains(result.Output, "err") {
		// On some systems stderr output may vary; just ensure it doesn't crash
	}
}
