package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/executor"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/trace"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

var version = "0.1.0"

// sandboxRegistry is a simple in-process registry for CLI usage.
// In a real deployment, the API server manages sandbox state.
var sandboxRegistry = struct {
	entries map[string]*sandboxEntry
}{entries: make(map[string]*sandboxEntry)}

type sandboxEntry struct {
	instance *sandbox.Instance
	config   sandbox.Config
	exec     *executor.Executor
	recorder *trace.Recorder
	engine   *policy.Engine
}

func main() {
	root := &cobra.Command{
		Use:   "agent-sandbox",
		Short: "AI Agent Security Sandbox",
		Long:  "Runtime sandbox for AI agents with policy enforcement, trace recording, and replay.",
	}

	root.PersistentFlags().String("output", "text", "Output format: text or json")

	root.AddCommand(
		createCmd(),
		startCmd(),
		execCmd(),
		stopCmd(),
		listCmd(),
		traceCmd(),
		replayCmd(),
		policyCmd(),
		versionCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func outputJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func isJSON(cmd *cobra.Command) bool {
	f, _ := cmd.Flags().GetString("output")
	return f == "json"
}

// --- create ---

func createCmd() *cobra.Command {
	var name, policyFile, rootDir string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new sandbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := uuid.New().String()
			if name == "" {
				name = "sandbox-" + id[:8]
			}
			if rootDir == "" {
				rootDir = filepath.Join(os.TempDir(), "agent-sandbox", id)
			}

			cfg := sandbox.DefaultConfig()
			cfg.ID = id
			cfg.Name = name
			cfg.RootDir = rootDir
			cfg.TracePath = filepath.Join(rootDir, "traces.db")

			engine := policy.NewEngine()
			if policyFile != "" {
				p, err := policy.ParseFile(policyFile)
				if err != nil {
					return fmt.Errorf("load policy: %w", err)
				}
				if err := engine.LoadPolicy(*p); err != nil {
					return fmt.Errorf("apply policy: %w", err)
				}
			}

			recorder, err := trace.NewRecorder("")
			if err != nil {
				return fmt.Errorf("create recorder: %w", err)
			}

			instance, err := sandbox.NewSandbox(cfg, engine, recorder)
			if err != nil {
				return fmt.Errorf("create sandbox: %w", err)
			}

			exec := executor.NewExecutor(cfg)

			sandboxRegistry.entries[id] = &sandboxEntry{
				instance: instance,
				config:   cfg,
				exec:     exec,
				recorder: recorder,
				engine:   engine,
			}

			if isJSON(cmd) {
				outputJSON(map[string]string{
					"id":       id,
					"name":     name,
					"status":   string(instance.Status()),
					"root_dir": rootDir,
				})
			} else {
				fmt.Printf("Created sandbox %s (%s)\n", name, id)
				fmt.Printf("  Root: %s\n", rootDir)
				fmt.Printf("  Status: %s\n", instance.Status())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Sandbox name")
	cmd.Flags().StringVar(&policyFile, "policy", "", "Policy YAML file")
	cmd.Flags().StringVar(&rootDir, "root", "", "Sandbox root directory")
	return cmd
}

// --- start ---

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <sandbox-id>",
		Short: "Start a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := getEntry(args[0])
			if err != nil {
				return err
			}
			if err := entry.instance.Start(context.Background()); err != nil {
				return err
			}
			if isJSON(cmd) {
				outputJSON(map[string]string{
					"id":     args[0],
					"status": string(entry.instance.Status()),
				})
			} else {
				fmt.Printf("Sandbox %s started\n", args[0])
			}
			return nil
		},
	}
}

// --- exec ---

func execCmd() *cobra.Command {
	var params []string

	cmd := &cobra.Command{
		Use:   "exec <sandbox-id> <action-type>",
		Short: "Execute an action in a sandbox",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := getEntry(args[0])
			if err != nil {
				return err
			}

			paramMap := make(map[string]string)
			for _, p := range params {
				parts := strings.SplitN(p, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid param format %q, expected key=value", p)
				}
				paramMap[parts[0]] = parts[1]
			}

			action := types.Action{
				ID:        uuid.New().String(),
				Type:      types.ActionType(args[1]),
				Params:    paramMap,
				Timestamp: time.Now(),
			}

			result, err := entry.instance.Execute(context.Background(), action, entry.exec.Execute)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				outputJSON(result)
			} else {
				fmt.Printf("Action: %s\n", args[1])
				fmt.Printf("Success: %t\n", result.Success)
				if result.Output != "" {
					fmt.Printf("Output:\n%s\n", result.Output)
				}
				if result.Error != "" {
					fmt.Printf("Error: %s\n", result.Error)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&params, "param", nil, "Action parameters (key=value)")
	return cmd
}

// --- stop ---

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <sandbox-id>",
		Short: "Stop a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := getEntry(args[0])
			if err != nil {
				return err
			}
			if err := entry.instance.Stop(context.Background()); err != nil {
				return err
			}
			if isJSON(cmd) {
				outputJSON(map[string]string{
					"id":     args[0],
					"status": string(entry.instance.Status()),
				})
			} else {
				fmt.Printf("Sandbox %s stopped\n", args[0])
			}
			return nil
		},
	}
}

