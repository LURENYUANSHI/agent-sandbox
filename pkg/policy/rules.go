package policy

import "github.com/LURENYUANSHI/agent-sandbox/pkg/types"

// DefaultPolicy returns a restrictive default policy.
func DefaultPolicy() *types.Policy {
	return &types.Policy{
		Name:          "default",
		Description:   "Default restrictive security policy",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:    "allow-tmp-read",
				Effect:  types.EffectAllow,
				Actions: []types.ActionType{types.ActionTypeFile},
				Paths:   []string{"/tmp/*"},
				FileOps: []types.FileOp{types.FileOpRead, types.FileOpList},
			},
			{
				Name:    "allow-tmp-write",
				Effect:  types.EffectAllow,
				Actions: []types.ActionType{types.ActionTypeFile},
				Paths:   []string{"/tmp/*"},
				FileOps: []types.FileOp{types.FileOpWrite},
			},
		},
	}
}

// StrictPolicy returns a strict policy that denies almost everything.
func StrictPolicy() *types.Policy {
	return &types.Policy{
		Name:          "strict",
		Description:   "Strict policy that denies most operations",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:    "allow-tmp-read-only",
				Effect:  types.EffectAllow,
				Actions: []types.ActionType{types.ActionTypeFile},
				Paths:   []string{"/tmp/*"},
				FileOps: []types.FileOp{types.FileOpRead},
			},
		},
	}
}

// PermissivePolicy returns a policy that allows most operations.
func PermissivePolicy() *types.Policy {
	return &types.Policy{
		Name:          "permissive",
		Description:   "Permissive policy for trusted agents",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{
				Name:    "deny-root-delete",
				Effect:  types.EffectDeny,
				Actions: []types.ActionType{types.ActionTypeFile},
				Paths:   []string{"/*"},
				FileOps: []types.FileOp{types.FileOpDelete},
			},
		},
	}
}
