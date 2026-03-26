package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Role represents a user's access level.
type Role string

const (
	RoleAdmin    Role = "admin"    // full access
	RoleOperator Role = "operator" // create/manage sandboxes, view traces
	RoleViewer   Role = "viewer"   // read-only access
)

// Permission represents a specific action that can be authorized.
type Permission string

const (
	PermSandboxCreate Permission = "sandbox:create"
	PermSandboxManage Permission = "sandbox:manage" // start, stop, destroy
	PermSandboxExec   Permission = "sandbox:exec"
	PermTraceView     Permission = "trace:view"
	PermTraceReplay   Permission = "trace:replay"
	PermPolicyManage  Permission = "policy:manage"
	PermAuditView     Permission = "audit:view"
	PermUserManage    Permission = "user:manage" // admin only
)

// rolePermissions maps each role to the set of permissions it grants.
var rolePermissions = map[Role]map[Permission]bool{
	RoleAdmin: {
		PermSandboxCreate: true,
		PermSandboxManage: true,
		PermSandboxExec:   true,
		PermTraceView:     true,
		PermTraceReplay:   true,
		PermPolicyManage:  true,
		PermAuditView:     true,
		PermUserManage:    true,
	},
	RoleOperator: {
		PermSandboxCreate: true,
		PermSandboxManage: true,
		PermSandboxExec:   true,
		PermTraceView:     true,
		PermTraceReplay:   true,
		PermPolicyManage:  true,
	},
	RoleViewer: {
		PermTraceView:   true,
		PermTraceReplay: true,
	},
}

// ValidRole returns true if the given string is a recognized role.
func ValidRole(r string) bool {
	switch Role(r) {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

// HasPermission returns true if the given role grants the specified permission.
func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}

// GetUserRole extracts the user's role from the gin context.
// Returns RoleViewer if no role is set (safe default).
func GetUserRole(c *gin.Context) Role {
	v, exists := c.Get("user_role")
	if !exists {
		return RoleViewer
	}
	r, ok := v.(string)
	if !ok {
		return RoleViewer
	}
	if !ValidRole(r) {
		return RoleViewer
	}
	return Role(r)
}

// RequirePermission returns middleware that checks whether the authenticated
// user's role grants the specified permission. If auth is disabled (dev mode),
// the check is skipped.
func RequirePermission(perm Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If no user_role is set, auth middleware was a no-op (dev mode) — allow.
		if _, exists := c.Get("user_role"); !exists {
			c.Next()
			return
		}

		role := GetUserRole(c)
		if !HasPermission(role, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			return
		}
		c.Next()
	}
}
