package policy

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// Engine implements types.PolicyEngine with rule-based evaluation.
type Engine struct {
	mu     sync.RWMutex
	policy *types.Policy
}

// NewEngine creates a policy engine with a deny-all default.
func NewEngine() *Engine {
	return &Engine{
		policy: &types.Policy{
			Name:          "empty",
			DefaultEffect: types.EffectDeny,
		},
	}
}

// LoadPolicy replaces the current policy.
func (e *Engine) LoadPolicy(policy types.Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = &policy
	return nil
}

// Evaluate checks an action against the loaded policy rules.
// Rules are evaluated in priority order (higher first). The first matching rule wins.
// If no rule matches, the policy's default effect applies.
func (e *Engine) Evaluate(action types.Action) types.PolicyDecision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.policy.Rules {
		if !matchActionType(rule.ActionType, action.Type) {
			continue
		}
		if !matchConditions(rule.Conditions, action) {
			continue
		}
		return types.PolicyDecision{
			Allowed: rule.Effect == types.EffectAllow,
			Rule:    rule.Name,
			Reason:  rule.Description,
		}
	}

	allowed := e.policy.DefaultEffect == types.EffectAllow
	return types.PolicyDecision{
		Allowed: allowed,
		Reason:  "default policy: " + string(e.policy.DefaultEffect),
	}
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
