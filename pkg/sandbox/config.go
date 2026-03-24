package sandbox

import "github.com/LURENYUANSHI/agent-sandbox/pkg/types"

// Config holds the configuration for creating a sandbox.
type Config struct {
	ID       string        `json:"id"`
	BasePath string        `json:"base_path"`
	Policy   *types.Policy `json:"policy"`
}
