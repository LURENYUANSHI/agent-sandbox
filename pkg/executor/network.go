package executor

import (
	"fmt"
	"net"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func executeNetwork(action *types.Action) (string, error) {
	addr := net.JoinHostPort(action.Host, fmt.Sprintf("%d", action.Port))

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("connecting to %s: %w", addr, err)
	}
	conn.Close()

	return fmt.Sprintf("connected to %s", addr), nil
}
