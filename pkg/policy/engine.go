package policy

import (
	"context"
	"sort"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Engine implements the types.PolicyEngine interface.
type Engine struct {
	mu           sync.RWMutex
	policy       types.Policy
	builtinRules []BuiltinRule
}

// NewPolicyEngine creates a new policy engine with the given default policy.
func NewPolicyEngine(defaultPolicy types.Policy) *Engine {
	return &Engine{
		policy:       defaultPolicy,
		builtinRules: DefaultBuiltinRules(),
	}
}

// Evaluate checks an action against built-in rules first, then user policy rules by priority.
func (e *Engine) Evaluate(_ context.Context, action types.Action) types.PolicyDecision {
	// Built-in rules are always checked first and cannot be overridden.
	for _, rule := range e.builtinRules {
		if decision, matched := rule.Check(action); matched {
			return decision
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Sort rules by priority descending (higher priority evaluated first).
	rules := make([]types.Rule, len(e.policy.Rules))
	copy(rules, e.policy.Rules)
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	// Return the first matching rule's effect.
	for i := range rules {
		if matchesRule(action, rules[i]) {
			r := rules[i]
			return types.PolicyDecision{
				Effect:  r.Effect,
				Allowed: r.Effect == types.EffectAllow,
				Rule:    r.Name,
				Reason:  r.Name,
			}
		}
	}

	// No rule matched — apply default effect.
	effect := e.policy.DefaultEffect
	if effect == "" {
		effect = types.EffectDeny
	}
	return types.PolicyDecision{
		Effect: effect,
		Reason: "default policy effect",
	}
}

// LoadPolicy replaces the current policy.
func (e *Engine) LoadPolicy(policy types.Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = policy
	return nil
}

// GetPolicy returns the current policy.
func (e *Engine) GetPolicy() types.Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.policy
}

// matchesRule checks if an action matches a rule's action patterns and resource patterns.
func matchesRule(action types.Action, rule types.Rule) bool {
	actionMatched := false
	for _, pattern := range rule.Actions {
		if MatchGlob(string(action.Type), pattern) {
			actionMatched = true
			break
		}
	}
	if !actionMatched {
		return false
	}

	// If no resources specified, match any resource.
	if len(rule.Resources) == 0 {
		return true
	}

	for _, pattern := range rule.Resources {
		if MatchGlob(action.Resource, pattern) {
			return true
		}
	}
	return false
}

// MatchGlob matches a string against a glob pattern.
//
// Supported syntax:
//   - *  matches any sequence of characters except /
//   - ** matches any sequence of characters including /
//   - ?  matches any single character except /
func MatchGlob(str, pattern string) bool {
	if pattern == "*" || pattern == "**" {
		return true
	}
	return doMatch(str, pattern)
}

func doMatch(str, pattern string) bool {
	for len(pattern) > 0 {
		// Double-star: matches everything including path separators.
		if len(pattern) >= 2 && pattern[0] == '*' && pattern[1] == '*' {
			rest := pattern[2:]
			if len(rest) > 0 && rest[0] == '/' {
				rest = rest[1:]
			}
			for i := 0; i <= len(str); i++ {
				if doMatch(str[i:], rest) {
					return true
				}
			}
			return false
		}

		// Single star: matches any sequence of non-/ characters.
		if pattern[0] == '*' {
			rest := pattern[1:]
			for i := 0; i <= len(str); i++ {
				if i > 0 && str[i-1] == '/' {
					break
				}
				if doMatch(str[i:], rest) {
					return true
				}
			}
			return false
		}

		if len(str) == 0 {
			return false
		}

		// Question mark: matches any single non-/ character.
		if pattern[0] == '?' {
			if str[0] == '/' {
				return false
			}
			str = str[1:]
			pattern = pattern[1:]
			continue
		}

		// Literal character match.
		if pattern[0] != str[0] {
			return false
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0
}
