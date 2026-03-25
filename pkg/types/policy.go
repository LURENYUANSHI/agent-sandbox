package types

// Effect is the result of a policy decision.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Rule defines a single policy rule matching an action type.
type Rule struct {
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description"`
	ActionType  ActionType `json:"action_type" yaml:"action_type"`
	Effect      Effect     `json:"effect" yaml:"effect"`
	Conditions  map[string]string `json:"conditions,omitempty" yaml:"conditions"`
	Priority    int        `json:"priority,omitempty" yaml:"priority"`
}

// Policy is a named collection of rules.
type Policy struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description"`
	Rules       []Rule `json:"rules" yaml:"rules"`
	DefaultEffect Effect `json:"default_effect" yaml:"default_effect"`
}

// PolicyDecision captures the result of evaluating an action against a policy.
type PolicyDecision struct {
	Allowed    bool   `json:"allowed"`
	Rule       string `json:"rule,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// PolicyEngine evaluates actions against loaded policies.
type PolicyEngine interface {
	Evaluate(action Action) PolicyDecision
	LoadPolicy(policy Policy) error
}

// TraceRecorder records trace events during sandbox execution.
type TraceRecorder interface {
	Record(event TraceEvent) error
	GetEvents(sandboxID string) ([]TraceEvent, error)
	Close() error
}
