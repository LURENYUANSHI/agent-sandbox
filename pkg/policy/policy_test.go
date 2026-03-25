package policy

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/config"
	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ---------------------------------------------------------------------------
// Engine: basic evaluation
// ---------------------------------------------------------------------------

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

	d := e.Evaluate(types.Action{
		Type:   types.ActionTypeFileRead,
		Params: map[string]string{},
	})
	if d.Allowed {
		t.Error("expected deny when condition key is missing")
	}
}

// ---------------------------------------------------------------------------
// Engine: priority ordering
// ---------------------------------------------------------------------------

func TestEngine_PriorityOrdering(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "priority-test",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:     "low-priority-allow",
				Actions:  []string{"file:*"},
				Effect:   types.EffectAllow,
				Priority: 10,
			},
			{
				Name:      "high-priority-deny",
				Actions:   []string{"file:*"},
				Resources: []string{"/secret/*"},
				Effect:    types.EffectDeny,
				Priority:  100,
			},
		},
	})

	// High-priority deny should win for /secret/ paths
	d := e.Evaluate(types.Action{Type: types.ActionFileRead, Resource: "/secret/key"})
	if d.Allowed {
		t.Error("expected deny from high-priority rule")
	}

	// Low-priority allow should apply for other paths
	d = e.Evaluate(types.Action{Type: types.ActionFileRead, Resource: "/tmp/file"})
	if !d.Allowed {
		t.Error("expected allow from low-priority rule")
	}
}

func TestEngine_PriorityReverseOrder(t *testing.T) {
	// Rules inserted in reverse priority order should still sort correctly.
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "reverse",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{Name: "low-deny", Actions: []string{"shell:*"}, Effect: types.EffectDeny, Priority: 1},
			{Name: "high-allow", Actions: []string{"shell:exec"}, Effect: types.EffectAllow, Priority: 100},
		},
	})

	d := e.Evaluate(types.Action{Type: types.ActionShellExec, Resource: "ls"})
	if !d.Allowed {
		t.Error("expected high-priority allow to win")
	}
}

// ---------------------------------------------------------------------------
// Engine: Actions/Resources glob matching
// ---------------------------------------------------------------------------

func TestEngine_ActionsResourcesMatch(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "glob-actions",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				Name:      "allow-file-read-tmp",
				Actions:   []string{"file:read"},
				Resources: []string{"/tmp/**"},
				Effect:    types.EffectAllow,
			},
		},
	})

	// Match
	d := e.Evaluate(types.Action{Type: types.ActionFileRead, Resource: "/tmp/deep/file.txt"})
	if !d.Allowed {
		t.Error("expected allow for /tmp/deep/file.txt")
	}

	// Wrong action type
	d = e.Evaluate(types.Action{Type: types.ActionFileWrite, Resource: "/tmp/file"})
	if d.Allowed {
		t.Error("expected deny for file:write")
	}

	// Wrong resource
	d = e.Evaluate(types.Action{Type: types.ActionFileRead, Resource: "/etc/passwd"})
	if d.Allowed {
		t.Error("expected deny for /etc/passwd")
	}
}

func TestEngine_ActionsWithoutResources(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "no-resources",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{Name: "allow-net", Actions: []string{"net:*"}, Effect: types.EffectAllow},
		},
	})

	d := e.Evaluate(types.Action{Type: types.ActionNetConnect, Resource: "example.com"})
	if !d.Allowed {
		t.Error("expected allow when rule has Actions but no Resources")
	}
}

// ---------------------------------------------------------------------------
// Engine: empty rules, default effect fallback
// ---------------------------------------------------------------------------

func TestEngine_EmptyRulesDefaultDeny(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{Name: "empty", DefaultEffect: types.EffectDeny})

	d := e.Evaluate(types.Action{Type: types.ActionTypeFileRead})
	if d.Allowed {
		t.Error("expected deny with empty rules and default deny")
	}
	if d.Reason != "default policy: deny" {
		t.Errorf("reason = %q", d.Reason)
	}
}

func TestEngine_EmptyRulesDefaultAllow(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{Name: "empty-allow", DefaultEffect: types.EffectAllow})

	d := e.Evaluate(types.Action{Type: types.ActionTypeProcess})
	if !d.Allowed {
		t.Error("expected allow with empty rules and default allow")
	}
}

func TestEngine_EmptyDefaultEffectFallsToDeny(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{Name: "no-default"})

	d := e.Evaluate(types.Action{Type: types.ActionTypeFileRead})
	if d.Allowed {
		t.Error("expected deny when default_effect is empty")
	}
}

