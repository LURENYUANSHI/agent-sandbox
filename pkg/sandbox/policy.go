package sandbox

import "github.com/LURENYUANSHI/agent-sandbox/pkg/types"

// ReloadPolicy hot-reloads the sandbox policy.
func (s *Sandbox) ReloadPolicy(p *types.Policy) {
	s.executor.ReloadPolicy(p)
}
