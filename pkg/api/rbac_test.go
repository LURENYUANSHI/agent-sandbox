package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const rbacTestSecret = "test-secret-rbac"

// --- HasPermission ---

func TestAdminHasAllPermissions(t *testing.T) {
	allPerms := []Permission{
		PermSandboxCreate, PermSandboxManage, PermSandboxExec,
		PermTraceView, PermTraceReplay,
		PermPolicyManage, PermAuditView, PermUserManage,
	}
	for _, perm := range allPerms {
		if !HasPermission(RoleAdmin, perm) {
			t.Errorf("admin should have permission %s", perm)
		}
	}
}

func TestOperatorPermissions(t *testing.T) {
	allowed := []Permission{
		PermSandboxCreate, PermSandboxManage, PermSandboxExec,
		PermTraceView, PermTraceReplay, PermPolicyManage,
	}
	denied := []Permission{PermAuditView, PermUserManage}

	for _, perm := range allowed {
		if !HasPermission(RoleOperator, perm) {
			t.Errorf("operator should have permission %s", perm)
		}
	}
	for _, perm := range denied {
		if HasPermission(RoleOperator, perm) {
			t.Errorf("operator should not have permission %s", perm)
		}
	}
}

func TestViewerPermissions(t *testing.T) {
	allowed := []Permission{PermTraceView, PermTraceReplay}
	denied := []Permission{
		PermSandboxCreate, PermSandboxManage, PermSandboxExec,
		PermPolicyManage, PermAuditView, PermUserManage,
	}

	for _, perm := range allowed {
		if !HasPermission(RoleViewer, perm) {
			t.Errorf("viewer should have permission %s", perm)
		}
	}
	for _, perm := range denied {
		if HasPermission(RoleViewer, perm) {
			t.Errorf("viewer should not have permission %s", perm)
		}
	}
}

func TestUnknownRoleHasNoPermissions(t *testing.T) {
	if HasPermission(Role("unknown"), PermTraceView) {
		t.Error("unknown role should have no permissions")
	}
}

// --- ValidRole ---

func TestValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"admin", true},
		{"operator", true},
		{"viewer", true},
		{"", false},
		{"superadmin", false},
	}
	for _, tt := range tests {
		if got := ValidRole(tt.role); got != tt.want {
			t.Errorf("ValidRole(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

// --- GetUserRole ---

func TestGetUserRoleFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setup    func(c *gin.Context)
		expected Role
	}{
		{"admin", func(c *gin.Context) { c.Set("user_role", "admin") }, RoleAdmin},
		{"operator", func(c *gin.Context) { c.Set("user_role", "operator") }, RoleOperator},
		{"viewer", func(c *gin.Context) { c.Set("user_role", "viewer") }, RoleViewer},
		{"missing defaults to viewer", func(c *gin.Context) {}, RoleViewer},
		{"invalid defaults to viewer", func(c *gin.Context) { c.Set("user_role", "bogus") }, RoleViewer},
		{"wrong type defaults to viewer", func(c *gin.Context) { c.Set("user_role", 123) }, RoleViewer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)
			if got := GetUserRole(c); got != tt.expected {
				t.Errorf("GetUserRole() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// --- RequirePermission middleware ---

func newRBACServer() *Server {
	return NewServer(ServerConfig{
		Port:        0,
		DevMode:     true,
		AuthEnabled: true,
		AuthSecret:  rbacTestSecret,
	})
}

func rbacToken(userID, role string) string {
	tok, err := GenerateToken(rbacTestSecret, userID, role, time.Hour)
	if err != nil {
		panic(err)
	}
	return tok
}

func TestViewerCannotCreateSandbox(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("viewer-user", "viewer")

	w := doAuthRequest(s, "POST", "/api/v1/sandboxes", token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer creating sandbox, got %d: %s", w.Code, w.Body.String())
	}
}

func TestViewerCanListSandboxes(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("viewer-user", "viewer")

	w := doAuthRequest(s, "GET", "/api/v1/sandboxes", token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for viewer listing sandboxes, got %d: %s", w.Code, w.Body.String())
	}
}

func TestViewerCannotAccessAudit(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("viewer-user", "viewer")

	w := doAuthRequest(s, "GET", "/api/v1/audit", token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer accessing audit, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOperatorCanCreateSandbox(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("op-user", "operator")

	w := doAuthRequest(s, "POST", "/api/v1/sandboxes", token)
	// 400 is expected because the body is empty, not 403
	if w.Code == http.StatusForbidden {
		t.Fatal("operator should be allowed to create sandboxes")
	}
}

func TestOperatorCannotAccessAudit(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("op-user", "operator")

	w := doAuthRequest(s, "GET", "/api/v1/audit", token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for operator accessing audit, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOperatorCannotManageUsers(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("op-user", "operator")

	w := doAuthRequest(s, "POST", "/api/v1/auth/token", token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for operator generating tokens, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminCanAccessAudit(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("admin-user", "admin")

	w := doAuthRequest(s, "GET", "/api/v1/audit", token)
	// Should not be 403 — might be 503 if audit logger is not configured, but that's ok
	if w.Code == http.StatusForbidden {
		t.Fatal("admin should be allowed to access audit log")
	}
}

func TestAdminCanGenerateTokens(t *testing.T) {
	s := newRBACServer()
	token := rbacToken("admin-user", "admin")

	w := doAuthRequest(s, "POST", "/api/v1/auth/token", token)
	// Should not be 403 — might be 400 because body is empty, but not forbidden
	if w.Code == http.StatusForbidden {
		t.Fatal("admin should be allowed to generate tokens")
	}
}

func TestDevModeSkipsPermissionCheck(t *testing.T) {
	s := newTestServer() // DevMode=true, AuthEnabled=false
	w := doRequest(s, "POST", "/api/v1/sandboxes", map[string]string{"name": "dev-test"})
	if w.Code == http.StatusForbidden {
		t.Fatal("dev mode should skip permission checks")
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHealthBypassesAuth(t *testing.T) {
	s := newRBACServer()
	// No token at all
	w := doAuthRequest(s, "GET", "/api/v1/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("health should bypass auth, got %d", w.Code)
	}
}
