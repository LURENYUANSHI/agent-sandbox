package executor

import (
	"fmt"
	"os/exec"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func executeProcess(action *types.Action) (string, error) {
	if action.Command == "" {
		return "", fmt.Errorf("command is required for process/shell actions")
	}

	cmd := exec.Command(action.Command, action.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("executing command %s: %w", action.Command, err)
	}
	return string(output), nil
}
