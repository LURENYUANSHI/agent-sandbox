package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// --- ValidateCreateSandbox ---

func TestValidateCreateSandbox_ValidName(t *testing.T) {
	tests := []struct {
		name string
		req  CreateSandboxRequest
	}{
		{"simple", CreateSandboxRequest{Name: "my-sandbox"}},
		{"with-underscore", CreateSandboxRequest{Name: "my_sandbox_01"}},
		{"empty-name", CreateSandboxRequest{Name: ""}},
		{"single-char", CreateSandboxRequest{Name: "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errs := ValidateCreateSandbox(tt.req); errs != nil {
				t.Errorf("expected no errors, got %v", errs.Errors)
			}
		})
	}
}

func TestValidateCreateSandbox_InvalidName(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateSandboxRequest
		wantMsg string
	}{
		{
			"too-long",
			CreateSandboxRequest{Name: strings.Repeat("a", 129)},
			"at most 128",
		},
		{
			"special-chars",
			CreateSandboxRequest{Name: "my sandbox!"},
			"alphanumeric",
		},
		{
			"starts-with-dash",
			CreateSandboxRequest{Name: "-sandbox"},
			"alphanumeric",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateCreateSandbox(tt.req)
			if errs == nil {
				t.Fatal("expected validation errors")
			}
			found := false
			for _, e := range errs.Errors {
				if e.Field == "name" && strings.Contains(e.Message, tt.wantMsg) {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error containing %q for field 'name', got %v", tt.wantMsg, errs.Errors)
			}
		})
	}
}

func TestValidateCreateSandbox_PolicyFileNotExists(t *testing.T) {
	errs := ValidateCreateSandbox(CreateSandboxRequest{
		Name:       "test",
		PolicyFile: "/nonexistent/path/policy.yaml",
	})
	if errs == nil {
		t.Fatal("expected validation errors for missing policy file")
	}
	if errs.Errors[0].Field != "policy_file" {
		t.Errorf("expected field policy_file, got %s", errs.Errors[0].Field)
	}
}

func TestValidateCreateSandbox_MaxLengthName(t *testing.T) {
	// Exactly 128 chars should be valid
	errs := ValidateCreateSandbox(CreateSandboxRequest{Name: strings.Repeat("a", 128)})
	if errs != nil {
		t.Errorf("128-char name should be valid, got %v", errs.Errors)
	}
}

// --- ValidateExecAction ---

func TestValidateExecAction_Valid(t *testing.T) {
	tests := []struct {
		name string
		req  ExecActionRequest
	}{
		{
			"file-write-colon",
			ExecActionRequest{Type: "file:write", Params: map[string]string{"path": "/tmp/test"}},
		},
		{
			"file-write-dot",
			ExecActionRequest{Type: "file.write", Params: map[string]string{"path": "/tmp/test"}},
		},
		{
			"shell-exec",
			ExecActionRequest{Type: "shell:exec", Params: map[string]string{"command": "ls"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errs := ValidateExecAction(tt.req); errs != nil {
				t.Errorf("expected no errors, got %v", errs.Errors)
			}
		})
	}
}

func TestValidateExecAction_MissingType(t *testing.T) {
	errs := ValidateExecAction(ExecActionRequest{
		Params: map[string]string{"path": "/tmp"},
	})
	if errs == nil {
		t.Fatal("expected validation errors")
	}
	if errs.Errors[0].Field != "type" {
		t.Errorf("expected field 'type', got %s", errs.Errors[0].Field)
	}
}

func TestValidateExecAction_UnknownType(t *testing.T) {
	errs := ValidateExecAction(ExecActionRequest{
		Type:   "unknown.action",
		Params: map[string]string{"path": "/tmp"},
	})
	if errs == nil {
		t.Fatal("expected validation errors for unknown type")
	}
	found := false
	for _, e := range errs.Errors {
		if e.Field == "type" && strings.Contains(e.Message, "unknown") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown type error, got %v", errs.Errors)
	}
}

func TestValidateExecAction_EmptyParams(t *testing.T) {
	errs := ValidateExecAction(ExecActionRequest{
		Type: "file:read",
	})
	if errs == nil {
		t.Fatal("expected validation errors for empty params")
	}
	found := false
	for _, e := range errs.Errors {
		if e.Field == "params" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected params error, got %v", errs.Errors)
	}
}

func TestValidateExecAction_NullByteInParams(t *testing.T) {
	errs := ValidateExecAction(ExecActionRequest{
		Type:   "file:read",
		Params: map[string]string{"path": "/tmp/test\x00evil"},
	})
	if errs == nil {
		t.Fatal("expected validation errors for null byte")
	}
	found := false
	for _, e := range errs.Errors {
		if strings.HasPrefix(e.Field, "params.") && strings.Contains(e.Message, "null bytes") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected null byte error, got %v", errs.Errors)
	}
}

func TestValidateExecAction_MultipleErrors(t *testing.T) {
	errs := ValidateExecAction(ExecActionRequest{})
	if errs == nil {
		t.Fatal("expected validation errors")
	}
	if len(errs.Errors) < 2 {
		t.Errorf("expected at least 2 errors (type + params), got %d", len(errs.Errors))
	}
}

// --- ValidatePolicy ---

func TestValidatePolicy_Valid(t *testing.T) {
	p := types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{ID: "r1", Effect: types.EffectDeny, Actions: []string{"file:*"}},
			{ID: "r2", Effect: types.EffectAllow, Actions: []string{"net:*"}},
		},
	}
	if errs := ValidatePolicy(p); errs != nil {
		t.Errorf("expected no errors, got %v", errs.Errors)
	}
}

