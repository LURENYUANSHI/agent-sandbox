package executor

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

const maxResponseSize = 5 * 1024 * 1024 // 5 MB

// NetworkExecutor handles network operations.
type NetworkExecutor struct {
	enabled bool
	client  *http.Client
}

// NewNetworkExecutor creates a network executor.
func NewNetworkExecutor(enabled bool) *NetworkExecutor {
	return &NetworkExecutor{
		enabled: enabled,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteNetHTTP performs an HTTP request.
func (n *NetworkExecutor) ExecuteNetHTTP(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	if !n.enabled {
		return nil, fmt.Errorf("network access is disabled for this sandbox")
	}

	url := action.Params["url"]
	method := action.Params["method"]
	if url == "" {
		return nil, fmt.Errorf("net.http requires 'url' parameter")
	}
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return &types.ActionResult{
		ActionID:  action.ID,
		Success:   resp.StatusCode >= 200 && resp.StatusCode < 400,
		Output:    string(body),
		ExitCode:  resp.StatusCode,
		BytesRead: int64(len(body)),
	}, nil
}

// ExecuteNetConnect checks TCP connectivity to a host:port.
func (n *NetworkExecutor) ExecuteNetConnect(ctx context.Context, action types.Action) (*types.ActionResult, error) {
	if !n.enabled {
		return nil, fmt.Errorf("network access is disabled for this sandbox")
	}

	host := action.Params["host"]
	port := action.Params["port"]
	if host == "" || port == "" {
		return nil, fmt.Errorf("net.connect requires 'host' and 'port' parameters")
	}

	addr := net.JoinHostPort(host, port)
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return &types.ActionResult{
			ActionID: action.ID,
			Success:  false,
			Error:    err.Error(),
			Output:   fmt.Sprintf("connection to %s failed", addr),
		}, nil
	}
	conn.Close()

	return &types.ActionResult{
		ActionID: action.ID,
		Success:  true,
		Output:   fmt.Sprintf("connected to %s", addr),
	}, nil
}
