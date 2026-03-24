package policy

import "github.com/LURENYUANSHI/agent-sandbox/pkg/types"

// BuiltInRules returns a set of safety rules that should always be enforced.
func BuiltInRules() []types.Rule {
	return []types.Rule{
		{
			Name:        "deny-delete-root",
			Description: "prevent deletion of root or system directories",
			ActionType:  types.ActionTypeFileDelete,
			Effect:      types.EffectDeny,
			Conditions:  map[string]string{"path": "/"},
			Priority:    1000,
		},
		{
			Name:        "deny-write-etc",
			Description: "prevent writing to /etc",
			ActionType:  types.ActionTypeFileWrite,
			Effect:      types.EffectDeny,
			Conditions:  map[string]string{"path": "/etc/*"},
			Priority:    1000,
		},
	}
}
