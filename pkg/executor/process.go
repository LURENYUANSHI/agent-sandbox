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

	// Split command into executable and args if no separate args provided
	var cmdName string
	var args []string
	if action.Params["args"] != "" {
		cmdName = command
		args = splitArgs(action.Params["args"])
	} else {
		parts := splitArgs(command)
		if len(parts) == 0 {
			return nil, fmt.Errorf("empty command")
		}
		cmdName = parts[0]
		args = parts[1:]
	}

	execCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, cmdName, args...)
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
// The command is passed as a single argument to the shell's -c/\/c flag
// so that it is interpreted as a complete command string.
func (p *ProcessExecutor) ExecuteShell(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	command := action.Params["command"]
	if command == "" {
		return nil, fmt.Errorf("shell.exec requires 'command' parameter")
	}

	shell, flag := shellCommand()

	execCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, shell, flag, command)
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
			return nil, fmt.Errorf("run shell command: %w", err)
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

func shellCommand() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd", "/c"
	}
	return "sh", "-c"
}

// parseArgs splits a command-line string into arguments, respecting
// single and double quoted strings. e.g. `echo "hello world"` becomes
// ["echo", "hello world"].
func parseArgs(s string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range s {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}
		switch r {
		case '"', '\'':
			quote = r
		case ' ', '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func splitArgs(s string) []string {
	if s == "" {
		return nil
	}
	return parseArgs(s)
}
