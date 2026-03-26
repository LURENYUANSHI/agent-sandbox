package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ANSI color codes.
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
)

// policyToExecType translates colon-separated policy action types to
// dot-separated executor action types.
func policyToExecType(t types.ActionType) types.ActionType {
	switch t {
	case types.ActionFileRead:
		return types.ActionTypeFileRead
	case types.ActionFileWrite:
		return types.ActionTypeFileWrite
	case types.ActionFileDelete:
		return types.ActionTypeFileDelete
	case types.ActionNetHTTP:
		return types.ActionTypeNetHTTP
	case types.ActionNetConnect:
		return types.ActionTypeNetConnect
	case types.ActionProcExec:
		return types.ActionTypeProcess
	case types.ActionShellExec:
		return types.ActionTypeShell
	default:
		return t
	}
}

type demoAction struct {
	description string
	action      types.Action
}

func main() {
	fmt.Printf("\n%s%s╔══════════════════════════════════════════════════════╗%s\n", bold, cyan, reset)
	fmt.Printf("%s%s║        AgentSandbox — Interactive Demo               ║%s\n", bold, cyan, reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n\n", bold, cyan, reset)

	// Create temp directory for sandbox root.
	tmpDir, err := os.MkdirTemp("", "sandbox-demo-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Load the default restrictive policy.
	pol, err := policy.LoadFromFile("configs/default-policy.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load policy: %v\n", err)
		os.Exit(1)
	}
	policyEngine := policy.NewPolicyEngine(*pol)

	// Create in-memory trace recorder.
	recorder, err := trace.NewRecorder("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create recorder: %v\n", err)
		os.Exit(1)
	}
	defer recorder.Close()

	// Configure the sandbox.
	cfg := sandbox.DefaultConfig()
	cfg.ID = "demo-sandbox-001"
	cfg.Name = "Interactive Demo"
	cfg.RootDir = filepath.Join(tmpDir, "root")
	cfg.NetworkEnabled = true
	cfg.TraceEnabled = true
	cfg.TracePath = filepath.Join(tmpDir, "traces.db")

	sbx, err := sandbox.NewSandbox(cfg, policyEngine, recorder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create sandbox: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Printf("%s▶ Starting sandbox...%s\n", yellow, reset)
	if err := sbx.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start sandbox: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s✓ Sandbox started%s (ID: %s, Policy: %s)\n\n", green, reset, cfg.ID, pol.Name)

	// Create the executor bound to this sandbox config.
	exec := executor.NewExecutor(cfg, config.Default().Executor)

	// executeFn translates policy types to executor types before dispatch.
	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		a.Type = policyToExecType(a.Type)
		return exec.Execute(ctx, a)
	}

	// Define the demo actions.
	actions := []demoAction{
		{
			description: "Write 'Hello, Sandbox!' to /tmp/hello.txt",
			action: types.Action{
				ID:       "action-001",
				Type:     types.ActionFileWrite,
				Resource: "/tmp/hello.txt",
				Params:   map[string]string{"path": "hello.txt", "content": "Hello, Sandbox!\n"},
			},
		},
		{
			description: "Read from /tmp/hello.txt",
			action: types.Action{
				ID:       "action-002",
				Type:     types.ActionFileRead,
				Resource: "/tmp/hello.txt",
				Params:   map[string]string{"path": "hello.txt"},
			},
		},
		{
			description: "Delete /etc/passwd (should be DENIED)",
			action: types.Action{
				ID:       "action-003",
				Type:     types.ActionFileDelete,
				Resource: "/etc/passwd",
				Params:   map[string]string{"path": "/etc/passwd"},
			},
		},
		{
			description: "Shell exec 'echo hello' (should be DENIED)",
			action: types.Action{
				ID:       "action-004",
				Type:     types.ActionShellExec,
				Resource: "echo hello",
				Params:   map[string]string{"command": "echo hello"},
			},
		},
		{
			description: "HTTP GET to api.github.com (allowed by policy)",
			action: types.Action{
				ID:       "action-005",
				Type:     types.ActionNetHTTP,
				Resource: "api.github.com",
				Params:   map[string]string{"url": "https://api.github.com", "method": "GET"},
			},
		},
		{
			description: "Execute 'ls' command (allowed by policy)",
			action: types.Action{
				ID:       "action-006",
				Type:     types.ActionProcExec,
				Resource: "ls",
				Params:   map[string]string{"command": "ls"},
			},
		},
	}

	// Run each action through the sandbox.
	fmt.Printf("%s%s── Running Demo Actions ─────────────────────────────%s\n\n", bold, blue, reset)

	allowed := 0
	denied := 0

	for i, da := range actions {
		fmt.Printf("%s[%d/%d]%s %s\n", bold, i+1, len(actions), reset, da.description)
		fmt.Printf("       Action: %-12s  Resource: %s\n", da.action.Type, da.action.Resource)

		result, err := sbx.Execute(ctx, da.action, executeFn)
		if err != nil {
			denied++
			fmt.Printf("       %s✗ DENIED%s — %s\n\n", red, reset, err.Error())
		} else {
			allowed++
			output := result.Output
			if len(output) > 80 {
				output = output[:80] + "..."
			}
			output = trimNewline(output)
			fmt.Printf("       %s✓ ALLOWED%s", green, reset)
			if output != "" {
				fmt.Printf(" — %s", output)
			}
			fmt.Printf("\n\n")
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Stop sandbox.
	fmt.Printf("%s▶ Stopping sandbox...%s\n", yellow, reset)
	if err := sbx.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	fmt.Printf("%s✓ Sandbox stopped%s\n\n", green, reset)

	// Display trace replay.
	fmt.Printf("%s%s── Trace Replay ────────────────────────────────────%s\n\n", bold, blue, reset)

	traces, err := sbx.GetTraces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get traces: %v\n", err)
	} else {
		for _, evt := range traces {
			ts := evt.Timestamp.Format("15:04:05.000")
			icon, color := traceIcon(evt.Type)

			detail := ""
			if t, ok := evt.Data["type"]; ok {
				detail = fmt.Sprintf(" [%s]", t)
			}
			if reason, ok := evt.Data["reason"]; ok {
				detail = fmt.Sprintf(" — %s", reason)
			}
			if errMsg, ok := evt.Data["error"]; ok {
				detail = fmt.Sprintf(" — %s", errMsg)
			}

			fmt.Printf("  %s%s%s %s  %-22s%s\n", color, icon, reset, ts, evt.Type, detail)
		}
		fmt.Println()
	}

	// Summary stats.
	total := allowed + denied
	fmt.Printf("%s%s── Summary ─────────────────────────────────────────%s\n", bold, blue, reset)
	fmt.Printf("  Total actions:  %d\n", total)
	fmt.Printf("  %sAllowed:        %d%s\n", green, allowed, reset)
	fmt.Printf("  %sDenied:         %d%s\n", red, denied, reset)
	fmt.Printf("  Trace events:   %d\n\n", len(traces))
}

func traceIcon(t types.EventType) (string, string) {
	switch t {
	case types.EventSandboxStarted, types.EventSandboxStopped:
		return "◆", yellow
	case types.EventActionDenied, types.EventActionFailed:
		return "✗", red
	case types.EventActionExecuted:
		return "✓", green
	case types.EventPolicyEvaluated:
		return "⚖", blue
	default:
		return "•", cyan
	}
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
