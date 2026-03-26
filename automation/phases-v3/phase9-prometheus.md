You are working on the AgentSandbox project in /c/Users/Administrator/ai-sandbox on a feature branch.

## Your Task: Add Prometheus metrics endpoint

### 1. Install dependency
```
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

### 2. pkg/metrics/metrics.go
Define metrics:
```go
var (
    SandboxesActive = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "agentsandbox_sandboxes_active",
        Help: "Number of currently active sandboxes",
    })
    ActionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "agentsandbox_actions_total",
        Help: "Total actions executed",
    }, []string{"action_type", "effect"})  // effect: allowed, denied
    ActionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "agentsandbox_action_duration_seconds",
        Help: "Action execution duration",
        Buckets: prometheus.DefBuckets,
    }, []string{"action_type"})
    PolicyEvaluations = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "agentsandbox_policy_evaluations_total",
        Help: "Total policy evaluations",
    }, []string{"effect"})
    APIRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "agentsandbox_api_requests_total",
        Help: "Total API requests",
    }, []string{"method", "path", "status"})
    APILatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "agentsandbox_api_latency_seconds",
        Help: "API request latency",
        Buckets: prometheus.DefBuckets,
    }, []string{"method", "path"})
)

func Init() { // register all metrics }
```

### 3. pkg/api/metrics_middleware.go
Gin middleware that records API request metrics (method, path, status code, latency).

### 4. Update pkg/api/server.go
- Register metrics endpoint: `GET /metrics` using promhttp.Handler()
- Apply metrics middleware to all routes
- Call metrics.Init() on startup

### 5. Update pkg/sandbox/sandbox.go
Increment/decrement SandboxesActive on start/stop.
Record ActionsTotal and ActionDuration on each execute.

### 6. Update pkg/policy/engine.go
Increment PolicyEvaluations on each evaluation.

### 7. pkg/metrics/metrics_test.go
- Test that metrics are registered
- Test counters increment correctly
- Test histogram records values

### Verification:
1. `go build ./...`
2. `go test ./... -count=1`
3. Commit: `feat: add Prometheus metrics for sandboxes, actions, policy evaluations, and API`

DO NOT push to remote, DO NOT create PRs or issues. Only local commits.
