package main

import (
	"fmt"
	"os"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/policy"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/sandbox"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: agent-sandbox <command>")
		fmt.Println("Commands: create, start, exec, stop, list, version")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("agent-sandbox v0.1.0")
	case "create":
		id := "default"
		if len(os.Args) > 2 {
			id = os.Args[2]
		}
		sb, err := sandbox.New(&sandbox.Config{
			ID:     id,
			Policy: policy.DefaultPolicy(),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Sandbox %s created (state: %s)\n", sb.ID, sb.State)
	case "demo":
		sb, _ := sandbox.New(&sandbox.Config{
			ID:     "demo",
			Policy: policy.DefaultPolicy(),
		})
		sb.Start()
		event, _ := sb.Execute(&types.Action{
			Type:   types.ActionTypeFile,
			Path:   "/tmp/demo.txt",
			FileOp: types.FileOpRead,
		})
		fmt.Printf("Action: %s %s -> %s (reason: %s)\n",
			event.Action.FileOp, event.Action.Path, event.Decision, event.Reason)
		sb.Stop()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
