package policy

import (
	"path/filepath"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Engine evaluates actions against a loaded policy.
type Engine struct {
	policy *types.Policy
	mu     sync.RWMutex
}

// NewEngine creates a policy engine with the given policy.
func NewEngine(policy *types.Policy) *Engine {
	return &Engine{policy: policy}
}

// Evaluate checks an action against loaded rules and returns the effect and reason.
func (e *Engine) Evaluate(action *types.Action) (types.Effect, string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.policy.Rules {
		if matchRule(&rule, action) {
			return rule.Effect, rule.Name
		}
	}
	return e.policy.DefaultEffect, "default policy"
}

// LoadPolicy replaces the current policy (hot-reload).
func (e *Engine) LoadPolicy(policy *types.Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = policy
}

// GetPolicy returns a copy of the current policy.
func (e *Engine) GetPolicy() *types.Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p := *e.policy
	return &p
}

func matchRule(rule *types.Rule, action *types.Action) bool {
	if !matchActionType(rule.Actions, action.Type) {
		return false
	}
	if len(rule.Paths) > 0 && action.Path != "" {
		if !matchPath(rule.Paths, action.Path) {
			return false
		}
	}
	if len(rule.FileOps) > 0 && action.FileOp != "" {
		if !matchFileOp(rule.FileOps, action.FileOp) {
			return false
		}
	}
	if len(rule.Hosts) > 0 && action.Host != "" {
		if !matchHost(rule.Hosts, action.Host) {
			return false
		}
	}
	return true
}

func matchActionType(allowed []types.ActionType, actual types.ActionType) bool {
	for _, a := range allowed {
		if a == actual {
			return true
		}
	}
	return false
}

func matchPath(patterns []string, path string) bool {
	// Normalize to OS path separators for consistent matching
	path = filepath.FromSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.FromSlash(pattern)
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Check if path starts with a directory pattern (e.g., "/tmp/*" matches "/tmp/foo/bar")
		if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			dir := pattern[:len(pattern)-1]
			if len(path) >= len(dir) && path[:len(dir)] == dir {
				return true
			}
		}
	}
	return false
}

func matchFileOp(allowed []types.FileOp, actual types.FileOp) bool {
	for _, op := range allowed {
		if op == actual {
			return true
		}
	}
	return false
}

func matchHost(allowed []string, actual string) bool {
	for _, h := range allowed {
		if h == actual || h == "*" {
			return true
		}
	}
	return false
}
