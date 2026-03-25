package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestCheckDiskUsage(t *testing.T) {
	dir := t.TempDir()
	rm := NewResourceMonitor(Config{RootDir: dir})

	// Empty dir should be 0
	usage, err := rm.CheckDiskUsage(dir)
	if err != nil {
		t.Fatalf("CheckDiskUsage: %v", err)
	}
	if usage != 0 {
		t.Errorf("expected 0 bytes for empty dir, got %d", usage)
	}

	// Write a file and check again
	data := []byte("hello world")
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), data, 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	usage, err = rm.CheckDiskUsage(dir)
	if err != nil {
		t.Fatalf("CheckDiskUsage: %v", err)
	}
	if usage != int64(len(data)) {
		t.Errorf("expected %d bytes, got %d", len(data), usage)
	}

	// Subdirectory with file
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0o755)
	data2 := []byte("more data")
	os.WriteFile(filepath.Join(subdir, "nested.txt"), data2, 0o644)

	usage, err = rm.CheckDiskUsage(dir)
	if err != nil {
		t.Fatalf("CheckDiskUsage: %v", err)
	}
	expected := int64(len(data) + len(data2))
	if usage != expected {
		t.Errorf("expected %d bytes, got %d", expected, usage)
	}
}

func TestCheckDiskUsage_NonexistentDir(t *testing.T) {
	rm := NewResourceMonitor(Config{})
	_, err := rm.CheckDiskUsage("/nonexistent/path/xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestCheckProcessCount(t *testing.T) {
	rm := NewResourceMonitor(Config{ID: "sb-1"})

	count, err := rm.CheckProcessCount("sb-1")
	if err != nil {
		t.Fatalf("CheckProcessCount: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	rm.IncrementProcessCount("sb-1")
	rm.IncrementProcessCount("sb-1")
	count, _ = rm.CheckProcessCount("sb-1")
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	rm.DecrementProcessCount("sb-1")
	count, _ = rm.CheckProcessCount("sb-1")
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	// Decrement below zero should stay at 0
	rm.DecrementProcessCount("sb-1")
	rm.DecrementProcessCount("sb-1")
	count, _ = rm.CheckProcessCount("sb-1")
	if count != 0 {
		t.Errorf("expected 0 after double decrement, got %d", count)
	}
}

func TestEnforceLimit_DiskUsage(t *testing.T) {
	dir := t.TempDir()

	// Write 500 bytes
	os.WriteFile(filepath.Join(dir, "existing.bin"), make([]byte, 500), 0o644)

	cfg := Config{
		ID:      "sb-disk",
		RootDir: dir,
		MaxDiskMB: 1, // 1 MB limit
	}
	rm := NewResourceMonitor(cfg)

	// Small write within limit should pass
	action := types.Action{
		ID:     "write-1",
		Type:   types.ActionFileWrite,
		Params: map[string]string{"content": "small"},
	}
	if err := rm.EnforceLimit(action, cfg); err != nil {
		t.Errorf("expected small write to pass: %v", err)
	}

	// Write that would exceed limit
	bigContent := make([]byte, 2*1024*1024) // 2 MB
	action2 := types.Action{
		ID:     "write-2",
		Type:   types.ActionFileWrite,
		Params: map[string]string{"content": string(bigContent)},
	}
	if err := rm.EnforceLimit(action2, cfg); err == nil {
		t.Error("expected disk limit error for large write")
	}
}

func TestEnforceLimit_ProcessCount(t *testing.T) {
	cfg := Config{
		ID:           "sb-proc",
		RootDir:      t.TempDir(),
		MaxProcesses: 2,
	}
	rm := NewResourceMonitor(cfg)

	action := types.Action{
		ID:   "exec-1",
		Type: types.ActionProcExec,
	}

	// Should pass with 0 processes
	if err := rm.EnforceLimit(action, cfg); err != nil {
		t.Errorf("expected pass with 0 procs: %v", err)
	}

	// Add processes up to limit
	rm.IncrementProcessCount("sb-proc")
	rm.IncrementProcessCount("sb-proc")

	// Should fail at limit
	if err := rm.EnforceLimit(action, cfg); err == nil {
		t.Error("expected process limit error")
	}
}

func TestEnforceLimit_UnrelatedAction(t *testing.T) {
	cfg := Config{
		ID:           "sb-unrelated",
		RootDir:      t.TempDir(),
		MaxDiskMB:    1,
		MaxProcesses: 2,
	}
	rm := NewResourceMonitor(cfg)

	// file:read should not be limited
	action := types.Action{
		ID:   "read-1",
		Type: types.ActionFileRead,
	}
	if err := rm.EnforceLimit(action, cfg); err != nil {
		t.Errorf("file:read should not be limited: %v", err)
	}

	// net:connect should not be limited
	action2 := types.Action{
		ID:   "net-1",
		Type: types.ActionNetConnect,
	}
	if err := rm.EnforceLimit(action2, cfg); err != nil {
		t.Errorf("net:connect should not be limited: %v", err)
	}
}

func TestEnforceLimit_DotSeparatedActionTypes(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		ID:           "sb-dot",
		RootDir:      dir,
		MaxDiskMB:    1,
		MaxProcesses: 1,
	}
	rm := NewResourceMonitor(cfg)

	// file.write (dot-separated) should also be checked
	action := types.Action{
		ID:     "write-dot",
		Type:   types.ActionTypeFileWrite,
		Params: map[string]string{"content": "ok"},
	}
	if err := rm.EnforceLimit(action, cfg); err != nil {
		t.Errorf("dot-separated file.write should pass: %v", err)
	}

	// process.exec (dot-separated) should also be checked
	rm.IncrementProcessCount("sb-dot")
	procAction := types.Action{
		ID:   "proc-dot",
		Type: types.ActionTypeProcess,
	}
	if err := rm.EnforceLimit(procAction, cfg); err == nil {
		t.Error("expected process limit error for dot-separated type")
	}
}