// ---------------------------------------------------------------------------
// Engine: constructors and GetPolicy
// ---------------------------------------------------------------------------

func TestNewPolicyEngine(t *testing.T) {
	p := types.Policy{
		Name:          "custom",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{Name: "r1", ActionType: types.ActionTypeFileRead, Effect: types.EffectDeny},
		},
	}
	e := NewPolicyEngine(p)
	got := e.GetPolicy()
	if got.Name != "custom" {
		t.Errorf("policy name = %q", got.Name)
	}
	if got.DefaultEffect != types.EffectAllow {
		t.Errorf("default effect = %q", got.DefaultEffect)
	}
}

func TestNewEngineWithConfig(t *testing.T) {
	cfg := config.PolicyConfig{MaxFileSizeMB: 50, PrivilegedPortLimit: 512}
	e := NewEngineWithConfig(cfg)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	// Verify the engine works
	d := e.Evaluate(types.Action{Type: types.ActionTypeFileRead})
	if d.Allowed {
		t.Error("expected deny from default engine")
	}
}

func TestEngine_GetPolicy(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{Name: "my-policy", DefaultEffect: types.EffectAllow})
	p := e.GetPolicy()
	if p.Name != "my-policy" {
		t.Errorf("got name %q", p.Name)
	}
}

// ---------------------------------------------------------------------------
// Engine: concurrent evaluation
// ---------------------------------------------------------------------------

func TestEngine_ConcurrentEvaluate(t *testing.T) {
	e := NewEngine()
	e.LoadPolicy(types.Policy{
		Name:          "concurrent",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{Name: "allow-reads", ActionType: types.ActionTypeFileRead, Effect: types.EffectAllow},
		},
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := e.Evaluate(types.Action{Type: types.ActionTypeFileRead})
			if !d.Allowed {
				t.Error("expected allow")
			}
		}()
	}
	wg.Wait()
}

