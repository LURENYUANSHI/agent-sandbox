package types

// Effect represents the outcome of a policy rule evaluation.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
	EffectAudit Effect = "audit"
)

// Rule defines a single policy rule that matches actions against patterns
// and produces an effect (allow, deny, or audit).
type Rule struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Actions     []string          `json:"actions,omitempty" yaml:"actions,omitempty"`
	Resources   []string          `json:"resources,omitempty" yaml:"resources,omitempty"`
	Effect      Effect            `json:"effect" yaml:"effect"`
	Priority    int               `json:"priority,omitempty" yaml:"priority,omitempty"`
	Conditions  map[string]string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	ActionType  ActionType        `json:"action_type,omitempty" yaml:"action_type,omitempty"`
}

// Policy is a named, versioned collection of rules that govern what actions
// are permitted within a sandbox.
type Policy struct {
	Name          string `json:"name" yaml:"name"`
	Version       string `json:"version,omitempty" yaml:"version,omitempty"`
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	DefaultEffect Effect `json:"default_effect" yaml:"default_effect"`
	Rules         []Rule `json:"rules" yaml:"rules"`
}
