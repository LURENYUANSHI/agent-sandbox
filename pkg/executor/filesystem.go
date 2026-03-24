package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func executeFile(basePath string, action *types.Action) (string, error) {
	path := resolvePath(basePath, action.Path)

	switch action.FileOp {
	case types.FileOpRead:
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading file %s: %w", path, err)
		}
		return string(data), nil

	case types.FileOpWrite:
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("creating directory %s: %w", dir, err)
		}
		if err := os.WriteFile(path, []byte(action.Content), 0o644); err != nil {
			return "", fmt.Errorf("writing file %s: %w", path, err)
		}
		return fmt.Sprintf("wrote %d bytes to %s", len(action.Content), path), nil

	case types.FileOpDelete:
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("deleting file %s: %w", path, err)
		}
		return fmt.Sprintf("deleted %s", path), nil

	case types.FileOpList:
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", fmt.Errorf("listing directory %s: %w", path, err)
		}
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		return strings.Join(names, "\n"), nil

	default:
		return "", fmt.Errorf("unknown file operation: %s", action.FileOp)
	}
}

func resolvePath(basePath, actionPath string) string {
	if filepath.IsAbs(actionPath) {
		return actionPath
	}
	return filepath.Join(basePath, actionPath)
}
