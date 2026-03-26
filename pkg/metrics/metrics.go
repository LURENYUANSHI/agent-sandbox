package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var once sync.Once

var (
	SandboxesActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agentsandbox_sandboxes_active",
		Help: "Number of currently active sandboxes",
	})
	ActionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "agentsandbox_actions_total",
		Help: "Total actions executed",
	}, []string{"action_type", "effect"})
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

func Init() {
	once.Do(func() {
		prometheus.MustRegister(
			SandboxesActive,
			ActionsTotal,
			ActionDuration,
			PolicyEvaluations,
			APIRequests,
			APILatency,
		)
	})
}
