package types

import "context"

// Effect represents the result of a policy evaluation.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Rule represents a single policy rule.
type Rule struct {
	ID        string   `yaml:"id" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	Actions   []string `yaml:"actions" json:"actions"`
	Resources []string `yaml:"resources" json:"resources"`
	Effect    Effect   `yaml:"effect" json:"effect"`
	Priority  int      `yaml:"priority" json:"priority"`
}

// Policy represents a complete security policy.
type Policy struct {
	Name          string `yaml:"name" json:"name"`
	Version       string `yaml:"version" json:"version"`
	Description   string `yaml:"description" json:"description"`
	DefaultEffect Effect `yaml:"default_effect" json:"default_effect"`
	Rules         []Rule `yaml:"rules" json:"rules"`
}

// PolicyDecision represents the result of evaluating an action against a policy.
type PolicyDecision struct {
	Effect Effect `json:"effect"`
	Rule   *Rule  `json:"rule,omitempty"`
	Reason string `json:"reason"`
}

// PolicyEngine defines the interface for policy evaluation.
type PolicyEngine interface {
	Evaluate(ctx context.Context, action Action) PolicyDecision
	LoadPolicy(policy Policy) error
	GetPolicy() Policy
}
