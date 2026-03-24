package types

// Effect represents whether a policy rule allows or denies an action.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Rule represents a single policy rule.
type Rule struct {
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description"`
	Effect      Effect     `json:"effect" yaml:"effect"`
	ActionType  ActionType `json:"action_type" yaml:"action_type"`
	Conditions  Conditions `json:"conditions,omitempty" yaml:"conditions"`
	Priority    int        `json:"priority,omitempty" yaml:"priority"`
}

// Conditions defines the matching criteria for a rule.
type Conditions struct {
	Paths      []string `json:"paths,omitempty" yaml:"paths"`
	Hosts      []string `json:"hosts,omitempty" yaml:"hosts"`
	Ports      []int    `json:"ports,omitempty" yaml:"ports"`
	Commands   []string `json:"commands,omitempty" yaml:"commands"`
	Operations []string `json:"operations,omitempty" yaml:"operations"`
}

// Policy is a collection of rules that govern sandbox behavior.
type Policy struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description"`
	Version     string `json:"version,omitempty" yaml:"version"`
	Rules       []Rule `json:"rules" yaml:"rules"`
}
