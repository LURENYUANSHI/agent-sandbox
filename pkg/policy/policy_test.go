package policy

import (
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

func TestEngine_DefaultDeny(t *testing.T) {
	e := NewEngine()
	action := types.Action{Type: types.ActionTypeFileRead, Params: map[string]string{"path": "/tmp/x"}}
	d := e.Evaluate(action)
	if d.Allowed {
		t.Error("expected deny by default")
	}
}

func TestEngine_AllowRule(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{Name: "allow-reads", ActionType: types.ActionTypeFileRead, Effect: types.EffectAllow},
		},
	})

	d := e.Evaluate(types.Action{Type: types.ActionTypeFileRead})
	if !d.Allowed {
		t.Error("expected allow for file.read")
	}

	d = e.Evaluate(types.Action{Type: types.ActionTypeFileWrite})
	if d.Allowed {
		t.Error("expected deny for file.write")
	}
}

func TestEngine_WildcardMatch(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "wildcard",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{Name: "allow-all-file", ActionType: "file.*", Effect: types.EffectAllow},
		},
	})

	tests := []struct {
		actionType types.ActionType
		want       bool
	}{
		{types.ActionTypeFileRead, true},
		{types.ActionTypeFileWrite, true},
		{types.ActionTypeFileDelete, true},
		{types.ActionTypeNetHTTP, false},
		{types.ActionTypeProcess, false},
	}

	for _, tt := range tests {
		d := e.Evaluate(types.Action{Type: tt.actionType})
		if d.Allowed != tt.want {
			t.Errorf("action %s: got allowed=%t, want %t", tt.actionType, d.Allowed, tt.want)
		}
	}
}

func TestEngine_ConditionMatch(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "conditional",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:       "allow-tmp-reads",
				ActionType: types.ActionTypeFileRead,
				Effect:     types.EffectAllow,
				Conditions: map[string]string{"path": "/tmp/*"},
			},
		},
	})

	d := e.Evaluate(types.Action{
		Type:   types.ActionTypeFileRead,
		Params: map[string]string{"path": "/tmp/foo"},
	})
	if !d.Allowed {
		t.Error("expected allow for /tmp/foo")
	}

	d = e.Evaluate(types.Action{
		Type:   types.ActionTypeFileRead,
		Params: map[string]string{"path": "/etc/passwd"},
	})
	if d.Allowed {
		t.Error("expected deny for /etc/passwd")
	}
}

func TestEngine_DefaultAllow(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "permissive",
		DefaultEffect: types.EffectAllow,
	})

	d := e.Evaluate(types.Action{Type: types.ActionTypeProcess})
	if !d.Allowed {
		t.Error("expected allow with default_effect=allow")
	}
}

func TestParse(t *testing.T) {
	yaml := []byte(`
name: test-policy
description: A test policy
default_effect: allow
rules:
  - name: deny-deletes
    action_type: file.delete
    effect: deny
`)
	p, err := Parse(yaml)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Name != "test-policy" {
		t.Errorf("name = %q", p.Name)
	}
	if p.DefaultEffect != types.EffectAllow {
		t.Errorf("default_effect = %q", p.DefaultEffect)
	}
	if len(p.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(p.Rules))
	}
	if p.Rules[0].Effect != types.EffectDeny {
		t.Errorf("rule effect = %q", p.Rules[0].Effect)
	}
}