// --- list ---

func listCmd() *cobra.Command {
	var statusFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			type item struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			}
			var items []item
			for id, entry := range sandboxRegistry.entries {
				s := string(entry.instance.Status())
				if statusFilter != "" && s != statusFilter {
					continue
				}
				items = append(items, item{ID: id, Name: entry.config.Name, Status: s})
			}

			if isJSON(cmd) {
				outputJSON(items)
				return nil
			}

			if len(items) == 0 {
				fmt.Println("No sandboxes found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS")
			for _, it := range items {
				fmt.Fprintf(w, "%s\t%s\t%s\n", it.ID, it.Name, it.Status)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status")
	return cmd
}

// --- trace ---

func traceCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "trace <sandbox-id>",
		Short: "Show trace events for a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := getEntry(args[0])
			if err != nil {
				return err
			}

			events, err := entry.instance.GetTraces()
			if err != nil {
				return fmt.Errorf("get traces: %w", err)
			}

			switch format {
			case "json":
				outputJSON(events)
			case "table":
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tTYPE\tACTION_ID\tTIMESTAMP\tDURATION")
				for _, e := range events {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						e.ID, e.Type, e.ActionID,
						e.Timestamp.Format(time.RFC3339),
						e.Duration)
				}
				w.Flush()
			default:
				// Default to table
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tTYPE\tACTION_ID\tTIMESTAMP\tDURATION")
				for _, e := range events {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						e.ID, e.Type, e.ActionID,
						e.Timestamp.Format(time.RFC3339),
						e.Duration)
				}
				w.Flush()
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: json, table, or otel")
	return cmd
}

// --- replay ---

func replayCmd() *cobra.Command {
	var step bool

	cmd := &cobra.Command{
		Use:   "replay <sandbox-id>",
		Short: "Replay trace events for a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := getEntry(args[0])
			if err != nil {
				return err
			}

			events, err := entry.instance.GetTraces()
			if err != nil {
				return fmt.Errorf("get traces: %w", err)
			}

			if len(events) == 0 {
				fmt.Println("No trace events to replay")
				return nil
			}

			fmt.Printf("Replaying %d events for sandbox %s\n", len(events), args[0])

			for i, event := range events {
				fmt.Printf("[%d/%d] %s  action=%s\n", i+1, len(events), event.Type, event.ActionID)
				if event.Data != nil {
					for k, v := range event.Data {
						fmt.Printf("       %s: %s\n", k, v)
					}
				}
				if step && i < len(events)-1 {
					fmt.Print("Press Enter to continue...")
					fmt.Scanln()
				}
			}
			fmt.Println("Replay complete")
			return nil
		},
	}
	cmd.Flags().BoolVar(&step, "step", false, "Step through events one at a time")
	return cmd
}

// --- policy validate ---

func policyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy management commands",
	}

	validate := &cobra.Command{
		Use:   "validate <policy-file>",
		Short: "Validate a policy YAML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := policy.ParseFile(args[0])
			if err != nil {
				if isJSON(cmd) {
					outputJSON(map[string]interface{}{
						"valid":  false,
						"errors": []string{err.Error()},
					})
				} else {
					fmt.Printf("Invalid policy: %s\n", err)
				}
				return nil
			}

			if isJSON(cmd) {
				outputJSON(map[string]interface{}{
					"valid":  true,
					"policy": p,
				})
			} else {
				fmt.Printf("Valid policy: %s\n", p.Name)
				fmt.Printf("  Description: %s\n", p.Description)
				fmt.Printf("  Rules: %d\n", len(p.Rules))
				fmt.Printf("  Default: %s\n", p.DefaultEffect)
			}
			return nil
		},
	}
	cmd.AddCommand(validate)
	return cmd
}

// --- version ---

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if isJSON(cmd) {
				outputJSON(map[string]string{"version": version})
			} else {
				fmt.Printf("agent-sandbox version %s\n", version)
			}
		},
	}
}

// --- helpers ---

func getEntry(id string) (*sandboxEntry, error) {
	entry, ok := sandboxRegistry.entries[id]
	if !ok {
		return nil, fmt.Errorf("sandbox %q not found", id)
	}
	return entry, nil
}
