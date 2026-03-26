You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code in pkg/api/auth.go and pkg/api/token.go.

## Your Task: Add Role-Based Access Control (RBAC)

### 1. pkg/api/rbac.go
Define roles and permissions:
```go
type Role string
const (
    RoleAdmin    Role = "admin"    // full access
    RoleOperator Role = "operator" // create/manage sandboxes, view traces
    RoleViewer   Role = "viewer"   // read-only access
)

type Permission string
const (
    PermSandboxCreate  Permission = "sandbox:create"
    PermSandboxManage  Permission = "sandbox:manage"  // start, stop, destroy
    PermSandboxExec    Permission = "sandbox:exec"
    PermTraceView      Permission = "trace:view"
    PermTraceReplay    Permission = "trace:replay"
    PermPolicyManage   Permission = "policy:manage"
    PermAuditView      Permission = "audit:view"
    PermUserManage     Permission = "user:manage"     // admin only
)
```

- Map each Role to its allowed Permissions
- `HasPermission(role Role, perm Permission) bool`
- `RequirePermission(perm Permission) gin.HandlerFunc` — middleware that checks the JWT role claim

### 2. Update pkg/api/token.go
- Add `Role` field to JWT claims
- Update `GenerateToken` to accept role parameter

### 3. Update pkg/api/auth.go
- Extract role from JWT claims and set in gin context
- Add helper: `GetUserRole(c *gin.Context) Role`

### 4. Update pkg/api/server.go
Apply permission middleware to routes:
- POST sandboxes → PermSandboxCreate (admin, operator)
- POST exec → PermSandboxExec (admin, operator)
- DELETE sandbox → PermSandboxManage (admin, operator)
- GET traces → PermTraceView (all roles)
- GET audit → PermAuditView (admin only)
- POST auth/token → PermUserManage (admin only) in production

### 5. pkg/api/rbac_test.go
- Test each role has correct permissions
- Test middleware blocks unauthorized access (viewer can't create sandbox)
- Test middleware allows authorized access
- Test admin has all permissions

### Verification:
1. `go build ./...`
2. `go test ./pkg/api/... -v -count=1`
3. Commit: `feat(api): add role-based access control with admin/operator/viewer roles`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
