package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
)

// policyToExecType translates colon-separated policy types to dot-separated executor types.
func policyToExecType(t types.ActionType) types.ActionType {
	switch t {
	case types.ActionFileRead:
		return types.ActionTypeFileRead
	case types.ActionFileWrite:
		return types.ActionTypeFileWrite
	case types.ActionFileDelete:
		return types.ActionTypeFileDelete
	case types.ActionProcExec:
		return types.ActionTypeProcess
	default:
		return t
	}
}

// sampleGoSource is a badly-formatted Go file that the coding agent will fix.
const sampleGoSource = `package main

import "fmt"

func   add(a int,b int)int{
return a+b
}

func main(){
result:=add(3,   4)
fmt.Println("result:",result)
}
`

// modifiedGoSource is the version with a new function added (still needs formatting).
const modifiedGoSource = `package main

import "fmt"

func   add(a int,b int)int{
return a+b
}

func   multiply(a int,b int)int{
return a*b
}

func main(){
sum:=add(3,   4)
product:=multiply(3,4)
fmt.Println("sum:",sum,"product:",product)
}
`

func main() {
	fmt.Printf("\n%s%s╔══════════════════════════════════════════════════════╗%s\n", bold, cyan, reset)
	fmt.Printf("%s%s║      AgentSandbox — Coding Agent Demo                ║%s\n", bold, cyan, reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n\n", bold, cyan, reset)

	tmpDir, err := os.MkdirTemp("", "coding-agent-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Load the coding-agent policy.
	pol, err := policy.LoadFromFile("configs/examples/coding-agent.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load policy: %v\n", err)
		os.Exit(1)
	}
	policyEngine := policy.NewPolicyEngine(*pol)

	recorder, err := trace.NewRecorder("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create recorder: %v\n", err)
		os.Exit(1)
	}
	defer recorder.Close()

	cfg := sandbox.DefaultConfig()
	cfg.ID = "coding-agent-001"
	cfg.Name = "Coding Agent"
	cfg.RootDir = filepath.Join(tmpDir, "workspace")
	cfg.NetworkEnabled = false
	cfg.TraceEnabled = true
	cfg.TracePath = filepath.Join(tmpDir, "traces.db")

	sbx, err := sandbox.NewSandbox(cfg, policyEngine, recorder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create sandbox: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Printf("%s▶ Starting coding agent sandbox...%s\n", yellow, reset)
	if err := sbx.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start sandbox: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s✓ Sandbox started%s (Policy: %s)\n\n", green, reset, pol.Name)

	exec := executor.NewExecutor(cfg, config.Default().Executor)
	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		a.Type = policyToExecType(a.Type)
		return exec.Execute(ctx, a)
	}

	// Step 1: Write a badly-formatted Go source file.
	step("Step 1", "Writing sample Go source file (badly formatted)")
	result, err := sbx.Execute(ctx, types.Action{
		ID:       "write-source",
		Type:     types.ActionFileWrite,
		Resource: "/workspace/hello.go",
		Params:   map[string]string{"path": "hello.go", "content": sampleGoSource},
	}, executeFn)
	printResult(result, err)

	// Step 2: Read the source file back.
	step("Step 2", "Reading source file")
	result, err = sbx.Execute(ctx, types.Action{
		ID:       "read-source",
		Type:     types.ActionFileRead,
		Resource: "/workspace/hello.go",
		Params:   map[string]string{"path": "hello.go"},
	}, executeFn)
	printResult(result, err)
	if result != nil && result.Output != "" {
		fmt.Printf("       %s--- file content ---%s\n", blue, reset)
		for _, line := range strings.Split(result.Output, "\n") {
			fmt.Printf("       %s│%s %s\n", blue, reset, line)
		}
		fmt.Println()
	}

	// Step 3: Write the modified version with a new function.
	step("Step 3", "Writing modified source (adding multiply function)")
	result, err = sbx.Execute(ctx, types.Action{
		ID:       "write-modified",
		Type:     types.ActionFileWrite,
		Resource: "/workspace/hello.go",
		Params:   map[string]string{"path": "hello.go", "content": modifiedGoSource},
	}, executeFn)
	printResult(result, err)

	// Step 4: Run go fmt on the file.
	step("Step 4", "Running 'go fmt' on hello.go")
	result, err = sbx.Execute(ctx, types.Action{
		ID:       "go-fmt",
		Type:     types.ActionProcExec,
		Resource: "go",
		Params:   map[string]string{"command": "go", "args": "fmt hello.go"},
	}, executeFn)
	printResult(result, err)

	// Step 5: Read the formatted file.
	step("Step 5", "Reading formatted source file")
	result, err = sbx.Execute(ctx, types.Action{
		ID:       "read-formatted",
		Type:     types.ActionFileRead,
		Resource: "/workspace/hello.go",
		Params:   map[string]string{"path": "hello.go"},
	}, executeFn)
	printResult(result, err)
	if result != nil && result.Output != "" {
		fmt.Printf("       %s--- formatted file ---%s\n", green, reset)
		for _, line := range strings.Split(result.Output, "\n") {
			fmt.Printf("       %s│%s %s\n", green, reset, line)
		}
		fmt.Println()
	}

	// Stop sandbox.
	fmt.Printf("%s▶ Stopping sandbox...%s\n", yellow, reset)
	sbx.Stop(ctx)
	fmt.Printf("%s✓ Sandbox stopped%s\n\n", green, reset)

	// Display the trace.
	fmt.Printf("%s%s── Execution Trace ─────────────────────────────────%s\n\n", bold, blue, reset)
	traces, err := sbx.GetTraces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get traces: %v\n", err)
		return
	}
	for _, evt := range traces {
		ts := evt.Timestamp.Format("15:04:05.000")
		icon, color := traceStyle(evt.Type)

		detail := ""
		if t, ok := evt.Data["type"]; ok {
			detail = fmt.Sprintf(" [%s]", t)
		}
		if reason, ok := evt.Data["reason"]; ok {
			detail = fmt.Sprintf(" — %s", reason)
		}

		fmt.Printf("  %s%s%s %s  %-22s%s\n", color, icon, reset, ts, evt.Type, detail)
	}
	fmt.Println()

	allowed, denied := 0, 0
	for _, evt := range traces {
		switch evt.Type {
		case types.EventActionExecuted:
			allowed++
		case types.EventActionDenied:
			denied++
		}
	}
	fmt.Printf("%s%s── Summary ─────────────────────────────────────────%s\n", bold, blue, reset)
	fmt.Printf("  %sAllowed: %d%s   %sDenied: %d%s   Trace events: %d\n\n",
		green, allowed, reset, red, denied, reset, len(traces))
}

func step(label, description string) {
	fmt.Printf("%s%s%s%s: %s\n", bold, yellow, label, reset, description)
	time.Sleep(50 * time.Millisecond)
}

func printResult(result *types.ActionResult, err error) {
	if err != nil {
		fmt.Printf("       %s✗ DENIED%s — %s\n\n", red, reset, err.Error())
		return
	}
	fmt.Printf("       %s✓ OK%s\n", green, reset)
}

func traceStyle(t types.EventType) (string, string) {
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
