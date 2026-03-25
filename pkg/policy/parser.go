package policy

import (
	"errors"
	"fmt"
	"os"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
	"gopkg.in/yaml.v3"
)

// LoadFromFile reads and parses a YAML policy file with validation.
func LoadFromFile(path string) (*types.Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file %s: %w", path, err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes parses YAML data into a Policy and validates it.
func LoadFromBytes(data []byte) (*types.Policy, error) {
	var policy types.Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parsing policy YAML: %w", err)
	}
	if err := Validate(&policy); err != nil {
		return nil, fmt.Errorf("validating policy: %w", err)
	}
	return &policy, nil
}

// Validate checks a policy for structural correctness.
func Validate(policy *types.Policy) error {
	if policy.Name == "" {
		return errors.New("policy name is required")
	}

	if policy.DefaultEffect != "" &&
		policy.DefaultEffect != types.EffectAllow &&
		policy.DefaultEffect != types.EffectDeny {
		return fmt.Errorf("invalid default effect: %q", policy.DefaultEffect)
	}

	seenIDs := make(map[string]bool, len(policy.Rules))
	for _, rule := range policy.Rules {
		if rule.ID == "" {
			return errors.New("rule ID is required")
		}
		if seenIDs[rule.ID] {
			return fmt.Errorf("duplicate rule ID: %q", rule.ID)
		}
		seenIDs[rule.ID] = true

		if rule.Effect != types.EffectAllow && rule.Effect != types.EffectDeny {
			return fmt.Errorf("rule %q: invalid effect %q", rule.ID, rule.Effect)
		}
		if len(rule.Actions) == 0 {
			return fmt.Errorf("rule %q: must have at least one action pattern", rule.ID)
		}
	}
	return nil
}

// ParseFile reads a YAML policy file and returns a Policy.
func ParseFile(path string) (*types.Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes YAML bytes into a Policy.
func Parse(data []byte) (*types.Policy, error) {
	var p types.Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse policy YAML: %w", err)
	}
	if p.DefaultEffect == "" {
		p.DefaultEffect = types.EffectDeny
	}
	return &p, nil
}
