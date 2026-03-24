package policy

import (
	"fmt"
	"os"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
	"gopkg.in/yaml.v3"
)

// ParseFile loads a policy from a YAML file.
func ParseFile(path string) (*types.Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes YAML bytes into a Policy.
func Parse(data []byte) (*types.Policy, error) {
	var p types.Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing policy YAML: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("policy must have a name")
	}
	if p.DefaultEffect == "" {
		p.DefaultEffect = types.EffectDeny
	}
	return &p, nil
}

// Validate checks a policy for structural correctness.
func Validate(p *types.Policy) error {
	if p.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if p.DefaultEffect != types.EffectAllow && p.DefaultEffect != types.EffectDeny {
		return fmt.Errorf("invalid default_effect: %s", p.DefaultEffect)
	}
	for i, rule := range p.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d: name is required", i)
		}
		if rule.Effect != types.EffectAllow && rule.Effect != types.EffectDeny {
			return fmt.Errorf("rule %s: invalid effect: %s", rule.Name, rule.Effect)
		}
		if len(rule.Actions) == 0 {
			return fmt.Errorf("rule %s: at least one action type is required", rule.Name)
		}
	}
	return nil
}
