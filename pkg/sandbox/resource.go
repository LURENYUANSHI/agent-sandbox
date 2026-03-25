package sandbox

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ResourceMonitor tracks resource usage per sandbox and enforces configured limits.
type ResourceMonitor struct {
	mu        sync.Mutex
	config    Config
	processes map[string]int // sandboxID → active process count
}

// NewResourceMonitor creates a ResourceMonitor for the given sandbox config.
func NewResourceMonitor(config Config) *ResourceMonitor {
	return &ResourceMonitor{
		config:    config,
		processes: make(map[string]int),
	}
}

// CheckDiskUsage calculates total disk usage in bytes under sandboxRoot.
func (rm *ResourceMonitor) CheckDiskUsage(sandboxRoot string) (int64, error) {
	var total int64
	err := filepath.WalkDir(sandboxRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("check disk usage %s: %w", sandboxRoot, err)
	}
	return total, nil
}

// CheckProcessCount returns the tracked process count for a sandbox.
func (rm *ResourceMonitor) CheckProcessCount(sandboxID string) (int, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.processes[sandboxID], nil
}

// IncrementProcessCount adds one to the tracked process count.
func (rm *ResourceMonitor) IncrementProcessCount(sandboxID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.processes[sandboxID]++
}

// DecrementProcessCount subtracts one from the tracked process count.
func (rm *ResourceMonitor) DecrementProcessCount(sandboxID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.processes[sandboxID] > 0 {
		rm.processes[sandboxID]--
	}
}

// EnforceLimit checks whether the given action would exceed resource limits.
// Returns a descriptive error if a limit would be exceeded, nil otherwise.
func (rm *ResourceMonitor) EnforceLimit(action types.Action, config Config) error {
	switch action.Type {
	case types.ActionFileWrite, types.ActionTypeFileWrite:
		return rm.enforceDiskLimit(action, config)
	case types.ActionProcExec, types.ActionTypeProcess, types.ActionTypeShell, types.ActionShellExec:
		return rm.enforceProcessLimit(action, config)
	}
	return nil
}

func (rm *ResourceMonitor) enforceDiskLimit(action types.Action, config Config) error {
	if config.MaxDiskMB <= 0 {
		return nil
	}

	currentUsage, err := rm.CheckDiskUsage(config.RootDir)
	if err != nil {
		return fmt.Errorf("check disk usage: %w", err)
	}

	// Estimate new file size from action content
	var newSize int64
	if content, ok := action.Params["content"]; ok {
		newSize = int64(len(content))
	}

	limitBytes := int64(config.MaxDiskMB) * 1024 * 1024
	if currentUsage+newSize > limitBytes {
		return fmt.Errorf("disk limit exceeded: current %d bytes + %d bytes new > %d MB limit",
			currentUsage, newSize, config.MaxDiskMB)
	}
	return nil
}

func (rm *ResourceMonitor) enforceProcessLimit(action types.Action, config Config) error {
	if config.MaxProcesses <= 0 {
		return nil
	}

	count, err := rm.CheckProcessCount(config.ID)
	if err != nil {
		return fmt.Errorf("check process count: %w", err)
	}

	if count >= config.MaxProcesses {
		return fmt.Errorf("process limit exceeded: %d running processes >= %d max",
			count, config.MaxProcesses)
	}
	return nil
}