func TestEngine_ConcurrentLoadAndEvaluate(t *testing.T) {
	e := NewEngine()

	var wg sync.WaitGroup
	// Concurrent loads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.LoadPolicy(types.Policy{
				Name:          "p",
				DefaultEffect: types.EffectAllow,
			})
		}()
	}
	// Concurrent evaluations
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Evaluate(types.Action{Type: types.ActionTypeFileRead})
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// MatchGlob edge cases
// ---------------------------------------------------------------------------

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		str     string
		pattern string
		want    bool
	}{
		// Exact match
		{"hello", "hello", true},
		{"hello", "world", false},
		// Empty
		{"", "", true},
		{"hello", "", false},
		{"", "hello", false},
		// Single star
		{"file.txt", "*.txt", true},
		{"file.txt", "*.go", false},
		{"path/file.txt", "*.txt", false}, // * doesn't match /
		{"abc", "*", true},
		{"", "*", true},
		// Double star
		{"a/b/c", "**", true},
		{"a/b/c", "a/**/c", true},
		{"a/b/d/c", "a/**/c", true},
		{"a/c", "a/**/c", true},
		// Question mark
		{"abc", "a?c", true},
		{"ac", "a?c", false},
		{"a/c", "a?c", false}, // ? doesn't match /
		// Combined
		{"file.read", "file.*", true},
		{"file.write", "file.*", true},
		{"net.http", "file.*", false},
		// Path patterns
		{"/tmp/foo/bar", "/tmp/**", true},
		{"/tmp/foo", "/tmp/*", true},
		{"/tmp/foo/bar", "/tmp/*", false},
		// Multiple wildcards
		{"a.b.c", "*.*.*", true},
		{"a.b", "*.*.*", false},
	}
	for _, tt := range tests {
		got := MatchGlob(tt.str, tt.pattern)
		if got != tt.want {
			t.Errorf("MatchGlob(%q, %q) = %t, want %t", tt.str, tt.pattern, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// matchActionType
// ---------------------------------------------------------------------------

func TestMatchActionType(t *testing.T) {
	tests := []struct {
		ruleType   types.ActionType
		actionType types.ActionType
		want       bool
	}{
		{"*", "anything", true},
		{"file.*", "file.read", true},
		{"file.*", "file.write", true},
		{"file.*", "net.http", false},
		{"file.read", "file.read", true},
		{"file.read", "file.write", false},
		{"net.*", "net.http", true},
		{"net.*", "network.tcp", false},
	}
	for _, tt := range tests {
		got := matchActionType(tt.ruleType, tt.actionType)
		if got != tt.want {
			t.Errorf("matchActionType(%q, %q) = %t, want %t", tt.ruleType, tt.actionType, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// matchConditions
// ---------------------------------------------------------------------------

func TestMatchConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions map[string]string
		action     types.Action
		want       bool
	}{
		{"nil conditions", nil, types.Action{Params: map[string]string{"a": "b"}}, true},
		{"empty conditions", map[string]string{}, types.Action{}, true},
		{"match", map[string]string{"path": "/tmp/*"}, types.Action{Params: map[string]string{"path": "/tmp/x"}}, true},
		{"no match", map[string]string{"path": "/tmp/*"}, types.Action{Params: map[string]string{"path": "/etc/x"}}, false},
		{"missing key", map[string]string{"path": "/tmp/*"}, types.Action{Params: map[string]string{}}, false},
		{"nil params", map[string]string{"path": "*"}, types.Action{}, false},
		{"bad pattern", map[string]string{"path": "[invalid"}, types.Action{Params: map[string]string{"path": "x"}}, false},
	}
	for _, tt := range tests {
		got := matchConditions(tt.conditions, tt.action)
		if got != tt.want {
			t.Errorf("%s: matchConditions = %t, want %t", tt.name, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Built-in rules (BuiltinRule type from rules.go)
// ---------------------------------------------------------------------------

func TestBuiltinRule_NoDeleteRoot(t *testing.T) {
	rule := noDeleteRoot()
	tests := []struct {
		action  types.Action
		blocked bool
	}{
		{types.Action{Type: types.ActionFileDelete, Resource: "/"}, true},
		{types.Action{Type: types.ActionFileDelete, Resource: "/etc"}, true},
		{types.Action{Type: types.ActionFileDelete, Resource: "/usr"}, true},
		{types.Action{Type: types.ActionFileDelete, Resource: "/boot"}, true},
		{types.Action{Type: types.ActionFileDelete, Resource: "/tmp/safe"}, false},
		{types.Action{Type: types.ActionFileRead, Resource: "/"}, false},   // wrong action type
		{types.Action{Type: types.ActionFileWrite, Resource: "/"}, false},  // wrong action type
	}
	for _, tt := range tests {
		decision, matched := rule.Check(tt.action)
		if matched != tt.blocked {
			t.Errorf("noDeleteRoot(%s, %s): matched=%t, want %t", tt.action.Type, tt.action.Resource, matched, tt.blocked)
		}
		if matched && decision.Allowed {
			t.Error("blocked action should not be allowed")
		}
	}
}

func TestBuiltinRule_NoKillInit(t *testing.T) {
	rule := noKillInit()
	tests := []struct {
		action  types.Action
		blocked bool
	}{
		{types.Action{Type: types.ActionProcKill, Resource: "1"}, true},
		{types.Action{Type: types.ActionProcKill, Resource: " 1 "}, true}, // whitespace
		{types.Action{Type: types.ActionProcKill, Resource: "2"}, false},
		{types.Action{Type: types.ActionProcExec, Resource: "1"}, false}, // wrong type
	}
	for _, tt := range tests {
		_, matched := rule.Check(tt.action)
		if matched != tt.blocked {
			t.Errorf("noKillInit(%s, %q): matched=%t, want %t", tt.action.Type, tt.action.Resource, matched, tt.blocked)
		}
	}
}

func TestBuiltinRule_NoDangerousCommands(t *testing.T) {
	rule := noDangerousCommands()
	tests := []struct {
		action  types.Action
		blocked bool
	}{
		{types.Action{Type: types.ActionShellExec, Resource: "mkfs /dev/sda"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: "dd if=/dev/zero of=/dev/sda"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: "dd if=/dev/random of=file"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: ":(){ :|:& };:"}, true},
		{types.Action{Type: types.ActionProcExec, Resource: "mkfs /dev/sda"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: "rm -rf /"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: "rm -rf /etc"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: "rm -rf /*"}, true},
		{types.Action{Type: types.ActionShellExec, Resource: "ls -la"}, false},
		{types.Action{Type: types.ActionShellExec, Resource: "echo hello"}, false},
		{types.Action{Type: types.ActionShellExec, Resource: "rm file.txt"}, false},
		{types.Action{Type: types.ActionFileRead, Resource: "mkfs"}, false}, // wrong type
	}
	for _, tt := range tests {
		_, matched := rule.Check(tt.action)
		if matched != tt.blocked {
			t.Errorf("noDangerousCommands(%s, %q): matched=%t, want %t", tt.action.Type, tt.action.Resource, matched, tt.blocked)
		}
	}
}

func TestBuiltinRule_NoPrivilegedPorts(t *testing.T) {
	rule := noPrivilegedPorts(1024)
	tests := []struct {
		action  types.Action
		blocked bool
	}{
		{types.Action{Type: types.ActionNetListen, Resource: "80"}, true},
		{types.Action{Type: types.ActionNetListen, Resource: "443"}, true},
		{types.Action{Type: types.ActionNetListen, Resource: "1"}, true},
		{types.Action{Type: types.ActionNetListen, Resource: "1024"}, false},
		{types.Action{Type: types.ActionNetListen, Resource: "8080"}, false},
		{types.Action{Type: types.ActionNetListen, Resource: "0.0.0.0:80"}, true},
		{types.Action{Type: types.ActionNetListen, Resource: "0.0.0.0:8080"}, false},
		{types.Action{Type: types.ActionNetListen, Resource: "invalid"}, false},
		{types.Action{Type: types.ActionNetConnect, Resource: "80"}, false}, // wrong type
	}
	for _, tt := range tests {
		_, matched := rule.Check(tt.action)
		if matched != tt.blocked {
			t.Errorf("noPrivilegedPorts(%s, %q): matched=%t, want %t", tt.action.Type, tt.action.Resource, matched, tt.blocked)
		}
	}
}

func TestBuiltinRule_MaxFileSizeLimit(t *testing.T) {
	maxSize := int64(100 * 1024 * 1024) // 100MB
	rule := maxFileSizeLimit(maxSize)

	tests := []struct {
		action  types.Action
		blocked bool
	}{
		// Exceeds limit
		{types.Action{Type: types.ActionFileWrite, Metadata: map[string]interface{}{"size": int64(200 * 1024 * 1024)}}, true},
		// Within limit
		{types.Action{Type: types.ActionFileWrite, Metadata: map[string]interface{}{"size": int64(50 * 1024 * 1024)}}, false},
		// No size metadata
		{types.Action{Type: types.ActionFileWrite, Metadata: map[string]interface{}{}}, false},
		// Nil metadata
		{types.Action{Type: types.ActionFileWrite}, false},
		// Wrong action type
		{types.Action{Type: types.ActionFileRead, Metadata: map[string]interface{}{"size": int64(200 * 1024 * 1024)}}, false},
		// int type
		{types.Action{Type: types.ActionFileWrite, Metadata: map[string]interface{}{"size": int(200 * 1024 * 1024)}}, true},
		// float64 type
		{types.Action{Type: types.ActionFileWrite, Metadata: map[string]interface{}{"size": float64(200 * 1024 * 1024)}}, true},
		// string type (unsupported)
		{types.Action{Type: types.ActionFileWrite, Metadata: map[string]interface{}{"size": "big"}}, false},
	}
	for _, tt := range tests {
		_, matched := rule.Check(tt.action)
		if matched != tt.blocked {
			t.Errorf("maxFileSizeLimit(type=%s, meta=%v): matched=%t, want %t",
				tt.action.Type, tt.action.Metadata, matched, tt.blocked)
		}
	}
}

func TestBuiltinRule_PathTraversalProtection(t *testing.T) {
	rule := pathTraversalProtection()
	tests := []struct {
		resource string
		blocked  bool
	}{
		{"../etc/passwd", true},
		{"foo/../bar", true},
		{"foo/..\\bar", true},
		{"..", true},
		{"foo/..", true},
		{"foo/bar", false},
		{"/absolute/path", false},
		{"relative/path", false},
		{"..hidden", false}, // not a traversal
		{"", false},
	}
	for _, tt := range tests {
		action := types.Action{Type: types.ActionFileRead, Resource: tt.resource}
		_, matched := rule.Check(action)
		if matched != tt.blocked {
			t.Errorf("pathTraversalProtection(%q): matched=%t, want %t", tt.resource, matched, tt.blocked)
		}
	}
}

func TestDefaultBuiltinRules(t *testing.T) {
	rules := DefaultBuiltinRules()
	if len(rules) != 6 {
		t.Fatalf("expected 6 default builtin rules, got %d", len(rules))
	}
}

func TestBuiltinRulesFromConfig(t *testing.T) {
	cfg := config.PolicyConfig{MaxFileSizeMB: 50, PrivilegedPortLimit: 512}
	rules := BuiltinRulesFromConfig(cfg)
	if len(rules) != 6 {
		t.Fatalf("expected 6 rules, got %d", len(rules))
	}

	// Verify custom port limit works: port 600 should be fine with limit 512
	// (no, 600 >= 512, so not blocked) but port 100 should be blocked
	for _, r := range rules {
		if r.ID == "no-privileged-ports" {
			_, blocked := r.Check(types.Action{Type: types.ActionNetListen, Resource: "100"})
			if !blocked {
				t.Error("expected port 100 to be blocked with limit 512")
			}
			_, blocked = r.Check(types.Action{Type: types.ActionNetListen, Resource: "600"})
			if blocked {
				t.Error("expected port 600 to be allowed with limit 512")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Engine: built-in rules enforcement via BuiltInRules() (types.Rule style)
// ---------------------------------------------------------------------------

func TestBuiltInRules(t *testing.T) {
	rules := BuiltInRules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 built-in rules, got %d", len(rules))
	}

	if rules[0].Name != "deny-delete-root" {
		t.Errorf("rule[0].Name = %q", rules[0].Name)
	}
	if rules[0].Effect != types.EffectDeny {
		t.Errorf("rule[0].Effect = %q", rules[0].Effect)
	}
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

	d := e.Evaluate(types.Action{
		Type:   types.ActionTypeFileDelete,
		Params: map[string]string{"path": "/"},
	})
	if d.Allowed {
		t.Error("expected deny for delete /")
	}

	d = e.Evaluate(types.Action{
		Type:   types.ActionTypeFileWrite,
		Params: map[string]string{"path": "/etc/passwd"},
	})
	if d.Allowed {
		t.Error("expected deny for write /etc/passwd")
	}
}

// ---------------------------------------------------------------------------
// isDestructiveRm
// ---------------------------------------------------------------------------

func TestIsDestructiveRm(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"rm -rf /", true},
		{"rm -rf /*", true},
		{"rm -rf /etc", true},
		{"rm -rf /home", true},
		{"rm -rf /tmp/safe", false},
		{"rm -r /", false},        // no force
		{"rm -f /", false},        // no recursive
		{"rm file.txt", false},    // no flags
		{"ls -la", false},         // not rm
		{"", false},               // empty
	}
	for _, tt := range tests {
		got := isDestructiveRm(tt.cmd)
		if got != tt.want {
			t.Errorf("isDestructiveRm(%q) = %t, want %t", tt.cmd, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// parsePort
// ---------------------------------------------------------------------------

func TestParsePort(t *testing.T) {
	tests := []struct {
		input string
		port  int
		ok    bool
	}{
		{"80", 80, true},
		{"8080", 8080, true},
		{"0.0.0.0:443", 443, true},
		{"localhost:3000", 3000, true},
		{"invalid", 0, false},
		{"host:notaport", 0, false},
	}
	for _, tt := range tests {
		port, ok := parsePort(tt.input)
		if ok != tt.ok || (ok && port != tt.port) {
			t.Errorf("parsePort(%q) = (%d, %t), want (%d, %t)", tt.input, port, ok, tt.port, tt.ok)
		}
	}
}

// ---------------------------------------------------------------------------
// Parse and ParseFile
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// LoadFromFile / LoadFromBytes / Validate
// ---------------------------------------------------------------------------

func TestLoadFromBytes_Valid(t *testing.T) {
	data := []byte(`
name: validated
default_effect: deny
rules:
  - id: r1
    actions: ["file:*"]
    effect: allow
`)
	p, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	if p.Name != "validated" {
		t.Errorf("name = %q", p.Name)
	}
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{{{bad`))
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestLoadFromBytes_ValidationError_MissingName(t *testing.T) {
	data := []byte(`
default_effect: deny
rules:
  - id: r1
    actions: ["file:*"]
    effect: allow
`)
	_, err := LoadFromBytes(data)
	if err == nil {
		t.Error("expected validation error for missing name")
	}
}

func TestLoadFromFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.yaml")
	os.WriteFile(path, []byte(`
name: from-file
default_effect: allow
rules:
  - id: r1
    actions: ["net:*"]
    effect: deny
`), 0o644)

	p, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p.Name != "from-file" {
		t.Errorf("name = %q", p.Name)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/policy.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidate_MissingName(t *testing.T) {
	p := &types.Policy{Rules: []types.Rule{{ID: "r1", Actions: []string{"*"}, Effect: types.EffectAllow}}}
	if err := Validate(p); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidate_InvalidDefaultEffect(t *testing.T) {
	p := &types.Policy{Name: "test", DefaultEffect: "invalid"}
	if err := Validate(p); err == nil {
		t.Error("expected error for invalid default effect")
	}
}

func TestValidate_DuplicateRuleIDs(t *testing.T) {
	p := &types.Policy{
		Name: "dup",
		Rules: []types.Rule{
			{ID: "r1", Actions: []string{"*"}, Effect: types.EffectAllow},
			{ID: "r1", Actions: []string{"*"}, Effect: types.EffectDeny},
		},
	}
	if err := Validate(p); err == nil {
		t.Error("expected error for duplicate rule IDs")
	}
}

func TestValidate_MissingRuleID(t *testing.T) {
	p := &types.Policy{
		Name:  "no-id",
		Rules: []types.Rule{{Actions: []string{"*"}, Effect: types.EffectAllow}},
	}
	if err := Validate(p); err == nil {
		t.Error("expected error for missing rule ID")
	}
}

func TestValidate_InvalidRuleEffect(t *testing.T) {
	p := &types.Policy{
		Name:  "bad-effect",
		Rules: []types.Rule{{ID: "r1", Actions: []string{"*"}, Effect: "maybe"}},
	}
	if err := Validate(p); err == nil {
		t.Error("expected error for invalid rule effect")
	}
}

func TestValidate_NoActions(t *testing.T) {
	p := &types.Policy{
		Name:  "no-actions",
		Rules: []types.Rule{{ID: "r1", Effect: types.EffectAllow}},
	}
	if err := Validate(p); err == nil {
		t.Error("expected error for rule with no actions")
	}
}

func TestValidate_ValidPolicy(t *testing.T) {
	p := &types.Policy{
		Name:          "valid",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{ID: "r1", Actions: []string{"file:*"}, Effect: types.EffectAllow},
			{ID: "r2", Actions: []string{"net:*"}, Effect: types.EffectDeny},
		},
	}
	if err := Validate(p); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyRules(t *testing.T) {
	p := &types.Policy{Name: "empty-rules"}
	if err := Validate(p); err != nil {
		t.Errorf("expected no error for empty rules: %v", err)
	}
}

// ---------------------------------------------------------------------------
// metadataInt64
// ---------------------------------------------------------------------------

func TestMetadataInt64(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want int64
		ok   bool
	}{
		{"int64 value", map[string]interface{}{"size": int64(42)}, "size", 42, true},
		{"int value", map[string]interface{}{"size": 42}, "size", 42, true},
		{"float64 value", map[string]interface{}{"size": 42.5}, "size", 42, true},
		{"string value", map[string]interface{}{"size": "big"}, "size", 0, false},
		{"missing key", map[string]interface{}{}, "size", 0, false},
		{"nil map", nil, "size", 0, false},
	}
	for _, tt := range tests {
		got, ok := metadataInt64(tt.m, tt.key)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("%s: got (%d, %t), want (%d, %t)", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

// ---------------------------------------------------------------------------
// matchesRule (combining Actions/Resources and ActionType strategies)
// ---------------------------------------------------------------------------

func TestMatchesRule(t *testing.T) {
	tests := []struct {
		name   string
		action types.Action
		rule   types.Rule
		want   bool
	}{
		{
			"Actions match, no resources",
			types.Action{Type: types.ActionFileRead},
			types.Rule{Actions: []string{"file:read"}, Effect: types.EffectAllow},
			true,
		},
		{
			"Actions match, resource match",
			types.Action{Type: types.ActionFileRead, Resource: "/tmp/x"},
			types.Rule{Actions: []string{"file:*"}, Resources: []string{"/tmp/*"}, Effect: types.EffectAllow},
			true,
		},
		{
			"Actions match, resource mismatch",
			types.Action{Type: types.ActionFileRead, Resource: "/etc/x"},
			types.Rule{Actions: []string{"file:*"}, Resources: []string{"/tmp/*"}, Effect: types.EffectAllow},
			false,
		},
		{
			"ActionType match",
			types.Action{Type: types.ActionTypeFileRead},
			types.Rule{ActionType: types.ActionTypeFileRead, Effect: types.EffectAllow},
			true,
		},
		{
			"No actions or actionType",
			types.Action{Type: types.ActionTypeFileRead},
			types.Rule{Effect: types.EffectAllow},
			false,
		},
	}
	for _, tt := range tests {
		got := matchesRule(tt.action, tt.rule)
		if got != tt.want {
			t.Errorf("%s: matchesRule = %t, want %t", tt.name, got, tt.want)
		}
	}
}
