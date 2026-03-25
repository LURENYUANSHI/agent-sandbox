package policy

import (
	"os"
	"path/filepath"
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

func TestParse_DefaultEffect(t *testing.T) {
	// When default_effect is omitted, should default to deny
	data := []byte(`
name: no-default
rules: []
`)
	p, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.DefaultEffect != types.EffectDeny {
		t.Errorf("expected default_effect=deny, got %q", p.DefaultEffect)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	data := []byte(`{{{invalid yaml`)
	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := []byte(`
name: file-policy
default_effect: allow
rules:
  - name: deny-net
    action_type: net.*
    effect: deny
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	p, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if p.Name != "file-policy" {
		t.Errorf("name = %q", p.Name)
	}
	if p.DefaultEffect != types.EffectAllow {
		t.Errorf("default_effect = %q", p.DefaultEffect)
	}
	if len(p.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(p.Rules))
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/policy.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestBuiltInRules(t *testing.T) {
	rules := BuiltInRules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 built-in rules, got %d", len(rules))
	}

	// Verify deny-delete-root
	if rules[0].Name != "deny-delete-root" {
		t.Errorf("rule[0].Name = %q", rules[0].Name)
	}
	if rules[0].Effect != types.EffectDeny {
		t.Errorf("rule[0].Effect = %q", rules[0].Effect)
	}

	// Verify deny-write-etc
	if rules[1].Name != "deny-write-etc" {
		t.Errorf("rule[1].Name = %q", rules[1].Name)
	}
}

func TestEngine_BuiltInRulesEnforcement(t *testing.T) {
	e := NewEngine()
	rules := BuiltInRules()
	e.LoadPolicy(types.Policy{
		Name:          "with-builtins",
		DefaultEffect: types.EffectAllow,
		Rules:         rules,
	})

	// Delete root should be denied
	d := e.Evaluate(types.Action{
		Type:   types.ActionTypeFileDelete,
		Params: map[string]string{"path": "/"},
	})
	if d.Allowed {
		t.Error("expected deny for delete /")
	}

	// Write to /etc should be denied
	d = e.Evaluate(types.Action{
		Type:   types.ActionTypeFileWrite,
		Params: map[string]string{"path": "/etc/passwd"},
	})
	if d.Allowed {
		t.Error("expected deny for write /etc/passwd")
	}
}

func TestEngine_GlobalWildcard(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "allow-all",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{Name: "allow-everything", ActionType: "*", Effect: types.EffectAllow},
		},
	})

	d := e.Evaluate(types.Action{Type: types.ActionTypeProcess})
	if !d.Allowed {
		t.Error("expected allow for * wildcard")
	}
}

func TestEngine_ConditionMissingKey(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "cond",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:       "needs-path",
				ActionType: types.ActionTypeFileRead,
				Effect:     types.EffectAllow,
				Conditions: map[string]string{"path": "/allowed/*"},
			},
		},
	})

	// Action without the "path" param should not match the rule
	d := e.Evaluate(types.Action{
		Type:   types.ActionTypeFileRead,
		Params: map[string]string{},
	})
	if d.Allowed {
		t.Error("expected deny when condition key is missing")
	}
}
