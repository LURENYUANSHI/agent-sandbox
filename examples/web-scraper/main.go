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

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
)

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

func main() {
	fmt.Printf("\n%s%s╔══════════════════════════════════════════════════════╗%s\n", bold, cyan, reset)
	fmt.Printf("%s%s║      AgentSandbox — Web Scraper Demo                 ║%s\n", bold, cyan, reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n\n", bold, cyan, reset)

	tmpDir, err := os.MkdirTemp("", "web-scraper-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Create a custom policy: permissive network, restricted filesystem.
	scraperPolicy := types.Policy{
		Name:          "web-scraper",
		Version:       "1.0",
		Description:   "Permissive network access with restricted filesystem",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				ID:        "allow-http-all",
				Name:      "Allow all HTTP requests",
				Actions:   []string{"net:http"},
				Resources: []string{"**"},
				Effect:    types.EffectAllow,
				Priority:  10,
			},
			{
				ID:        "allow-local-file-rw",
				Name:      "Allow read/write in sandbox results directory",
				Actions:   []string{"file:read", "file:write"},
				Resources: []string{"/sandbox/**", "/results/**"},
				Effect:    types.EffectAllow,
				Priority:  10,
			},
			{
				ID:        "deny-shell",
				Name:      "Deny shell execution",
				Actions:   []string{"shell:exec"},
				Resources: []string{"**"},
				Effect:    types.EffectDeny,
				Priority:  100,
			},
		},
	}

	policyEngine := policy.NewPolicyEngine(scraperPolicy)

	recorder, err := trace.NewRecorder("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create recorder: %v\n", err)
		os.Exit(1)
	}
	defer recorder.Close()

	cfg := sandbox.DefaultConfig()
	cfg.ID = "web-scraper-001"
	cfg.Name = "Web Scraper"
	cfg.RootDir = filepath.Join(tmpDir, "scraper-root")
	cfg.NetworkEnabled = true
	cfg.TraceEnabled = true
	cfg.TracePath = filepath.Join(tmpDir, "traces.db")

	sbx, err := sandbox.NewSandbox(cfg, policyEngine, recorder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create sandbox: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Printf("%s▶ Starting web scraper sandbox...%s\n", yellow, reset)
	if err := sbx.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start sandbox: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s✓ Sandbox started%s (Policy: %s)\n\n", green, reset, scraperPolicy.Name)

	exec := executor.NewExecutor(cfg, config.Default().Executor)
	executeFn := func(ctx context.Context, a types.Action) (*types.ActionResult, error) {
		a.Type = policyToExecType(a.Type)
		return exec.Execute(ctx, a)
	}

	allowed := 0
	denied := 0

	// Target URLs to scrape.
	urls := []struct {
		name string
		url  string
	}{
		{"GitHub API", "https://api.github.com"},
		{"GitHub Status", "https://www.githubstatus.com/api/v2/status.json"},
	}

	fmt.Printf("%s%s── Scraping URLs ───────────────────────────────────%s\n\n", bold, blue, reset)

	for i, u := range urls {
		actionID := fmt.Sprintf("fetch-%d", i+1)
		fmt.Printf("%s[%d/%d]%s Fetching %s\n", bold, i+1, len(urls), reset, u.name)
		fmt.Printf("       URL: %s\n", u.url)

		result, err := sbx.Execute(ctx, types.Action{
			ID:       actionID,
			Type:     types.ActionNetHTTP,
			Resource: u.url,
			Params:   map[string]string{"url": u.url, "method": "GET"},
		}, executeFn)

		if err != nil {
			denied++
			fmt.Printf("       %s✗ FAILED%s — %s\n\n", red, reset, err.Error())
			continue
		}
		allowed++

		preview := result.Output
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("       %s✓ OK%s (%d bytes)\n", green, reset, len(result.Output))
		fmt.Printf("       Preview: %s\n\n", preview)

		// Write the result to a sandbox-local file.
		outFile := fmt.Sprintf("result-%d.json", i+1)
		fmt.Printf("       Saving to /results/%s\n", outFile)
		writeResult, err := sbx.Execute(ctx, types.Action{
			ID:       fmt.Sprintf("save-%d", i+1),
			Type:     types.ActionFileWrite,
			Resource: "/results/" + outFile,
			Params:   map[string]string{"path": outFile, "content": result.Output},
		}, executeFn)
		if err != nil {
			denied++
			fmt.Printf("       %s✗ DENIED%s — %s\n\n", red, reset, err.Error())
		} else {
			allowed++
			fmt.Printf("       %s✓ Saved%s (%d bytes written)\n\n", green, reset, writeResult.BytesWrite)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Attempt to write outside the sandbox (should be denied by policy).
	fmt.Printf("%s%s── Testing Sandbox Boundaries ──────────────────────%s\n\n", bold, blue, reset)

	fmt.Printf("%s[!]%s Attempting to write outside sandbox (/etc/scraped.json)\n", bold, reset)
	_, err = sbx.Execute(ctx, types.Action{
		ID:       "escape-attempt",
		Type:     types.ActionFileWrite,
		Resource: "/etc/scraped.json",
		Params:   map[string]string{"path": "/etc/scraped.json", "content": "should not be here"},
	}, executeFn)
	if err != nil {
		denied++
		fmt.Printf("       %s✗ DENIED%s — %s\n", red, reset, err.Error())
		fmt.Printf("       %s(Sandbox boundary enforced!)%s\n\n", yellow, reset)
	} else {
		allowed++
		fmt.Printf("       %s✓ ALLOWED%s (unexpected)\n\n", green, reset)
	}

	// Stop sandbox.
	fmt.Printf("%s▶ Stopping sandbox...%s\n", yellow, reset)
	sbx.Stop(ctx)
	fmt.Printf("%s✓ Sandbox stopped%s\n\n", green, reset)

	// Display trace.
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

	// Summary.
	fmt.Printf("%s%s── Summary ─────────────────────────────────────────%s\n", bold, blue, reset)
	fmt.Printf("  URLs scraped:   %d\n", len(urls))
	fmt.Printf("  %sAllowed:        %d%s\n", green, allowed, reset)
	fmt.Printf("  %sDenied:         %d%s\n", red, denied, reset)
	fmt.Printf("  Trace events:   %d\n\n", len(traces))
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
