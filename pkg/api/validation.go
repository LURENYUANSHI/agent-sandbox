package api

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/LURENYUANSHI/agent-sandbox/pkg/types"
)

// ValidationError represents a single field validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// namePattern matches alphanumeric characters, hyphens, and underscores.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// knownActionTypes lists all valid action types accepted by the executor.
var knownActionTypes = map[types.ActionType]bool{
	types.ActionFileRead:   true,
	types.ActionFileWrite:  true,
	types.ActionFileDelete: true,
	types.ActionNetConnect: true,
	types.ActionNetListen:  true,
	types.ActionNetHTTP:    true,
	types.ActionProcExec:   true,
	types.ActionProcKill:   true,
	types.ActionShellExec:  true,
	// Dot-separated variants
	types.ActionTypeFileRead:   true,
	types.ActionTypeFileWrite:  true,
	types.ActionTypeFileDelete: true,
	types.ActionTypeNetHTTP:    true,
	types.ActionTypeNetConnect: true,
	types.ActionTypeProcess:    true,
	types.ActionTypeShell:      true,
}

// ValidateCreateSandbox validates a CreateSandboxRequest.
func ValidateCreateSandbox(req CreateSandboxRequest) *ValidationErrors {
	var errs []ValidationError

	if req.Name != "" {
		if len(req.Name) > 128 {
			errs = append(errs, ValidationError{
				Field:   "name",
				Message: "name must be at most 128 characters",
			})
		} else if !namePattern.MatchString(req.Name) {
			errs = append(errs, ValidationError{
				Field:   "name",
				Message: "name must be alphanumeric, hyphens, or underscores, and start with an alphanumeric character",
			})
		}
	}

	if req.PolicyFile != "" {
		if _, err := os.Stat(req.PolicyFile); os.IsNotExist(err) {
			errs = append(errs, ValidationError{
				Field:   "policy_file",
				Message: fmt.Sprintf("policy file does not exist: %s", req.PolicyFile),
			})
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return &ValidationErrors{Errors: errs}
}

// ValidateExecAction validates an ExecActionRequest.
func ValidateExecAction(req ExecActionRequest) *ValidationErrors {
	var errs []ValidationError

	if req.Type == "" {
		errs = append(errs, ValidationError{
			Field:   "type",
			Message: "action type is required",
		})
	} else if !knownActionTypes[types.ActionType(req.Type)] {
		errs = append(errs, ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("unknown action type: %s", req.Type),
		})
	}

	if len(req.Params) == 0 {
		errs = append(errs, ValidationError{
			Field:   "params",
			Message: "params must not be empty",
		})
	}

	// Check for null bytes in resource-related param values
	for key, val := range req.Params {
		if strings.ContainsRune(val, '\x00') {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("params.%s", key),
				Message: "value must not contain null bytes",
			})
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return &ValidationErrors{Errors: errs}
}

// ValidatePolicy validates a policy structure.
func ValidatePolicy(p types.Policy) *ValidationErrors {
	var errs []ValidationError

	// Check default effect
	validEffects := map[types.Effect]bool{
		types.EffectAllow: true,
		types.EffectDeny:  true,
		types.EffectAudit: true,
	}
	if p.DefaultEffect != "" && !validEffects[p.DefaultEffect] {
		errs = append(errs, ValidationError{
			Field:   "default_effect",
			Message: fmt.Sprintf("invalid default effect: %s", p.DefaultEffect),
		})
	}

	// Check rule IDs are unique
	seenIDs := make(map[string]bool)
	for i, rule := range p.Rules {
		if rule.ID != "" {
			if seenIDs[rule.ID] {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("rules[%d].id", i),
					Message: fmt.Sprintf("duplicate rule ID: %s", rule.ID),
				})
			}
			seenIDs[rule.ID] = true
		}

		// Validate effect
		if rule.Effect != "" && !validEffects[rule.Effect] {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("rules[%d].effect", i),
				Message: fmt.Sprintf("invalid effect: %s", rule.Effect),
			})
		}

		// Validate action patterns (must not be empty strings)
		for j, action := range rule.Actions {
			if strings.TrimSpace(action) == "" {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("rules[%d].actions[%d]", i, j),
					Message: "action pattern must not be empty",
				})
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return &ValidationErrors{Errors: errs}
}

// respondValidationError writes a 400 response with structured validation errors.
func respondValidationError(c *gin.Context, verrs *ValidationErrors) {
	c.JSON(http.StatusBadRequest, verrs)
}
