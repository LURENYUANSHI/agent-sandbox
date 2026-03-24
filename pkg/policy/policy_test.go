package policy

import (
	"testing"
	"time"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name       string
		policy     *types.Policy
		action     *types.Action
		wantEffect types.Effect
	}{
		{
			name:   "default policy allows tmp read",
			policy: DefaultPolicy(),
			action: &types.Action{
				Type:   types.ActionTypeFile,
				Path:   "/tmp/test.txt",
				FileOp: types.FileOpRead,
			},
			wantEffect: types.EffectAllow,
		},
		{
			name:   "default policy denies root delete",
			policy: DefaultPolicy(),
			action: &types.Action{
				Type:   types.ActionTypeFile,
				Path:   "/etc/passwd",
				FileOp: types.FileOpDelete,
			},
			wantEffect: types.EffectDeny,
		},
		{
			name:   "strict policy denies tmp write",
			policy: StrictPolicy(),
			action: &types.Action{
				Type:   types.ActionTypeFile,
				Path:   "/tmp/test.txt",
				FileOp: types.FileOpWrite,
			},
			wantEffect: types.EffectDeny,
		},
		{
			name:   "permissive policy allows network",
			policy: PermissivePolicy(),
			action: &types.Action{
				Type: types.ActionTypeNetwork,
				Host: "example.com",
				Port: 443,
			},
			wantEffect: types.EffectAllow,
		},
		{
			name:   "permissive policy denies file delete at root",
			policy: PermissivePolicy(),
			action: &types.Action{
				Type:   types.ActionTypeFile,
				Path:   "/important",
				FileOp: types.FileOpDelete,
			},
			wantEffect: types.EffectDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(tt.policy)
			tt.action.Timestamp = time.Now()
			effect, _ := engine.Evaluate(tt.action)
			if effect != tt.wantEffect {
				t.Errorf("got %s, want %s", effect, tt.wantEffect)
			}
		})
	}
}

func TestLoadPolicy(t *testing.T) {
	engine := NewEngine(StrictPolicy())

	action := &types.Action{
		Type:      types.ActionTypeFile,
		Path:      "/tmp/test.txt",
		FileOp:    types.FileOpWrite,
		Timestamp: time.Now(),
	}

	effect, _ := engine.Evaluate(action)
	if effect != types.EffectDeny {
		t.Fatalf("strict policy should deny write, got %s", effect)
	}

	engine.LoadPolicy(PermissivePolicy())
	effect, _ = engine.Evaluate(action)
	if effect != types.EffectAllow {
		t.Fatalf("permissive policy should allow write, got %s", effect)
	}
}

func TestParsePolicy(t *testing.T) {
	yaml := []byte(`
name: test-policy
description: A test policy
default_effect: deny
rules:
  - name: allow-file-read
    effect: allow
    actions: [file]
    paths: ["/tmp/*"]
    file_ops: [read]
`)
	p, err := Parse(yaml)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if p.Name != "test-policy" {
		t.Errorf("name = %s, want test-policy", p.Name)
	}
	if len(p.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(p.Rules))
	}
	if p.Rules[0].Effect != types.EffectAllow {
		t.Errorf("rule effect = %s, want allow", p.Rules[0].Effect)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  *types.Policy
		wantErr bool
	}{
		{
			name:    "valid policy",
			policy:  DefaultPolicy(),
			wantErr: false,
		},
		{
			name:    "missing name",
			policy:  &types.Policy{DefaultEffect: types.EffectDeny},
			wantErr: true,
		},
		{
			name: "invalid effect",
			policy: &types.Policy{
				Name:          "bad",
				DefaultEffect: "maybe",
			},
			wantErr: true,
		},
		{
			name: "rule missing actions",
			policy: &types.Policy{
				Name:          "bad-rule",
				DefaultEffect: types.EffectDeny,
				Rules: []types.Rule{
					{Name: "empty", Effect: types.EffectAllow},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
