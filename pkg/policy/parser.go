package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

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
