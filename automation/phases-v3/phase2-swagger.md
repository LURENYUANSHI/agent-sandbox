You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

Read existing code in pkg/api/ to understand all endpoints.

## Your Task: Generate OpenAPI/Swagger documentation

### 1. Install swag
```bash
go install github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
```

### 2. Add Swagger annotations to pkg/api/handlers.go
Add `// @Summary`, `// @Description`, `// @Tags`, `// @Accept`, `// @Produce`, `// @Param`, `// @Success`, `// @Failure`, `// @Router` comments above each handler function.

Example:
```go
// @Summary Create a new sandbox
// @Description Create a new sandbox with the given configuration
// @Tags sandboxes
// @Accept json
// @Produce json
// @Param body body CreateSandboxRequest true "Sandbox configuration"
// @Success 201 {object} SandboxResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/sandboxes [post]
```

Do this for ALL endpoints (sandboxes CRUD, exec, traces, replay, dashboard, audit, health, auth).

### 3. Add main annotation to cmd/server/main.go
```go
// @title AgentSandbox API
// @version 0.3.0
// @description AI Agent Security Sandbox - Runtime isolation, policy enforcement, trace recording & replay
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
```

### 4. Generate docs
```bash
swag init -g cmd/server/main.go -o docs/swagger
```

### 5. Register Swagger UI route in pkg/api/server.go
```go
import swaggerFiles "github.com/swaggo/files"
import ginSwagger "github.com/swaggo/gin-swagger"
// ...
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

### 6. If swag tool is not available, manually create docs/swagger/swagger.json
Write a valid OpenAPI 3.0 spec JSON file covering all endpoints. Then serve it statically from the API server at GET /api/v1/docs.

### Verification:
1. `go build ./...`
2. `go test ./... -count=1`
3. Commit: `docs(api): add OpenAPI/Swagger documentation for all API endpoints`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
