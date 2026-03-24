package types

// Effect determines whether a rule allows or denies an action.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Rule defines a single policy rule that matches actions.
type Rule struct {
	Name    string       `json:"name" yaml:"name"`
	Effect  Effect       `json:"effect" yaml:"effect"`
	Actions []ActionType `json:"actions" yaml:"actions"`
	Paths   []string     `json:"paths,omitempty" yaml:"paths"`
	FileOps []FileOp     `json:"file_ops,omitempty" yaml:"file_ops"`
	Hosts   []string     `json:"hosts,omitempty" yaml:"hosts"`
}

// Policy is a named set of rules with a default effect.
type Policy struct {
	Name          string `json:"name" yaml:"name"`
	Description   string `json:"description" yaml:"description"`
	DefaultEffect Effect `json:"default_effect" yaml:"default_effect"`
	Rules         []Rule `json:"rules" yaml:"rules"`
}
