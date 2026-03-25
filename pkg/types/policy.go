package types

import "time"

// Effect represents the outcome of a policy rule evaluation.
type Effect string

const (
	// EffectAllow permits the action to proceed.
	EffectAllow Effect = "allow"
	// EffectDeny blocks the action from executing.
	EffectDeny Effect = "deny"
	// EffectAudit permits the action but logs it for review.
	EffectAudit Effect = "audit"
)

// Rule defines a single policy rule that matches actions against patterns
// and produces an effect (allow, deny, or audit).
type Rule struct {
	// ID is the unique identifier for this rule.
	ID string `json:"id" yaml:"id"`
	// Name is a human-readable name for the rule.
	Name string `json:"name" yaml:"name"`
	// Description explains what this rule does and why.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Actions contains glob patterns matching action types (e.g., "file:*", "net:http").
	Actions []string `json:"actions" yaml:"actions"`
	// Resources contains patterns matching target resources (e.g., "/tmp/**", "*.example.com").
	Resources []string `json:"resources" yaml:"resources"`
	// Effect is the outcome when this rule matches (allow, deny, or audit).
	Effect Effect `json:"effect" yaml:"effect"`
	// Priority determines evaluation order; higher values are evaluated first.
	Priority int `json:"priority" yaml:"priority"`
	// Conditions holds optional key-value conditions for fine-grained matching.
	Conditions map[string]string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Policy is a named, versioned collection of rules that govern what actions
// are permitted within a sandbox.
type Policy struct {
	// Name is the human-readable name for this policy.
	Name string `json:"name" yaml:"name"`
	// Version tracks the policy revision.
	Version string `json:"version" yaml:"version"`
	// Description explains the purpose and scope of this policy.
	Description string `json:"description" yaml:"description"`
	// DefaultEffect is applied when no rule matches an action.
	DefaultEffect Effect `json:"default_effect" yaml:"default_effect"`
	// Rules is the ordered list of rules in this policy.
	Rules []Rule `json:"rules" yaml:"rules"`
}

// PolicyDecision captures the result of evaluating an action against the policy engine.
type PolicyDecision struct {
	// Effect is the final decision (allow, deny, or audit).
	Effect Effect `json:"effect"`
	// Rule is the rule that produced this decision, if any.
	Rule *Rule `json:"rule,omitempty"`
	// Reason explains why the decision was made.
	Reason string `json:"reason"`
	// Timestamp records when the decision was made.
	Timestamp time.Time `json:"timestamp"`
}
