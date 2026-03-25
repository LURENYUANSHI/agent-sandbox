package policy

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func denyAllPolicy() types.Policy {
	return types.Policy{
		Name:          "deny-all",
		Version:       "1.0",
		DefaultEffect: types.EffectDeny,
	}
}

func allowAllPolicy() types.Policy {
	return types.Policy{
		Name:          "allow-all",
		Version:       "1.0",
		DefaultEffect: types.EffectAllow,
	}
}

// ---------------------------------------------------------------------------
// Engine — basic allow / deny
// ---------------------------------------------------------------------------

func TestEvaluate_BasicAllowDeny(t *testing.T) {
	tests := []struct {
		name   string
		policy types.Policy
		action types.Action
		want   types.Effect
	}{
		{
			name:   "deny by default when no rules match",
			policy: denyAllPolicy(),
			action: types.Action{Type: types.ActionFileRead, Resource: "/secret"},
			want:   types.EffectDeny,
		},
		{
			name:   "allow by default when no rules match",
			policy: allowAllPolicy(),
			action: types.Action{Type: types.ActionFileRead, Resource: "/anything"},
			want:   types.EffectAllow,
		},
		{
			name: "matching allow rule",
			policy: types.Policy{
				Name:          "test",
				DefaultEffect: types.EffectDeny,
				Rules: []types.Rule{{
					ID: "r1", Name: "allow tmp", Actions: []string{"file:read"},
					Resources: []string{"/tmp/**"}, Effect: types.EffectAllow, Priority: 10,
				}},
			},
			action: types.Action{Type: types.ActionFileRead, Resource: "/tmp/data.txt"},
			want:   types.EffectAllow,
		},
		{
			name: "matching deny rule",
			policy: types.Policy{
				Name:          "test",
				DefaultEffect: types.EffectAllow,
				Rules: []types.Rule{{
					ID: "r1", Name: "deny shell", Actions: []string{"shell:exec"},
					Resources: []string{"*"}, Effect: types.EffectDeny, Priority: 10,
				}},
			},
			action: types.Action{Type: types.ActionShellExec, Resource: "bash"},
			want:   types.EffectDeny,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewPolicyEngine(tt.policy)
			got := e.Evaluate(ctx, tt.action)
			if got.Effect != tt.want {
				t.Errorf("got effect %q, want %q (reason: %s)", got.Effect, tt.want, got.Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Glob matching — action patterns
// ---------------------------------------------------------------------------

func TestEvaluate_ActionGlob(t *testing.T) {
	policy := types.Policy{
		Name:          "glob-test",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{{
			ID: "r1", Name: "allow all file ops", Actions: []string{"file:*"},
			Resources: []string{"**"}, Effect: types.EffectAllow, Priority: 10,
		}},
	}

	tests := []struct {
		action types.ActionType
		want   types.Effect
	}{
		{types.ActionFileRead, types.EffectAllow},
		{types.ActionFileWrite, types.EffectAllow},
		{types.ActionFileDelete, types.EffectAllow},
		{types.ActionNetHTTP, types.EffectDeny},   // no match
		{types.ActionShellExec, types.EffectDeny},  // no match
	}

	ctx := context.Background()
	e := NewPolicyEngine(policy)
	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := e.Evaluate(ctx, types.Action{Type: tt.action, Resource: "/tmp/x"})
			if got.Effect != tt.want {
				t.Errorf("action %s: got %q, want %q", tt.action, got.Effect, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Glob matching — resource patterns
// ---------------------------------------------------------------------------

func TestEvaluate_ResourceGlob(t *testing.T) {
	policy := types.Policy{
		Name:          "resource-glob",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				ID: "r1", Name: "allow tmp", Actions: []string{"file:read"},
				Resources: []string{"/tmp/**"}, Effect: types.EffectAllow, Priority: 10,
			},
			{
				ID: "r2", Name: "allow github", Actions: []string{"net:http"},
				Resources: []string{"*.github.com"}, Effect: types.EffectAllow, Priority: 10,
			},
		},
	}

	tests := []struct {
		name     string
		action   types.Action
		want     types.Effect
	}{
		{"tmp file", types.Action{Type: types.ActionFileRead, Resource: "/tmp/foo.txt"}, types.EffectAllow},
		{"nested tmp", types.Action{Type: types.ActionFileRead, Resource: "/tmp/a/b/c"}, types.EffectAllow},
		{"outside tmp", types.Action{Type: types.ActionFileRead, Resource: "/etc/passwd"}, types.EffectDeny},
		{"github domain", types.Action{Type: types.ActionNetHTTP, Resource: "api.github.com"}, types.EffectAllow},
		{"non-github", types.Action{Type: types.ActionNetHTTP, Resource: "evil.com"}, types.EffectDeny},
	}

	ctx := context.Background()
	e := NewPolicyEngine(policy)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.Evaluate(ctx, tt.action)
			if got.Effect != tt.want {
				t.Errorf("got %q, want %q (reason: %s)", got.Effect, tt.want, got.Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Priority ordering
// ---------------------------------------------------------------------------

func TestEvaluate_PriorityOrdering(t *testing.T) {
	policy := types.Policy{
		Name:          "priority-test",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{
			{
				ID: "low", Name: "low-priority allow", Actions: []string{"file:read"},
				Resources: []string{"/data/**"}, Effect: types.EffectAllow, Priority: 10,
			},
			{
				ID: "high", Name: "high-priority deny", Actions: []string{"file:read"},
				Resources: []string{"/data/secret/**"}, Effect: types.EffectDeny, Priority: 100,
			},
		},
	}

	ctx := context.Background()
	e := NewPolicyEngine(policy)

	// /data/public should be allowed (only low-priority rule matches).
	got := e.Evaluate(ctx, types.Action{Type: types.ActionFileRead, Resource: "/data/public/a"})
	if got.Effect != types.EffectAllow {
		t.Errorf("/data/public: got %q, want allow", got.Effect)
	}

	// /data/secret should be denied (high-priority rule matches first).
	got = e.Evaluate(ctx, types.Action{Type: types.ActionFileRead, Resource: "/data/secret/key"})
	if got.Effect != types.EffectDeny {
		t.Errorf("/data/secret: got %q, want deny", got.Effect)
	}
}

// ---------------------------------------------------------------------------
// Built-in rules
// ---------------------------------------------------------------------------

func TestBuiltinRules(t *testing.T) {
	tests := []struct {
		name   string
		action types.Action
		denied bool
	}{
		// NoDeleteRoot
		{"delete root", types.Action{Type: types.ActionFileDelete, Resource: "/"}, true},
		{"delete /etc", types.Action{Type: types.ActionFileDelete, Resource: "/etc"}, true},
		{"delete /etc/", types.Action{Type: types.ActionFileDelete, Resource: "/etc/"}, true},
		{"delete /tmp/file ok", types.Action{Type: types.ActionFileDelete, Resource: "/tmp/file"}, false},

		// NoKillInit
		{"kill PID 1", types.Action{Type: types.ActionProcKill, Resource: "1"}, true},
		{"kill PID 42", types.Action{Type: types.ActionProcKill, Resource: "42"}, false},

		// NoDangerousCommands
		{"rm -rf /", types.Action{Type: types.ActionShellExec, Resource: "rm -rf /"}, true},
		{"rm -rf /*", types.Action{Type: types.ActionShellExec, Resource: "rm -rf /*"}, true},
		{"rm -rf /tmp ok", types.Action{Type: types.ActionShellExec, Resource: "rm -rf /tmp/stuff"}, false},
		{"mkfs", types.Action{Type: types.ActionShellExec, Resource: "mkfs /dev/sda"}, true},
		{"dd zero", types.Action{Type: types.ActionShellExec, Resource: "dd if=/dev/zero of=/dev/sda"}, true},
		{"safe echo", types.Action{Type: types.ActionShellExec, Resource: "echo hello"}, false},

		// NoPrivilegedPorts
		{"listen port 80", types.Action{Type: types.ActionNetListen, Resource: ":80"}, true},
		{"listen port 443", types.Action{Type: types.ActionNetListen, Resource: "0.0.0.0:443"}, true},
		{"listen port 8080", types.Action{Type: types.ActionNetListen, Resource: ":8080"}, false},
		{"listen port 3000", types.Action{Type: types.ActionNetListen, Resource: "3000"}, false},

		// MaxFileSizeLimit (default 100MB)
		{"write large file", types.Action{
			Type: types.ActionFileWrite, Resource: "/tmp/big",
			Metadata: map[string]interface{}{"size": int64(200 * 1024 * 1024)},
		}, true},
		{"write small file", types.Action{
			Type: types.ActionFileWrite, Resource: "/tmp/small",
			Metadata: map[string]interface{}{"size": int64(1024)},
		}, false},
		{"write no size metadata", types.Action{
			Type: types.ActionFileWrite, Resource: "/tmp/nosize",
		}, false},

		// PathTraversalProtection
		{"path traversal ../", types.Action{Type: types.ActionFileRead, Resource: "/tmp/../../etc/passwd"}, true},
		{"path traversal backslash", types.Action{Type: types.ActionFileRead, Resource: "..\\windows\\system32"}, true},
		{"path bare ..", types.Action{Type: types.ActionFileRead, Resource: ".."}, true},
		{"path trailing /..", types.Action{Type: types.ActionFileRead, Resource: "/foo/.."}, true},
		{"normal path", types.Action{Type: types.ActionFileRead, Resource: "/tmp/safe"}, false},
	}

	ctx := context.Background()
	// Use an allow-all user policy so only built-in rules can deny.
	e := NewPolicyEngine(allowAllPolicy())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.Evaluate(ctx, tt.action)
			if tt.denied && got.Effect != types.EffectDeny {
				t.Errorf("expected deny, got %q (reason: %s)", got.Effect, got.Reason)
			}
			if !tt.denied && got.Effect == types.EffectDeny {
				t.Errorf("expected allow, got deny (reason: %s)", got.Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Built-in rules cannot be overridden by user policy
// ---------------------------------------------------------------------------

func TestBuiltinRulesCannotBeOverridden(t *testing.T) {
	// Policy explicitly allows deleting root and listening on port 80.
	policy := types.Policy{
		Name:          "override-attempt",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{
				ID: "allow-delete-root", Name: "allow delete root",
				Actions: []string{"file:delete"}, Resources: []string{"/"}, Effect: types.EffectAllow, Priority: 9999,
			},
			{
				ID: "allow-port-80", Name: "allow port 80",
				Actions: []string{"net:listen"}, Resources: []string{":80"}, Effect: types.EffectAllow, Priority: 9999,
			},
			{
				ID: "allow-rm-rf", Name: "allow rm -rf",
				Actions: []string{"shell:exec"}, Resources: []string{"*"}, Effect: types.EffectAllow, Priority: 9999,
			},
		},
	}

	ctx := context.Background()
	e := NewPolicyEngine(policy)

	tests := []struct {
		name   string
		action types.Action
	}{
		{"delete root", types.Action{Type: types.ActionFileDelete, Resource: "/"}},
		{"listen 80", types.Action{Type: types.ActionNetListen, Resource: ":80"}},
		{"rm -rf /", types.Action{Type: types.ActionShellExec, Resource: "rm -rf /"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.Evaluate(ctx, tt.action)
			if got.Effect != types.EffectDeny {
				t.Errorf("built-in should deny despite user policy; got %q (reason: %s)", got.Effect, got.Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MatchGlob unit tests
// ---------------------------------------------------------------------------

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		str     string
		pattern string
		want    bool
	}{
		// Literal
		{"foo", "foo", true},
		{"foo", "bar", false},

		// Single star
		{"file:read", "file:*", true},
		{"file:write", "file:*", true},
		{"net:http", "file:*", false},
		{"abc", "*", true},
		{"", "*", true},
		{"/tmp/foo", "/tmp/*", true},
		{"/tmp/foo/bar", "/tmp/*", false}, // * does not cross /

		// Double star
		{"/tmp/foo/bar", "/tmp/**", true},
		{"/tmp/a/b/c/d", "/tmp/**", true},
		{"/tmp", "/tmp/**", false}, // ** needs at least the separator
		{"foo/bar/baz.go", "**/*.go", true},
		{"baz.go", "**/*.go", true},

		// Question mark
		{"ab", "a?", true},
		{"a/", "a?", false}, // ? does not match /
		{"abc", "a?c", true},
		{"aXc", "a?c", true},

		// Domain patterns
		{"api.github.com", "*.github.com", true},
		{"github.com", "*.github.com", false}, // * must match at least one char segment
		{"evil.github.com.attacker.com", "*.github.com", false},

		// Mixed
		{"foo/bar/baz", "foo/**/baz", true},
		{"foo/baz", "foo/**/baz", true},
		{"foo/a/b/baz", "foo/**/baz", true},

		// Wildcard-all shortcuts
		{"anything", "*", true},
		{"/a/b/c", "**", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"→"+tt.str, func(t *testing.T) {
			got := MatchGlob(tt.str, tt.pattern)
			if got != tt.want {
				t.Errorf("MatchGlob(%q, %q) = %v, want %v", tt.str, tt.pattern, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parser — YAML round-trip
// ---------------------------------------------------------------------------

func TestLoadFromBytes(t *testing.T) {
	yaml := []byte(`
name: test-policy
version: "1.0"
description: "A test policy"
default_effect: deny
rules:
  - id: r1
    name: "Allow tmp"
    actions: ["file:read"]
    resources: ["/tmp/**"]
    effect: allow
    priority: 10
`)

	policy, err := LoadFromBytes(yaml)
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	if policy.Name != "test-policy" {
		t.Errorf("name = %q, want test-policy", policy.Name)
	}
	if policy.DefaultEffect != types.EffectDeny {
		t.Errorf("default_effect = %q, want deny", policy.DefaultEffect)
	}
	if len(policy.Rules) != 1 {
		t.Fatalf("rules count = %d, want 1", len(policy.Rules))
	}
	r := policy.Rules[0]
	if r.ID != "r1" || r.Effect != types.EffectAllow || r.Priority != 10 {
		t.Errorf("rule mismatch: %+v", r)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Write a temporary policy file.
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := []byte(`
name: file-test
version: "1.0"
default_effect: allow
rules: []
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	policy, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if policy.Name != "file-test" {
		t.Errorf("name = %q, want file-test", policy.Name)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// ---------------------------------------------------------------------------
// Parser — validation errors
// ---------------------------------------------------------------------------

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			"missing name",
			`version: "1.0"
default_effect: deny
rules: []`,
			"policy name is required",
		},
		{
			"invalid default effect",
			`name: bad
default_effect: maybe
rules: []`,
			"invalid default effect",
		},
		{
			"missing rule ID",
			`name: ok
rules:
  - name: "No ID"
    actions: ["file:read"]
    effect: allow`,
			"rule ID is required",
		},
		{
			"duplicate rule ID",
			`name: ok
rules:
  - id: dup
    name: "First"
    actions: ["file:read"]
    effect: allow
  - id: dup
    name: "Second"
    actions: ["file:write"]
    effect: deny`,
			"duplicate rule ID",
		},
		{
			"invalid rule effect",
			`name: ok
rules:
  - id: r1
    name: "Bad effect"
    actions: ["file:read"]
    effect: maybe`,
			"invalid effect",
		},
		{
			"no actions",
			`name: ok
rules:
  - id: r1
    name: "No actions"
    actions: []
    effect: allow`,
			"must have at least one action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFromBytes([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !containsSubstring(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err, tt.wantErr)
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// LoadPolicy / GetPolicy
// ---------------------------------------------------------------------------

func TestLoadAndGetPolicy(t *testing.T) {
	e := NewPolicyEngine(denyAllPolicy())
	if e.GetPolicy().Name != "deny-all" {
		t.Fatal("initial policy mismatch")
	}

	newPolicy := allowAllPolicy()
	if err := e.LoadPolicy(newPolicy); err != nil {
		t.Fatalf("LoadPolicy: %v", err)
	}
	if e.GetPolicy().Name != "allow-all" {
		t.Errorf("policy not updated, got %q", e.GetPolicy().Name)
	}
}

// ---------------------------------------------------------------------------
// Default effect fallback when empty
// ---------------------------------------------------------------------------

func TestEvaluate_EmptyDefaultEffect(t *testing.T) {
	policy := types.Policy{Name: "no-default"}
	e := NewPolicyEngine(policy)
	got := e.Evaluate(context.Background(), types.Action{Type: types.ActionFileRead, Resource: "/x"})
	if got.Effect != types.EffectDeny {
		t.Errorf("empty default should be deny, got %q", got.Effect)
	}
}

// ---------------------------------------------------------------------------
// Rule with no resources matches any resource
// ---------------------------------------------------------------------------

func TestEvaluate_NoResourcesMatchesAny(t *testing.T) {
	policy := types.Policy{
		Name:          "no-resources",
		DefaultEffect: types.EffectDeny,
		Rules: []types.Rule{{
			ID: "r1", Name: "deny all shell",
			Actions: []string{"shell:exec"}, Effect: types.EffectDeny, Priority: 10,
		}},
	}
	e := NewPolicyEngine(policy)
	got := e.Evaluate(context.Background(), types.Action{Type: types.ActionShellExec, Resource: "anything"})
	if got.Effect != types.EffectDeny {
		t.Errorf("expected deny, got %q", got.Effect)
	}
}

// ---------------------------------------------------------------------------
// Thread safety — concurrent Evaluate + LoadPolicy
// ---------------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	e := NewPolicyEngine(denyAllPolicy())
	ctx := context.Background()

	var wg sync.WaitGroup
	const goroutines = 50

	// Half goroutines evaluate, half swap policies.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					e.Evaluate(ctx, types.Action{Type: types.ActionFileRead, Resource: "/tmp/x"})
				}
			}()
		} else {
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = e.LoadPolicy(allowAllPolicy())
					_ = e.GetPolicy()
					_ = e.LoadPolicy(denyAllPolicy())
				}
			}()
		}
	}

	wg.Wait()
	// If we reach here without a race condition panic, the test passes.
}

// ---------------------------------------------------------------------------
// Parse real config files
// ---------------------------------------------------------------------------

func TestParseDefaultPolicy(t *testing.T) {
	policy, err := LoadFromFile("../../configs/default-policy.yaml")
	if err != nil {
		t.Fatalf("failed to load default policy: %v", err)
	}
	if policy.Name != "default" {
		t.Errorf("name = %q, want default", policy.Name)
	}
	if len(policy.Rules) < 3 {
		t.Errorf("expected at least 3 rules, got %d", len(policy.Rules))
	}
}

func TestParseExamplePolicies(t *testing.T) {
	examples := []string{"strict", "permissive", "coding-agent"}
	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "configs", "examples", name+".yaml")
			policy, err := LoadFromFile(path)
			if err != nil {
				t.Fatalf("failed to load %s: %v", name, err)
			}
			if policy.Name != name {
				t.Errorf("name = %q, want %q", policy.Name, name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: parse YAML then evaluate
// ---------------------------------------------------------------------------

func TestIntegration_DefaultPolicyEvaluation(t *testing.T) {
	policy, err := LoadFromFile("../../configs/default-policy.yaml")
	if err != nil {
		t.Fatal(err)
	}

	e := NewPolicyEngine(*policy)
	ctx := context.Background()

	tests := []struct {
		name   string
		action types.Action
		want   types.Effect
	}{
		{
			"allow read /tmp file",
			types.Action{Type: types.ActionFileRead, Resource: "/tmp/data.txt"},
			types.EffectAllow,
		},
		{
			"allow write /sandbox file",
			types.Action{Type: types.ActionFileWrite, Resource: "/sandbox/output.json"},
			types.EffectAllow,
		},
		{
			"deny read /etc/passwd",
			types.Action{Type: types.ActionFileRead, Resource: "/etc/passwd"},
			types.EffectDeny,
		},
		{
			"allow http to github",
			types.Action{Type: types.ActionNetHTTP, Resource: "api.github.com"},
			types.EffectAllow,
		},
		{
			"deny http to unknown",
			types.Action{Type: types.ActionNetHTTP, Resource: "evil.com"},
			types.EffectDeny,
		},
		{
			"allow proc ls",
			types.Action{Type: types.ActionProcExec, Resource: "ls"},
			types.EffectAllow,
		},
		{
			"deny proc rm",
			types.Action{Type: types.ActionProcExec, Resource: "rm"},
			types.EffectDeny,
		},
		{
			"deny shell exec",
			types.Action{Type: types.ActionShellExec, Resource: "bash"},
			types.EffectDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.Evaluate(ctx, tt.action)
			if got.Effect != tt.want {
				t.Errorf("got %q, want %q (reason: %s)", got.Effect, tt.want, got.Reason)
			}
		})
	}
}