func TestValidatePolicy_DuplicateRuleIDs(t *testing.T) {
	p := types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{ID: "r1", Effect: types.EffectDeny},
			{ID: "r1", Effect: types.EffectAllow},
		},
	}
	errs := ValidatePolicy(p)
	if errs == nil {
		t.Fatal("expected validation errors for duplicate IDs")
	}
	found := false
	for _, e := range errs.Errors {
		if strings.Contains(e.Message, "duplicate") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate ID error, got %v", errs.Errors)
	}
}

func TestValidatePolicy_InvalidEffect(t *testing.T) {
	p := types.Policy{
		Name:          "test",
		DefaultEffect: types.Effect("invalid"),
	}
	errs := ValidatePolicy(p)
	if errs == nil {
		t.Fatal("expected validation errors for invalid effect")
	}
	if errs.Errors[0].Field != "default_effect" {
		t.Errorf("expected field default_effect, got %s", errs.Errors[0].Field)
	}
}

func TestValidatePolicy_InvalidRuleEffect(t *testing.T) {
	p := types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{ID: "r1", Effect: types.Effect("bad")},
		},
	}
	errs := ValidatePolicy(p)
	if errs == nil {
		t.Fatal("expected validation errors for invalid rule effect")
	}
	found := false
	for _, e := range errs.Errors {
		if strings.Contains(e.Field, "rules[0].effect") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected rule effect error, got %v", errs.Errors)
	}
}

func TestValidatePolicy_EmptyActionPattern(t *testing.T) {
	p := types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectAllow,
		Rules: []types.Rule{
			{ID: "r1", Effect: types.EffectAllow, Actions: []string{"file:*", "  "}},
		},
	}
	errs := ValidatePolicy(p)
	if errs == nil {
		t.Fatal("expected validation errors for empty action pattern")
	}
	found := false
	for _, e := range errs.Errors {
		if strings.Contains(e.Message, "action pattern") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected empty action pattern error, got %v", errs.Errors)
	}
}

func TestValidatePolicy_EmptyRules(t *testing.T) {
	p := types.Policy{
		Name:          "test",
		DefaultEffect: types.EffectAllow,
		Rules:         []types.Rule{},
	}
	if errs := ValidatePolicy(p); errs != nil {
		t.Errorf("empty rules should be valid, got %v", errs.Errors)
	}
}

// --- Handler integration: validation returns 400 ---

func TestCreateSandboxValidationError(t *testing.T) {
	s := newTestServer()
	w := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{
		"name": "invalid name with spaces!",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	errors, ok := body["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Errorf("expected structured validation errors, got %v", body)
	}
}

func TestExecActionValidationError(t *testing.T) {
	s := newTestServer()
	createResp := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "test"})
	id := parseBody(createResp)["id"].(string)
	doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/start", nil)

	// Send request with unknown action type
	w := doRequest(s, "POST", "/api/v1/sandboxes/"+id+"/exec", map[string]interface{}{
		"type":   "bogus.action",
		"params": map[string]string{"key": "val"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
