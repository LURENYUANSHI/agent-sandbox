package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// FilesystemExecutor handles file read/write/delete within a sandbox root.
type FilesystemExecutor struct {
	rootDir      string
	maxReadSize  int64
	maxWriteSize int64
}

// NewFilesystemExecutor creates a filesystem executor rooted at dir.
func NewFilesystemExecutor(rootDir string, cfg config.ExecutorConfig) *FilesystemExecutor {
	return &FilesystemExecutor{
		rootDir:      rootDir,
		maxReadSize:  int64(cfg.MaxReadSizeMB) * 1024 * 1024,
		maxWriteSize: int64(cfg.MaxWriteSizeMB) * 1024 * 1024,
	}
}

// resolvePath validates that the target path is inside the sandbox root.
// It resolves symlinks and prevents directory traversal.
func (f *FilesystemExecutor) resolvePath(target string) (string, error) {
	// Reject paths containing null bytes
	if strings.ContainsRune(target, '\x00') {
		return "", fmt.Errorf("path contains null byte")
	}

	// Make path absolute relative to sandbox root
	var absPath string
	if filepath.IsAbs(target) {
		absPath = target
	} else {
		absPath = filepath.Join(f.rootDir, target)
	}

	// Clean the path to remove .. components
	absPath = filepath.Clean(absPath)

	// Resolve symlinks if the path exists
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file doesn't exist yet (e.g., for writes), validate the cleaned path
		if os.IsNotExist(err) {
			resolved = absPath
		} else {
			return "", fmt.Errorf("resolve path: %w", err)
		}
	}

	// Ensure resolved path is within sandbox root
	rootAbs, err := filepath.Abs(f.rootDir)
	if err != nil {
		return "", fmt.Errorf("resolve root dir: %w", err)
	}
	rootAbs = filepath.Clean(rootAbs)

	if !strings.HasPrefix(resolved, rootAbs+string(filepath.Separator)) && resolved != rootAbs {
		return "", fmt.Errorf("path %q escapes sandbox root %q", target, f.rootDir)
	}

	return resolved, nil
}

// ExecuteFileRead reads a file and returns its content.
func (f *FilesystemExecutor) ExecuteFileRead(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	path := action.Params["path"]
	if path == "" {
		return nil, fmt.Errorf("file.read requires 'path' parameter")
	}

	resolved, err := f.resolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if info.Size() > f.maxReadSize {
		return nil, fmt.Errorf("file size %d exceeds max read size %d", info.Size(), f.maxReadSize)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return &types.ActionResult{
		ActionID:  action.ID,
		Success:   true,
		Output:    string(data),
		BytesRead: int64(len(data)),
	}, nil
}

// ExecuteFileWrite writes content to a file.
func (f *FilesystemExecutor) ExecuteFileWrite(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	path := action.Params["path"]
	content := action.Params["content"]
	if path == "" {
		return nil, fmt.Errorf("file.write requires 'path' parameter")
	}

	if int64(len(content)) > f.maxWriteSize {
		return nil, fmt.Errorf("content size %d exceeds max write size %d", len(content), f.maxWriteSize)
	}

	resolved, err := f.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return nil, fmt.Errorf("create parent directory: %w", err)
	}

	if err := os.WriteFile(resolved, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	return &types.ActionResult{
		ActionID:   action.ID,
		Success:    true,
		Output:     fmt.Sprintf("wrote %d bytes to %s", len(content), path),
		BytesWrite: int64(len(content)),
	}, nil
}

// ExecuteFileDelete removes a file.
func (f *FilesystemExecutor) ExecuteFileDelete(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	path := action.Params["path"]
	if path == "" {
		return nil, fmt.Errorf("file.delete requires 'path' parameter")
	}

	resolved, err := f.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Prevent deleting the sandbox root itself
	rootAbs, err := filepath.Abs(f.rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve root dir: %w", err)
	}
	if filepath.Clean(resolved) == filepath.Clean(rootAbs) {
		return nil, fmt.Errorf("cannot delete sandbox root directory")
	}

	if err := os.Remove(resolved); err != nil {
		return nil, fmt.Errorf("delete file: %w", err)
	}

	return &types.ActionResult{
		ActionID: action.ID,
		Success:  true,
		Output:   fmt.Sprintf("deleted %s", path),
	}, nil
}
