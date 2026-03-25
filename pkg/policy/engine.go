package policy

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Engine implements the types.PolicyEngine interface with built-in safety rules,
// priority-based evaluation, and glob pattern matching.
type Engine struct {
	mu           sync.RWMutex
	policy       *types.Policy
	builtinRules []BuiltinRule
}

// NewPolicyEngine creates a new policy engine with built-in rules and the given default policy.
func NewPolicyEngine(defaultPolicy types.Policy) *Engine {
	return &Engine{
		policy:       &defaultPolicy,
		builtinRules: DefaultBuiltinRules(),
	}
}

// NewEngine creates a policy engine with a deny-all default and built-in rules.
func NewEngine() *Engine {
	return &Engine{
		policy: &types.Policy{
			Name:          "empty",
			DefaultEffect: types.EffectDeny,
		},
		builtinRules: DefaultBuiltinRules(),
	}
}

// NewEngineWithConfig creates a policy engine using the provided PolicyConfig.
func NewEngineWithConfig(cfg config.PolicyConfig) *Engine {
	return &Engine{
		policy: &types.Policy{
			Name:          "empty",
			DefaultEffect: types.EffectDeny,
		},
		builtinRules: BuiltinRulesFromConfig(cfg),
	}
}

// Evaluate checks an action against built-in rules first, then user policy rules.
// Implements the types.PolicyEngine interface.
func (e *Engine) Evaluate(action types.Action) types.PolicyDecision {
	// Built-in rules are always checked first and cannot be overridden.
	for _, rule := range e.builtinRules {
		if decision, matched := rule.Check(action); matched {
			return decision
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Try glob-based matching (phase3 rules with Actions/Resources patterns).
	rules := make([]types.Rule, len(e.policy.Rules))
	copy(rules, e.policy.Rules)
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

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
		Effect:  effect,
		Allowed: effect == types.EffectAllow,
		Reason:  "default policy: " + string(effect),
	}
}

// LoadPolicy replaces the current policy.
func (e *Engine) LoadPolicy(policy types.Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = &policy
	return nil
}

// GetPolicy returns the current policy.
func (e *Engine) GetPolicy() types.Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return *e.policy
}

// matchesRule checks if an action matches a rule using both glob-based and
// ActionType-based matching strategies.
func matchesRule(action types.Action, rule types.Rule) bool {
	// Strategy 1: Glob-based matching on Actions/Resources patterns.
	if len(rule.Actions) > 0 {
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

	// Strategy 2: ActionType-based matching with conditions (phase5 style).
	if rule.ActionType != "" {
		if !matchActionType(rule.ActionType, action.Type) {
			return false
		}
		return matchConditions(rule.Conditions, action)
	}

	return false
}

// matchActionType checks if a rule's action type matches the action.
// Supports wildcard prefix matching: "file.*" matches "file.read", "file.write", etc.
func matchActionType(ruleType, actionType types.ActionType) bool {
	rt := string(ruleType)
	at := string(actionType)
	if rt == "*" {
		return true
	}
	if strings.HasSuffix(rt, ".*") {
		prefix := strings.TrimSuffix(rt, ".*")
		return strings.HasPrefix(at, prefix+".")
	}
	return rt == at
}

// matchConditions checks if all rule conditions match the action parameters.
func matchConditions(conditions map[string]string, action types.Action) bool {
	for key, pattern := range conditions {
		val, ok := action.Params[key]
		if !ok {
			return false
		}
		matched, err := filepath.Match(pattern, val)
		if err != nil || !matched {
			return false
		}
	}
	return true
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

		if pattern[0] == '?' {
			if str[0] == '/' {
				return false
			}
			str = str[1:]
			pattern = pattern[1:]
			continue
		}

		if pattern[0] != str[0] {
			return false
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0
}
