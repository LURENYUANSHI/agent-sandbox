package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ProcessExecutor runs processes and shell commands within the sandbox.
type ProcessExecutor struct {
	workDir string
	timeout time.Duration
}

// NewProcessExecutor creates a process executor.
func NewProcessExecutor(workDir string, timeout time.Duration) *ProcessExecutor {
	return &ProcessExecutor{workDir: workDir, timeout: timeout}
}

// ExecuteProcess runs a command with arguments.
func (p *ProcessExecutor) ExecuteProcess(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	command := action.Params["command"]
	if command == "" {
		return nil, fmt.Errorf("process.exec requires 'command' parameter")
	}

	args := splitArgs(action.Params["args"])

	execCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, command, args...)
	cmd.Dir = p.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return &types.ActionResult{
				ActionID: action.ID,
				Success:  false,
				Error:    "process timed out",
				Output:   stdout.String(),
				ExitCode: -1,
			}, nil
		} else {
			return nil, fmt.Errorf("run process: %w", err)
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n--- stderr ---\n" + stderr.String()
	}

	return &types.ActionResult{
		ActionID: action.ID,
		Success:  exitCode == 0,
		Output:   output,
		ExitCode: exitCode,
	}, nil
}

// ExecuteShell runs a command through the system shell.
func (p *ProcessExecutor) ExecuteShell(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	command := action.Params["command"]
	if command == "" {
		return nil, fmt.Errorf("shell.exec requires 'command' parameter")
	}

	shell, flag := shellCommand()

	// Wrap as a process execution
	shellAction := types.Action{
		ID:        action.ID,
		Type:      types.ActionTypeProcess,
		Timestamp: action.Timestamp,
		Params: map[string]string{
			"command": shell,
			"args":    flag + " " + command,
		},
	}

	return p.ExecuteProcess(ctx, shellAction)
}

func shellCommand() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd", "/c"
	}
	return "sh", "-c"
}

func splitArgs(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}
