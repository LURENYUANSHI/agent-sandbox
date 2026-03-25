package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Executor dispatches actions to the appropriate handler based on ActionType.
type Executor struct {
	config sandbox.Config
	fs     *FilesystemExecutor
	net    *NetworkExecutor
	proc   *ProcessExecutor
}

// NewExecutor creates an executor bound to a sandbox configuration.
func NewExecutor(sbxCfg sandbox.Config, execCfg config.ExecutorConfig) *Executor {
	return &Executor{
		config: sbxCfg,
		fs:     NewFilesystemExecutor(sbxCfg.RootDir, execCfg),
		net:    NewNetworkExecutor(sbxCfg.NetworkEnabled, execCfg),
		proc:   NewProcessExecutor(sbxCfg.RootDir, time.Duration(sbxCfg.TimeoutSeconds)*time.Second),
	}
}

// Execute dispatches the action to the correct sub-executor.
func (e *Executor) Execute(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	start := time.Now()

	var result *types.ActionResult
	var err error

	switch action.Type {
	case types.ActionTypeFileRead:
		result, err = e.fs.ExecuteFileRead(ctx, action)
	case types.ActionTypeFileWrite:
		result, err = e.fs.ExecuteFileWrite(ctx, action)
	case types.ActionTypeFileDelete:
		result, err = e.fs.ExecuteFileDelete(ctx, action)
	case types.ActionTypeNetHTTP:
		result, err = e.net.ExecuteNetHTTP(ctx, action)
	case types.ActionTypeNetConnect:
		result, err = e.net.ExecuteNetConnect(ctx, action)
	case types.ActionTypeProcess:
		result, err = e.proc.ExecuteProcess(ctx, action)
	case types.ActionTypeShell:
		result, err = e.proc.ExecuteShell(ctx, action)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", action.Type)
	}

	if err != nil {
		return nil, err
	}
	result.Duration = time.Since(start)
	return result, nil
}
